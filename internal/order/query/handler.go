package query

import (
	"context"
	"fmt"

	"github.com/parthasarathi/go-grpc-http/internal/order/model"
	"github.com/parthasarathi/go-grpc-http/internal/order/repository"
)

// Handler processes read-side queries from the Redis read model.
type Handler struct {
	reader repository.ReadRepository
}

func NewHandler(reader repository.ReadRepository) *Handler {
	return &Handler{reader: reader}
}

func (h *Handler) GetOrder(ctx context.Context, orderID string) (*model.OrderView, error) {
	view, err := h.reader.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if view == nil {
		return nil, fmt.Errorf("order not found")
	}
	return view, nil
}

func (h *Handler) ListOrdersByUser(ctx context.Context, userID string) ([]*model.OrderView, error) {
	views, err := h.reader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return views, nil
}
