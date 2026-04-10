package repository

import (
	"context"

	"github.com/parthasarathi/go-grpc-http/internal/order/model"
)

// ReadRepository provides read access to the denormalized order views.
type ReadRepository interface {
	GetByID(ctx context.Context, orderID string) (*model.OrderView, error)
	GetByUserID(ctx context.Context, userID string) ([]*model.OrderView, error)
	Save(ctx context.Context, view *model.OrderView) error
}
