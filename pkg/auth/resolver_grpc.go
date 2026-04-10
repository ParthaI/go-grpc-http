package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userv1 "github.com/parthasarathi/go-grpc-http/gen/go/user/v1"
)

// GRPCTokenResolver looks up auth_token by calling user-service's
// internal GetAuthToken RPC via gRPC.
// Used by product-service, order-service, and any service that doesn't own the users table.
type GRPCTokenResolver struct {
	client userv1.UserServiceClient
}

func NewGRPCTokenResolver(userServiceAddr string) (*GRPCTokenResolver, error) {
	conn, err := grpc.NewClient(userServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to user service: %w", err)
	}
	return &GRPCTokenResolver{client: userv1.NewUserServiceClient(conn)}, nil
}

func (r *GRPCTokenResolver) ResolveAuthToken(ctx context.Context, userID string) (string, error) {
	resp, err := r.client.GetAuthToken(ctx, &userv1.GetAuthTokenRequest{UserId: userID})
	if err != nil {
		return "", fmt.Errorf("get auth token from user-service: %w", err)
	}
	if resp.AuthToken == "" {
		return "", fmt.Errorf("user has no auth token")
	}
	return resp.AuthToken, nil
}
