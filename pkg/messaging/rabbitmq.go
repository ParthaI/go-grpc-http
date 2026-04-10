package messaging

import (
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ manages a connection and channel to RabbitMQ.
type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	logger  *slog.Logger
}

// NewRabbitMQ connects to RabbitMQ and opens a channel.
func NewRabbitMQ(url string, logger *slog.Logger) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	// Declare the notifications exchange (topic type for routing by event type)
	err = ch.ExchangeDeclare(
		"notifications", // name
		"topic",         // type — allows routing like "orders.*", "payments.*"
		true,            // durable — survives broker restart
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("rabbitmq exchange declare: %w", err)
	}

	logger.Info("connected to rabbitmq", slog.String("url", url))

	return &RabbitMQ{Conn: conn, Channel: ch, logger: logger}, nil
}

// Close shuts down the channel and connection.
func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
