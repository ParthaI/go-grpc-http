package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderv1 "github.com/parthasarathi/go-grpc-http/gen/go/order/v1"
	paymentv1 "github.com/parthasarathi/go-grpc-http/gen/go/payment/v1"
	productv1 "github.com/parthasarathi/go-grpc-http/gen/go/product/v1"
	userv1 "github.com/parthasarathi/go-grpc-http/gen/go/user/v1"
)

type Config struct {
	HTTPPort           string
	UserServiceAddr    string
	ProductServiceAddr string
	OrderServiceAddr   string
	PaymentServiceAddr string
}

func Run(ctx context.Context, cfg Config, logger *slog.Logger) error {
	mux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := userv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, cfg.UserServiceAddr, opts); err != nil {
		return fmt.Errorf("register user service: %w", err)
	}

	if err := productv1.RegisterProductServiceHandlerFromEndpoint(ctx, mux, cfg.ProductServiceAddr, opts); err != nil {
		return fmt.Errorf("register product service: %w", err)
	}

	if err := orderv1.RegisterOrderCommandServiceHandlerFromEndpoint(ctx, mux, cfg.OrderServiceAddr, opts); err != nil {
		return fmt.Errorf("register order command service: %w", err)
	}

	if err := orderv1.RegisterOrderQueryServiceHandlerFromEndpoint(ctx, mux, cfg.OrderServiceAddr, opts); err != nil {
		return fmt.Errorf("register order query service: %w", err)
	}

	if err := paymentv1.RegisterPaymentServiceHandlerFromEndpoint(ctx, mux, cfg.PaymentServiceAddr, opts); err != nil {
		return fmt.Errorf("register payment service: %w", err)
	}

	// Health check endpoint
	topMux := http.NewServeMux()
	topMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	topMux.Handle("/", mux)

	handler := corsMiddleware(topMux)

	logger.Info("gateway listening", slog.String("port", cfg.HTTPPort))
	if err := http.ListenAndServe(":"+cfg.HTTPPort, handler); err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
