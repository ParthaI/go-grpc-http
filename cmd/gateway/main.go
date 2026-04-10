package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/parthasarathi/go-grpc-http/internal/gateway"
	"github.com/parthasarathi/go-grpc-http/pkg/observability"
)

func main() {
	logger := observability.NewLogger("api-gateway")

	cfg := gateway.Config{
		HTTPPort:           envOrDefault("HTTP_PORT", "8080"),
		UserServiceAddr:    envOrDefault("USER_SERVICE_ADDR", "localhost:50051"),
		ProductServiceAddr: envOrDefault("PRODUCT_SERVICE_ADDR", "localhost:50052"),
		OrderServiceAddr:   envOrDefault("ORDER_SERVICE_ADDR", "localhost:50053"),
		PaymentServiceAddr: envOrDefault("PAYMENT_SERVICE_ADDR", "localhost:50054"),
	}

	ctx := context.Background()
	if err := gateway.Run(ctx, cfg, logger); err != nil {
		logger.Error("gateway failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
