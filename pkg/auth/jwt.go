package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims embedded in every JWT token.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// TokenResolver looks up a user's auth_token given their user ID.
// Each service implements this differently:
//   - user-service: direct DB lookup
//   - other services: gRPC call to user-service
type TokenResolver interface {
	ResolveAuthToken(ctx context.Context, userID string) (string, error)
}

// JWTManager handles JWT generation and validation.
// Tokens are signed with each user's unique auth_token (per-user secret).
type JWTManager struct {
	duration time.Duration
}

func NewJWTManager(duration time.Duration) *JWTManager {
	return &JWTManager{duration: duration}
}

// Generate creates a JWT signed with the user's auth_token.
func (m *JWTManager) Generate(userID, email, authToken string) (string, int64, error) {
	expiresAt := time.Now().Add(m.duration)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(authToken))
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}

	return tokenStr, expiresAt.Unix(), nil
}

// ParseUnverified extracts claims without verifying the signature.
// Used to get the user_id so we can look up their auth_token for verification.
func (m *JWTManager) ParseUnverified(tokenStr string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenStr, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("parse unverified: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// Validate verifies a JWT using the provided auth_token as the signing secret.
func (m *JWTManager) Validate(tokenStr, authToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(authToken), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
