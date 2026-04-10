package aggregate

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/parthasarathi/go-grpc-http/internal/order/model"
)

// Order is the aggregate root that enforces business rules
// and produces domain events.
type Order struct {
	ID         string
	UserID     string
	Items      []model.OrderItem
	TotalCents int64
	Currency   string
	Status     model.OrderStatus
	Version    int
	Changes    []model.StoredEvent // uncommitted events
}

// NewOrder creates a fresh aggregate (not yet persisted).
func NewOrder() *Order {
	return &Order{}
}

// LoadFromEvents rebuilds aggregate state by replaying stored events.
func LoadFromEvents(events []model.StoredEvent) (*Order, error) {
	o := NewOrder()
	for _, e := range events {
		if err := o.apply(e, false); err != nil {
			return nil, fmt.Errorf("replay event %s: %w", e.EventType, err)
		}
		o.Version = e.Version
	}
	return o, nil
}

// PlaceOrder creates a new order and produces an OrderCreated event.
func (o *Order) PlaceOrder(userID string, items []model.OrderItem, currency string) error {
	if len(items) == 0 {
		return fmt.Errorf("order must have at least one item")
	}

	var total int64
	for _, item := range items {
		total += item.PriceCents * int64(item.Quantity)
	}

	now := time.Now().UTC()
	o.ID = uuid.New().String()

	payload := model.OrderCreatedPayload{
		OrderID:    o.ID,
		UserID:     userID,
		Items:      items,
		TotalCents: total,
		Currency:   currency,
		CreatedAt:  now,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	event := model.StoredEvent{
		AggregateID:   o.ID,
		AggregateType: "order",
		EventType:     model.EventOrderCreated,
		Payload:       data,
		Version:       o.Version + 1,
		CreatedAt:     now,
	}

	return o.apply(event, true)
}

// Cancel cancels a pending order and produces an OrderCancelled event.
func (o *Order) Cancel(reason string) error {
	if o.Status != model.StatusPending {
		return fmt.Errorf("can only cancel pending orders, current status: %s", o.Status)
	}

	now := time.Now().UTC()
	payload := model.OrderCancelledPayload{
		OrderID:     o.ID,
		Reason:      reason,
		CancelledAt: now,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	event := model.StoredEvent{
		AggregateID:   o.ID,
		AggregateType: "order",
		EventType:     model.EventOrderCancelled,
		Payload:       data,
		Version:       o.Version + 1,
		CreatedAt:     now,
	}

	return o.apply(event, true)
}

// MarkPaid marks the order as paid and produces an OrderPaid event.
func (o *Order) MarkPaid(paymentID string) error {
	if o.Status != model.StatusPending {
		return fmt.Errorf("can only pay pending orders, current status: %s", o.Status)
	}

	now := time.Now().UTC()
	payload := model.OrderPaidPayload{
		OrderID:   o.ID,
		PaymentID: paymentID,
		PaidAt:    now,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	event := model.StoredEvent{
		AggregateID:   o.ID,
		AggregateType: "order",
		EventType:     model.EventOrderPaid,
		Payload:       data,
		Version:       o.Version + 1,
		CreatedAt:     now,
	}

	return o.apply(event, true)
}

// apply mutates state and optionally tracks the event as uncommitted.
func (o *Order) apply(event model.StoredEvent, isNew bool) error {
	switch event.EventType {
	case model.EventOrderCreated:
		var p model.OrderCreatedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		o.ID = p.OrderID
		o.UserID = p.UserID
		o.Items = p.Items
		o.TotalCents = p.TotalCents
		o.Currency = p.Currency
		o.Status = model.StatusPending

	case model.EventOrderPaid:
		o.Status = model.StatusPaid

	case model.EventOrderCancelled:
		o.Status = model.StatusCancelled
	}

	o.Version = event.Version
	if isNew {
		o.Changes = append(o.Changes, event)
	}
	return nil
}
