package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	productv1 "github.com/parthasarathi/go-grpc-http/gen/go/product/v1"
	productHandler "github.com/parthasarathi/go-grpc-http/internal/product"
	"github.com/parthasarathi/go-grpc-http/internal/product/repository"
	"github.com/parthasarathi/go-grpc-http/internal/product/service"
	"github.com/parthasarathi/go-grpc-http/pkg/auth"
	"github.com/parthasarathi/go-grpc-http/pkg/database"
	"github.com/parthasarathi/go-grpc-http/pkg/health"
	"github.com/parthasarathi/go-grpc-http/pkg/interceptors"
	"github.com/parthasarathi/go-grpc-http/pkg/observability"
)

func main() {
	logger := observability.NewLogger("product-service")

	port := envOrDefault("GRPC_PORT", "50052")
	dbURL := envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/productdb?sslmode=disable")
	userServiceAddr := envOrDefault("USER_SERVICE_ADDR", "localhost:50051")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewPostgres(ctx, dbURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Auto-migrate for development
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			price_cents BIGINT NOT NULL,
			currency VARCHAR(3) NOT NULL DEFAULT 'USD',
			stock_quantity INT NOT NULL DEFAULT 0,
			sku VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`INSERT INTO products (id, name, description, price_cents, currency, stock_quantity, sku) VALUES
			('a0000001-0000-0000-0000-000000000001', 'Mechanical Keyboard', 'Cherry MX Blue switches, RGB backlight, full size', 12999, 'USD', 50, 'KB-MECH-001'),
			('a0000001-0000-0000-0000-000000000002', 'Wireless Mouse', 'Ergonomic design, 2.4GHz wireless, 6 buttons', 4999, 'USD', 120, 'MS-WIFI-001'),
			('a0000001-0000-0000-0000-000000000003', 'USB-C Hub', '7-in-1: HDMI, USB 3.0 x3, SD, microSD, PD charging', 3999, 'USD', 75, 'HUB-USB7-001'),
			('a0000001-0000-0000-0000-000000000004', '4K Monitor 27"', 'IPS panel, 144Hz, HDR400, USB-C input', 44999, 'USD', 20, 'MON-4K27-001'),
			('a0000001-0000-0000-0000-000000000005', 'Webcam HD 1080p', 'Auto-focus, dual microphone, privacy shutter', 5999, 'USD', 90, 'CAM-HD-001'),
			('a0000001-0000-0000-0000-000000000006', 'Bluetooth Speaker', 'Waterproof IPX7, 24hr battery, stereo pairing', 7999, 'USD', 60, 'SPK-BT-001'),
			('a0000001-0000-0000-0000-000000000007', 'Noise Cancelling Headphones', 'ANC, 30hr battery, foldable, multipoint', 24999, 'USD', 35, 'HP-ANC-001'),
			('a0000001-0000-0000-0000-000000000008', 'Laptop Stand', 'Aluminum, adjustable height, ventilated', 3499, 'USD', 100, 'STD-LAP-001'),
			('a0000001-0000-0000-0000-000000000009', 'Desk Lamp LED', 'Touch dimmer, 5 color temps, USB charging port', 2999, 'USD', 80, 'LMP-LED-001'),
			('a0000001-0000-0000-0000-000000000010', 'Wireless Charger Pad', 'Qi 15W fast charge, LED indicator, slim design', 1999, 'USD', 150, 'CHG-QI-001'),
			('a0000001-0000-0000-0000-000000000011', 'Smart Watch', 'GPS, heart rate, sleep tracking, 7-day battery', 29999, 'USD', 25, 'WATCH-SM-001'),
			('a0000001-0000-0000-0000-000000000012', 'Portable SSD 1TB', 'NVMe, 1050MB/s read, USB-C, shock resistant', 8999, 'USD', 45, 'SSD-1TB-001'),
			('a0000001-0000-0000-0000-000000000013', 'Ethernet Cable 3m', 'Cat8, 40Gbps, shielded, gold-plated connectors', 1299, 'USD', 200, 'CBL-ETH-001'),
			('a0000001-0000-0000-0000-000000000014', 'Mouse Pad XL', '900x400mm, stitched edges, non-slip rubber base', 1999, 'USD', 130, 'PAD-XL-001'),
			('a0000001-0000-0000-0000-000000000015', 'USB Microphone', 'Cardioid condenser, mute button, gain control', 6999, 'USD', 40, 'MIC-USB-001'),
			('a0000001-0000-0000-0000-000000000016', 'Graphics Tablet', '10x6 inch, 8192 pressure levels, wireless pen', 7999, 'USD', 30, 'TAB-GFX-001'),
			('a0000001-0000-0000-0000-000000000017', 'Cable Management Kit', '120 pieces: clips, ties, sleeves, labels', 1499, 'USD', 180, 'CBL-KIT-001'),
			('a0000001-0000-0000-0000-000000000018', 'Monitor Arm', 'Single arm, gas spring, VESA 75/100, clamp mount', 4499, 'USD', 55, 'ARM-MON-001'),
			('a0000001-0000-0000-0000-000000000019', 'Keyboard Wrist Rest', 'Memory foam, ergonomic, non-slip, washable cover', 1999, 'USD', 95, 'RST-WRT-001'),
			('a0000001-0000-0000-0000-000000000020', 'Privacy Screen Filter 27"', 'Anti-glare, anti-blue light, easy install', 3999, 'USD', 40, 'FLT-PRV-001')
		ON CONFLICT (sku) DO NOTHING`,
	}
	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			logger.Error("failed to run migration", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	jwtManager := auth.NewJWTManager(0)
	tokenResolver, err := auth.NewGRPCTokenResolver(userServiceAddr)
	if err != nil {
		logger.Error("failed to create token resolver", slog.String("error", err.Error()))
		os.Exit(1)
	}

	repo := repository.NewPostgresRepository(pool)
	svc := service.NewProductService(repo)
	srv := productHandler.NewServer(svc)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(logger),
			interceptors.LoggingInterceptor(logger),
			auth.UnaryAuthInterceptor(jwtManager, tokenResolver),
		),
	)

	productv1.RegisterProductServiceServer(grpcServer, srv)
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
		logger.Info("product-service listening", slog.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc serve error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down product-service")
	grpcServer.GracefulStop()
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
