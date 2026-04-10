package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/parthasarathi/go-grpc-http/internal/notification/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Log(ctx context.Context, n *model.Notification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notification_log (id, event_type, recipient, channel, subject, body, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		n.ID, n.EventType, n.Recipient, n.Channel, n.Subject, n.Body, n.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}
