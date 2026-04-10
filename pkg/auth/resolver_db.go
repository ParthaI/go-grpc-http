package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTokenResolver looks up auth_token directly from the users database.
// Used by user-service which owns the users table.
type DBTokenResolver struct {
	pool *pgxpool.Pool
}

func NewDBTokenResolver(pool *pgxpool.Pool) *DBTokenResolver {
	return &DBTokenResolver{pool: pool}
}

func (r *DBTokenResolver) ResolveAuthToken(ctx context.Context, userID string) (string, error) {
	var authToken string
	err := r.pool.QueryRow(ctx, "SELECT auth_token FROM users WHERE id = $1", userID).Scan(&authToken)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("user not found")
	}
	if err != nil {
		return "", fmt.Errorf("query auth_token: %w", err)
	}
	return authToken, nil
}
