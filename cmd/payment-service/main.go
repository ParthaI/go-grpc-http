package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	paymentv1 "github.com/parthasarathi/go-grpc-http/gen/go/payment/v1"
	paymentHandler "github.com/parthasarathi/go-grpc-http/internal/payment"
	paymentevent "github.com/parthasarathi/go-grpc-http/internal/payment/event"
	"github.com/parthasarathi/go-grpc-http/internal/payment/repository"
	"github.com/parthasarathi/go-grpc-http/pkg/auth"
	"github.com/parthasarathi/go-grpc-http/pkg/database"
	"github.com/parthasarathi/go-grpc-http/pkg/health"
	"github.com/parthasarathi/go-grpc-http/pkg/interceptors"
	"github.com/parthasarathi/go-grpc-http/pkg/observability"
)

func main() {
	logger := observability.NewLogger("payment-service")

	port := envOrDefault("GRPC_PORT", "50054")
	dbURL := envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5436/paymentdb?sslmode=disable")
	natsURL := envOrDefault("NATS_URL", "nats://localhost:4222")
	userServiceAddr := envOrDefault("USER_SERVICE_ADDR", "localhost:50051")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewPostgres(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Auto-migrate
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS payments (
			id VARCHAR(36) PRIMARY KEY,
			order_id VARCHAR(36) NOT NULL,
			amount_cents BIGINT NOT NULL,
			currency VARCHAR(3) NOT NULL DEFAULT 'USD',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			method VARCHAR(50) NOT NULL DEFAULT 'card',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		logger.Error("failed to run migration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		logger.Error("failed to connect to nats", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("failed to get jetstream", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create PAYMENTS stream
	js.AddStream(&nats.StreamConfig{
		Name:     "PAYMENTS",
		Subjects: []string{"payments.*"},
	})

	repo := repository.NewRepository(pool)
	pub := paymentevent.NewPublisher(js)

	// Start event subscriber (listens to orders.created)
	sub := paymentevent.NewSubscriber(js, repo, pub, logger)
	go func() {
		if err := sub.Start(ctx); err != nil {
			logger.Error("subscriber error", slog.String("error", err.Error()))
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

	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentHandler.NewServer(repo))
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer(
		func(ctx context.Context) error { return pool.Ping(ctx) },
	))
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		logger.Info("payment-service listening", slog.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc serve error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down payment-service")
	cancel()
	grpcServer.GracefulStop()
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
