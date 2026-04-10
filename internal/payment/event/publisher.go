package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// PaymentEvent is published to NATS when payment completes or fails.
type PaymentEvent struct {
	PaymentID string    `json:"payment_id"`
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) PublishCompleted(paymentID, orderID string) error {
	return p.publish("payments.completed", paymentID, orderID, "completed")
}

func (p *Publisher) PublishFailed(paymentID, orderID string) error {
	return p.publish("payments.failed", paymentID, orderID, "failed")
}

func (p *Publisher) publish(subject, paymentID, orderID, status string) error {
	evt := PaymentEvent{
		PaymentID: paymentID,
		OrderID:   orderID,
		Status:    status,
		Timestamp: time.Now().UTC(),
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := p.js.Publish(subject, data); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}
