package health

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/health/grpc_health_v1"
)

// Server implements the gRPC Health Check protocol.
type Server struct {
	grpc_health_v1.UnimplementedHealthServer
	checkers []func(ctx context.Context) error
}

func NewServer(checkers ...func(ctx context.Context) error) *Server {
	return &Server{checkers: checkers}
}

func (s *Server) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	for _, check := range s.checkers {
		if err := check(ctx); err != nil {
			return &grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			}, status.Errorf(codes.Unavailable, "unhealthy: %v", err)
		}
	}
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}
