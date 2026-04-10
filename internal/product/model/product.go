package model

import "time"

type Product struct {
	ID            string
	Name          string
	Description   string
	PriceCents    int64
	Currency      string
	StockQuantity int32
	SKU           string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
