package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	notifyevent "github.com/parthasarathi/go-grpc-http/internal/notification/event"
	"github.com/parthasarathi/go-grpc-http/pkg/messaging"
	"github.com/parthasarathi/go-grpc-http/pkg/observability"
)

func main() {
	logger := observability.NewLogger("bridge-service")

	natsURL := envOrDefault("NATS_URL", "nats://localhost:4222")
	rabbitmqURL := envOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// NATS (consume events from other services)
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

	// RabbitMQ (forward events here)
	rmq, err := messaging.NewRabbitMQ(rabbitmqURL, logger)
	if err != nil {
		logger.Error("failed to connect to rabbitmq", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer rmq.Close()

	// Bridge: NATS → RabbitMQ
	bridge := notifyevent.NewBridge(js, rmq.Channel, logger)
	go func() {
		logger.Info("bridge-service started, forwarding NATS → RabbitMQ")
		if err := bridge.Start(ctx); err != nil {
			logger.Error("bridge error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down bridge-service")
	cancel()
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
