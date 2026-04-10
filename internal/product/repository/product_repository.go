package repository

import (
	"context"

	"github.com/parthasarathi/go-grpc-http/internal/product/model"
)

type ProductRepository interface {
	Create(ctx context.Context, product *model.Product) error
	GetByID(ctx context.Context, id string) (*model.Product, error)
	List(ctx context.Context, page, pageSize int32) ([]*model.Product, int32, error)
	Update(ctx context.Context, product *model.Product) error
	UpdateStock(ctx context.Context, id string, quantity int32) (*model.Product, error)
	ReserveStock(ctx context.Context, productID string, quantity int32) (int32, error)
	ReleaseStock(ctx context.Context, productID string, quantity int32) (int32, error)
}
