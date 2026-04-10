package event

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/parthasarathi/go-grpc-http/internal/order/model"
)

// Store persists and loads events from PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Append persists new events with optimistic concurrency (unique aggregate_id+version).
func (s *Store) Append(ctx context.Context, events []model.StoredEvent) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, e := range events {
		_, err := tx.Exec(ctx, `
			INSERT INTO event_store (aggregate_id, aggregate_type, event_type, payload, version, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			e.AggregateID, e.AggregateType, string(e.EventType),
			e.Payload, e.Version, e.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert event (version %d): %w", e.Version, err)
		}
	}

	return tx.Commit(ctx)
}

// Load retrieves all events for an aggregate, ordered by version.
func (s *Store) Load(ctx context.Context, aggregateID string) ([]model.StoredEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, aggregate_id, aggregate_type, event_type, payload, version, created_at
		FROM event_store
		WHERE aggregate_id = $1
		ORDER BY version ASC`, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []model.StoredEvent
	for rows.Next() {
		var e model.StoredEvent
		var eventType string
		if err := rows.Scan(&e.ID, &e.AggregateID, &e.AggregateType, &eventType, &e.Payload, &e.Version, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.EventType = model.EventType(eventType)
		events = append(events, e)
	}

	return events, nil
}
