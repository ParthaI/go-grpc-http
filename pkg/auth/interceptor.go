package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ctxKey struct{}

// publicMethods that don't require authentication.
var publicMethods = map[string]bool{
	// Health checks
	"/grpc.health.v1.Health/Check": true,
	"/user.v1.UserService/Register":            true,
	"/user.v1.UserService/Login":               true,
	"/product.v1.ProductService/GetProduct":     true,
	"/product.v1.ProductService/ListProducts":   true,
	"/order.v1.OrderQueryService/GetOrder":      true,
	"/order.v1.OrderQueryService/ListOrdersByUser": true,
	"/payment.v1.PaymentService/GetPayment":          true,
	"/payment.v1.PaymentService/ListPaymentsByOrder": true,
	// Internal service-to-service RPCs (not exposed via gateway to external clients)
	"/product.v1.ProductService/ReserveStock":  true,
	"/product.v1.ProductService/ReleaseStock":  true,
	"/user.v1.UserService/GetAuthToken":        true, // internal: used by other services for JWT verification
}

// UnaryAuthInterceptor validates JWT using per-user auth_token.
// Flow: extract token -> parse unverified to get user_id -> resolve auth_token -> verify signature.
func UnaryAuthInterceptor(jwtManager *JWTManager, resolver TokenResolver) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		claims, err := extractAndVerifyClaims(ctx, jwtManager, resolver)
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, ctxKey{}, claims)
		return handler(ctx, req)
	}
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ctxKey{}).(*Claims)
	return claims, ok
}

func extractAndVerifyClaims(ctx context.Context, jwtManager *JWTManager, resolver TokenResolver) (*Claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	tokenStr := values[0]
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	}

	// Step 1: Parse without verification to get user_id
	unverified, err := jwtManager.ParseUnverified(tokenStr)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "malformed token: %v", err)
	}
	if unverified.UserID == "" {
		return nil, status.Error(codes.Unauthenticated, "token missing user_id claim")
	}

	// Step 2: Look up the user's auth_token
	authToken, err := resolver.ResolveAuthToken(ctx, unverified.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "resolve auth token: %v", err)
	}

	// Step 3: Verify the JWT signature with the user's auth_token
	claims, err := jwtManager.Validate(tokenStr, authToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	return claims, nil
}
