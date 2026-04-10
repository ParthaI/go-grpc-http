package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/parthasarathi/go-grpc-http/internal/payment/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, p *model.Payment) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO payments (id, order_id, amount_cents, currency, status, method, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, p.OrderID, p.AmountCents, p.Currency,
		string(p.Status), p.Method, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*model.Payment, error) {
	var p model.Payment
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, order_id, amount_cents, currency, status, method, created_at, updated_at
		FROM payments WHERE id = $1`, id).Scan(
		&p.ID, &p.OrderID, &p.AmountCents, &p.Currency,
		&status, &p.Method, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}
	p.Status = model.PaymentStatus(status)
	return &p, nil
}

func (r *Repository) GetByOrderID(ctx context.Context, orderID string) ([]*model.Payment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, amount_cents, currency, status, method, created_at, updated_at
		FROM payments WHERE order_id = $1 ORDER BY created_at DESC`, orderID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*model.Payment
	for rows.Next() {
		var p model.Payment
		var status string
		if err := rows.Scan(
			&p.ID, &p.OrderID, &p.AmountCents, &p.Currency,
			&status, &p.Method, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		p.Status = model.PaymentStatus(status)
		payments = append(payments, &p)
	}
	return payments, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status model.PaymentStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE payments SET status = $1, updated_at = NOW() WHERE id = $2`,
		string(status), id)
	if err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("payment not found")
	}
	return nil
}
