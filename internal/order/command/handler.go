package command

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	productv1 "github.com/parthasarathi/go-grpc-http/gen/go/product/v1"
	"github.com/parthasarathi/go-grpc-http/internal/order/aggregate"
	"github.com/parthasarathi/go-grpc-http/internal/order/event"
	"github.com/parthasarathi/go-grpc-http/internal/order/model"
)

// Handler processes write-side commands.
type Handler struct {
	eventStore    *event.Store
	publisher     *event.Publisher
	productClient productv1.ProductServiceClient
}

func NewHandler(eventStore *event.Store, publisher *event.Publisher, productAddr string) (*Handler, error) {
	conn, err := grpc.NewClient(productAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to product service: %w", err)
	}

	return &Handler{
		eventStore:    eventStore,
		publisher:     publisher,
		productClient: productv1.NewProductServiceClient(conn),
	}, nil
}

// PlaceOrder validates stock, creates the order aggregate, persists events, and publishes.
func (h *Handler) PlaceOrder(ctx context.Context, userID string, items []model.OrderItem, currency string) (*aggregate.Order, error) {
	// Enrich items with product details and reserve stock
	var reserved []model.OrderItem
	enrichedItems := make([]model.OrderItem, 0, len(items))
	for _, item := range items {
		// Fetch product details (name, price)
		product, err := h.productClient.GetProduct(ctx, &productv1.GetProductRequest{
			ProductId: item.ProductID,
		})
		if err != nil {
			h.rollbackReservations(ctx, reserved)
			return nil, fmt.Errorf("get product %s: %w", item.ProductID, err)
		}

		// Reserve stock
		resp, err := h.productClient.ReserveStock(ctx, &productv1.ReserveStockRequest{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
		if err != nil {
			h.rollbackReservations(ctx, reserved)
			return nil, fmt.Errorf("reserve stock for %s: %w", item.ProductID, err)
		}
		if !resp.Success {
			h.rollbackReservations(ctx, reserved)
			return nil, fmt.Errorf("insufficient stock for product %s", item.ProductID)
		}

		enriched := model.OrderItem{
			ProductID:   item.ProductID,
			ProductName: product.Name,
			Quantity:    item.Quantity,
			PriceCents:  product.PriceCents,
		}
		reserved = append(reserved, enriched)
		enrichedItems = append(enrichedItems, enriched)
	}

	// Create aggregate and produce events
	order := aggregate.NewOrder()
	if err := order.PlaceOrder(userID, enrichedItems, currency); err != nil {
		h.rollbackReservations(ctx, reserved)
		return nil, fmt.Errorf("place order: %w", err)
	}

	// Persist events to event store
	if err := h.eventStore.Append(ctx, order.Changes); err != nil {
		h.rollbackReservations(ctx, reserved)
		return nil, fmt.Errorf("persist events: %w", err)
	}

	// Publish events to NATS
	if err := h.publisher.Publish(order.Changes); err != nil {
		// Events are persisted but not published — projector can replay from store
		return order, nil
	}

	return order, nil
}

// CancelOrder loads the aggregate, cancels it, persists and publishes events.
func (h *Handler) CancelOrder(ctx context.Context, orderID, reason string) error {
	events, err := h.eventStore.Load(ctx, orderID)
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}
	if len(events) == 0 {
		return fmt.Errorf("order not found")
	}

	order, err := aggregate.LoadFromEvents(events)
	if err != nil {
		return fmt.Errorf("rebuild aggregate: %w", err)
	}

	if err := order.Cancel(reason); err != nil {
		return err
	}

	if err := h.eventStore.Append(ctx, order.Changes); err != nil {
		return fmt.Errorf("persist events: %w", err)
	}

	// Release stock for cancelled items
	for _, item := range order.Items {
		h.productClient.ReleaseStock(ctx, &productv1.ReleaseStockRequest{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	if err := h.publisher.Publish(order.Changes); err != nil {
		return nil // events persisted, publish failure is non-fatal
	}

	return nil
}

func (h *Handler) rollbackReservations(ctx context.Context, items []model.OrderItem) {
	for _, item := range items {
		h.productClient.ReleaseStock(ctx, &productv1.ReleaseStockRequest{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}
}
