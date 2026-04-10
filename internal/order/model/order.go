package model

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusCancelled OrderStatus = "cancelled"
)

type OrderItem struct {
	ProductID   string
	ProductName string
	Quantity    int32
	PriceCents  int64
}

// Order is the write-side domain model (aggregate state).
type Order struct {
	ID         string
	UserID     string
	Items      []OrderItem
	TotalCents int64
	Currency   string
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// OrderView is the read-side denormalized model stored in Redis.
type OrderView struct {
	OrderID    string      `json:"order_id"`
	UserID     string      `json:"user_id"`
	Items      []OrderItem `json:"items"`
	TotalCents int64       `json:"total_cents"`
	Currency   string      `json:"currency"`
	Status     string      `json:"status"`
	PaymentID  string      `json:"payment_id,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}
