package model

import "time"

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"
	StatusCompleted PaymentStatus = "completed"
	StatusFailed    PaymentStatus = "failed"
	StatusRefunded  PaymentStatus = "refunded"
)

type Payment struct {
	ID          string
	OrderID     string
	AmountCents int64
	Currency    string
	Status      PaymentStatus
	Method      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
