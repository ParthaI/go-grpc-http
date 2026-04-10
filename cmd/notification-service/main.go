package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	notifyevent "github.com/parthasarathi/go-grpc-http/internal/notification/event"
	"github.com/parthasarathi/go-grpc-http/internal/notification/repository"
	"github.com/parthasarathi/go-grpc-http/internal/notification/sender"
	"github.com/parthasarathi/go-grpc-http/pkg/database"
	"github.com/parthasarathi/go-grpc-http/pkg/messaging"
	"github.com/parthasarathi/go-grpc-http/pkg/observability"
)

func main() {
	logger := observability.NewLogger("notification-service")

	dbURL := envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5437/notificationdb?sslmode=disable")
	rabbitmqURL := envOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	// SMTP config (leave empty to use mock email logging)
	smtpHost := envOrDefault("SMTP_HOST", "")
	smtpPort := envOrDefault("SMTP_PORT", "587")
	smtpUser := envOrDefault("SMTP_USER", "")
	smtpPassword := envOrDefault("SMTP_PASSWORD", "")
	smtpFrom := envOrDefault("SMTP_FROM", "")
	notifyEmail := envOrDefault("NOTIFY_EMAIL", "") // recipient for all notifications

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	pool, err := database.NewPostgres(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Auto-migrate
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notification_log (
			id VARCHAR(36) PRIMARY KEY,
			event_type VARCHAR(100) NOT NULL,
			recipient VARCHAR(255) NOT NULL,
			channel VARCHAR(50) NOT NULL DEFAULT 'email',
			subject VARCHAR(500) NOT NULL,
			body TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		logger.Error("failed to run migration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// RabbitMQ (consume notifications from here)
	rmq, err := messaging.NewRabbitMQ(rabbitmqURL, logger)
	if err != nil {
		logger.Error("failed to connect to rabbitmq", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer rmq.Close()

	repo := repository.NewRepository(pool)
	emailSender := sender.NewEmailSender(smtpHost, smtpPort, smtpUser, smtpPassword, smtpFrom, logger)

	// Subscriber: consumes from RabbitMQ queue → sends notifications
	sub := notifyevent.NewSubscriber(rmq.Channel, repo, emailSender, notifyEmail, logger)
	go func() {
		logger.Info("notification-service started, consuming from RabbitMQ")
		if err := sub.Start(ctx); err != nil {
			logger.Error("subscriber error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down notification-service")
	cancel()
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
