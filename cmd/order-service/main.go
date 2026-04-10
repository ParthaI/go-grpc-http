package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	orderv1 "github.com/parthasarathi/go-grpc-http/gen/go/order/v1"
	orderHandler "github.com/parthasarathi/go-grpc-http/internal/order"
	"github.com/parthasarathi/go-grpc-http/internal/order/command"
	"github.com/parthasarathi/go-grpc-http/internal/order/event"
	"github.com/parthasarathi/go-grpc-http/internal/order/query"
	"github.com/parthasarathi/go-grpc-http/internal/order/repository"
	"github.com/parthasarathi/go-grpc-http/pkg/auth"
	"github.com/parthasarathi/go-grpc-http/pkg/database"
	"github.com/parthasarathi/go-grpc-http/pkg/health"
	"github.com/parthasarathi/go-grpc-http/pkg/interceptors"
	"github.com/parthasarathi/go-grpc-http/pkg/observability"
)

func main() {
	logger := observability.NewLogger("order-service")

	port := envOrDefault("GRPC_PORT", "50053")
	dbURL := envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5435/orderdb?sslmode=disable")
	redisAddr := envOrDefault("REDIS_ADDR", "localhost:6379")
	natsURL := envOrDefault("NATS_URL", "nats://localhost:4222")
	productAddr := envOrDefault("PRODUCT_SERVICE_ADDR", "localhost:50052")
	userServiceAddr := envOrDefault("USER_SERVICE_ADDR", "localhost:50051")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL (event store)
	pool, err := database.NewPostgres(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Auto-migrate event store table
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS event_store (
			id BIGSERIAL PRIMARY KEY,
			aggregate_id VARCHAR(36) NOT NULL,
			aggregate_type VARCHAR(50) NOT NULL DEFAULT 'order',
			event_type VARCHAR(100) NOT NULL,
			payload JSONB NOT NULL,
			version INT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(aggregate_id, version)
		)
	`); err != nil {
		logger.Error("failed to create event_store table", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Redis (read model)
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer rdb.Close()

	// NATS JetStream
	nc, err := nats.Connect(natsURL)
	if err != nil {
		logger.Error("failed to connect to nats", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("failed to get jetstream context", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create JetStream stream for orders
	js.AddStream(&nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"orders.*"},
	})

	// Wire CQRS components
	eventStore := event.NewStore(pool)
	publisher := event.NewPublisher(js)
	readRepo := repository.NewRedisReadRepository(rdb)

	cmdHandler, err := command.NewHandler(eventStore, publisher, productAddr)
	if err != nil {
		logger.Error("failed to create command handler", slog.String("error", err.Error()))
		os.Exit(1)
	}
	queryHandler := query.NewHandler(readRepo)

	// Start projector in background (builds Redis read model from events)
	projector := event.NewProjector(js, readRepo, logger)
	go func() {
		if err := projector.Start(ctx); err != nil {
			logger.Error("projector error", slog.String("error", err.Error()))
		}
	}()

	// Start payment event subscriber (marks orders as paid/cancelled)
	paymentSub := event.NewPaymentSubscriber(js, eventStore, publisher, readRepo, logger)
	go func() {
		if err := paymentSub.Start(ctx); err != nil {
			logger.Error("payment subscriber error", slog.String("error", err.Error()))
		}
	}()

	// gRPC server
	jwtManager := auth.NewJWTManager(0)
	tokenResolver, err := auth.NewGRPCTokenResolver(userServiceAddr)
	if err != nil {
		logger.Error("failed to create token resolver", slog.String("error", err.Error()))
		os.Exit(1)
	}
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(logger),
			interceptors.LoggingInterceptor(logger),
			auth.UnaryAuthInterceptor(jwtManager, tokenResolver),
		),
	)

	orderv1.RegisterOrderCommandServiceServer(grpcServer, orderHandler.NewCommandServer(cmdHandler))
	orderv1.RegisterOrderQueryServiceServer(grpcServer, orderHandler.NewQueryServer(queryHandler))
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer(
		func(ctx context.Context) error { return pool.Ping(ctx) },
		func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
	))
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		logger.Info("order-service listening", slog.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc serve error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down order-service")
	cancel()
	grpcServer.GracefulStop()
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
