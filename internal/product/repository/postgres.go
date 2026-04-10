package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/parthasarathi/go-grpc-http/internal/product/model"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) ProductRepository {
	return &postgresRepo{pool: pool}
}

func (r *postgresRepo) Create(ctx context.Context, product *model.Product) error {
	query := `
		INSERT INTO products (id, name, description, price_cents, currency, stock_quantity, sku, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		product.ID, product.Name, product.Description,
		product.PriceCents, product.Currency, product.StockQuantity,
		product.SKU, product.CreatedAt, product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*model.Product, error) {
	query := `
		SELECT id, name, description, price_cents, currency, stock_quantity, sku, created_at, updated_at
		FROM products WHERE id = $1`

	var p model.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description,
		&p.PriceCents, &p.Currency, &p.StockQuantity,
		&p.SKU, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &p, nil
}

func (r *postgresRepo) List(ctx context.Context, page, pageSize int32) ([]*model.Product, int32, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var totalCount int32
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM products").Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	query := `
		SELECT id, name, description, price_cents, currency, stock_quantity, sku, created_at, updated_at
		FROM products ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description,
			&p.PriceCents, &p.Currency, &p.StockQuantity,
			&p.SKU, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, &p)
	}

	return products, totalCount, nil
}

func (r *postgresRepo) Update(ctx context.Context, product *model.Product) error {
	query := `
		UPDATE products
		SET name = $1, description = $2, price_cents = $3, currency = $4, updated_at = $5
		WHERE id = $6`

	tag, err := r.pool.Exec(ctx, query,
		product.Name, product.Description, product.PriceCents,
		product.Currency, product.UpdatedAt, product.ID,
	)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

func (r *postgresRepo) UpdateStock(ctx context.Context, id string, quantity int32) (*model.Product, error) {
	query := `
		UPDATE products SET stock_quantity = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, description, price_cents, currency, stock_quantity, sku, created_at, updated_at`

	var p model.Product
	err := r.pool.QueryRow(ctx, query, quantity, id).Scan(
		&p.ID, &p.Name, &p.Description,
		&p.PriceCents, &p.Currency, &p.StockQuantity,
		&p.SKU, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update stock: %w", err)
	}
	return &p, nil
}

func (r *postgresRepo) ReserveStock(ctx context.Context, productID string, quantity int32) (int32, error) {
	query := `
		UPDATE products
		SET stock_quantity = stock_quantity - $1, updated_at = NOW()
		WHERE id = $2 AND stock_quantity >= $1
		RETURNING stock_quantity`

	var remaining int32
	err := r.pool.QueryRow(ctx, query, quantity, productID).Scan(&remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("insufficient stock")
	}
	if err != nil {
		return 0, fmt.Errorf("reserve stock: %w", err)
	}
	return remaining, nil
}

func (r *postgresRepo) ReleaseStock(ctx context.Context, productID string, quantity int32) (int32, error) {
	query := `
		UPDATE products
		SET stock_quantity = stock_quantity + $1, updated_at = NOW()
		WHERE id = $2
		RETURNING stock_quantity`

	var remaining int32
	err := r.pool.QueryRow(ctx, query, quantity, productID).Scan(&remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("product not found")
	}
	if err != nil {
		return 0, fmt.Errorf("release stock: %w", err)
	}
	return remaining, nil
}
