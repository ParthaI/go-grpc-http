package event

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/nats-io/nats.go"
)

// Bridge forwards events from NATS JetStream to RabbitMQ.
// Other services continue publishing to NATS (no changes needed).
// The notification service consumes from RabbitMQ instead.
type Bridge struct {
	js      nats.JetStreamContext
	channel *amqp.Channel
	logger  *slog.Logger
}

func NewBridge(js nats.JetStreamContext, channel *amqp.Channel, logger *slog.Logger) *Bridge {
	return &Bridge{js: js, channel: channel, logger: logger}
}

func (b *Bridge) Start(ctx context.Context) error {
	// Forward order events: NATS orders.* → RabbitMQ notifications exchange
	orderSub, err := b.js.Subscribe("orders.*", func(msg *nats.Msg) {
		if err := b.forward(msg.Subject, msg.Data); err != nil {
			b.logger.Error("bridge forward order event", slog.String("subject", msg.Subject), slog.String("error", err.Error()))
		}
		msg.Ack()
	}, nats.Durable("rabbitmq-bridge-orders"), nats.DeliverAll())
	if err != nil {
		return fmt.Errorf("bridge subscribe orders.*: %w", err)
	}

	// Forward payment events: NATS payments.* → RabbitMQ notifications exchange
	paymentSub, err := b.js.Subscribe("payments.*", func(msg *nats.Msg) {
		if err := b.forward(msg.Subject, msg.Data); err != nil {
			b.logger.Error("bridge forward payment event", slog.String("subject", msg.Subject), slog.String("error", err.Error()))
		}
		msg.Ack()
	}, nats.Durable("rabbitmq-bridge-payments"), nats.DeliverAll())
	if err != nil {
		return fmt.Errorf("bridge subscribe payments.*: %w", err)
	}

	b.logger.Info("NATS→RabbitMQ bridge started")

	<-ctx.Done()
	orderSub.Unsubscribe()
	paymentSub.Unsubscribe()
	return nil
}

// forward publishes a message to the RabbitMQ "notifications" exchange
// using the NATS subject as the routing key (e.g., "orders.created").
func (b *Bridge) forward(routingKey string, body []byte) error {
	err := b.channel.Publish(
		"notifications", // exchange
		routingKey,      // routing key = NATS subject (e.g., "orders.created")
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // survive broker restart
		},
	)
	if err != nil {
		return fmt.Errorf("rabbitmq publish %s: %w", routingKey, err)
	}

	b.logger.Debug("bridged event", slog.String("routing_key", routingKey), slog.Int("bytes", len(body)))
	return nil
}
