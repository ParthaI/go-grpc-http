package model

import "time"

type EventType string

const (
	EventOrderCreated   EventType = "order.created"
	EventOrderPaid      EventType = "order.paid"
	EventOrderCancelled EventType = "order.cancelled"
)

// StoredEvent represents a persisted event in the event store.
type StoredEvent struct {
	ID            int64
	AggregateID   string
	AggregateType string
	EventType     EventType
	Payload       []byte
	Version       int
	CreatedAt     time.Time
}

// Domain event payloads

type OrderCreatedPayload struct {
	OrderID    string      `json:"order_id"`
	UserID     string      `json:"user_id"`
	Items      []OrderItem `json:"items"`
	TotalCents int64       `json:"total_cents"`
	Currency   string      `json:"currency"`
	CreatedAt  time.Time   `json:"created_at"`
}

type OrderPaidPayload struct {
	OrderID   string    `json:"order_id"`
	PaymentID string    `json:"payment_id"`
	PaidAt    time.Time `json:"paid_at"`
}

type OrderCancelledPayload struct {
	OrderID     string    `json:"order_id"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}
