package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	userv1 "github.com/parthasarathi/go-grpc-http/gen/go/user/v1"
	userHandler "github.com/parthasarathi/go-grpc-http/internal/user"
	"github.com/parthasarathi/go-grpc-http/internal/user/repository"
	"github.com/parthasarathi/go-grpc-http/internal/user/service"
	"github.com/parthasarathi/go-grpc-http/pkg/auth"
	"github.com/parthasarathi/go-grpc-http/pkg/database"
	"github.com/parthasarathi/go-grpc-http/pkg/health"
	"github.com/parthasarathi/go-grpc-http/pkg/interceptors"
	"github.com/parthasarathi/go-grpc-http/pkg/observability"
)

func main() {
	logger := observability.NewLogger("user-service")

	port := envOrDefault("GRPC_PORT", "50051")
	dbURL := envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/userdb?sslmode=disable")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewPostgres(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Run migrations inline for development
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			first_name VARCHAR(100) NOT NULL DEFAULT '',
			last_name VARCHAR(100) NOT NULL DEFAULT '',
			auth_token TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_indexes WHERE indexname = 'idx_users_auth_token'
			) THEN
				CREATE UNIQUE INDEX idx_users_auth_token ON users(auth_token) WHERE auth_token != '';
			END IF;
		END $$`,
	}
	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			logger.Error("failed to run migration", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	jwtManager := auth.NewJWTManager(24 * time.Hour)
	tokenResolver := auth.NewDBTokenResolver(pool) // user-service: direct DB lookup
	repo := repository.NewPostgresRepository(pool)
	svc := service.NewUserService(repo, jwtManager)
	srv := userHandler.NewServer(svc)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(logger),
			interceptors.LoggingInterceptor(logger),
			auth.UnaryAuthInterceptor(jwtManager, tokenResolver),
		),
	)

	userv1.RegisterUserServiceServer(grpcServer, srv)
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
		logger.Info("user-service listening", slog.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc serve error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down user-service")
	grpcServer.GracefulStop()
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
