package event

import (
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/parthasarathi/go-grpc-http/internal/order/model"
)

// Publisher sends domain events to NATS JetStream.
type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
	return &Publisher{js: js}
}

// Publish sends events to NATS subjects based on event type.
// Subject format: "orders.created", "orders.paid", "orders.cancelled"
func (p *Publisher) Publish(events []model.StoredEvent) error {
	for _, e := range events {
		subject := natsSubject(e.EventType)
		if _, err := p.js.Publish(subject, e.Payload); err != nil {
			return fmt.Errorf("publish %s: %w", subject, err)
		}
	}
	return nil
}

func natsSubject(eventType model.EventType) string {
	switch eventType {
	case model.EventOrderCreated:
		return "orders.created"
	case model.EventOrderPaid:
		return "orders.paid"
	case model.EventOrderCancelled:
		return "orders.cancelled"
	default:
		return "orders.unknown"
	}
}
