package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/parthasarathi/go-grpc-http/internal/notification/model"
	"github.com/parthasarathi/go-grpc-http/internal/notification/repository"
	"github.com/parthasarathi/go-grpc-http/internal/notification/sender"
	ordermodel "github.com/parthasarathi/go-grpc-http/internal/order/model"
	paymentevent "github.com/parthasarathi/go-grpc-http/internal/payment/event"
)

// Subscriber consumes notification events from RabbitMQ.
type Subscriber struct {
	channel *amqp.Channel
	repo    *repository.Repository
	email   *sender.EmailSender
	logger  *slog.Logger
}

// notifyEmail is the recipient address for notifications.
// In production, you'd resolve user_id → email via UserService gRPC call.
// For now, configurable via NOTIFY_EMAIL env var.
var NotifyEmail = ""

func NewSubscriber(channel *amqp.Channel, repo *repository.Repository, email *sender.EmailSender, notifyEmail string, logger *slog.Logger) *Subscriber {
	NotifyEmail = notifyEmail
	return &Subscriber{channel: channel, repo: repo, email: email, logger: logger}
}

func (s *Subscriber) Start(ctx context.Context) error {
	// Declare a durable queue for notifications
	q, err := s.channel.QueueDeclare(
		"notification-queue", // name
		true,                 // durable — survives broker restart
		false,                // auto-delete
		false,                // exclusive
		false,                // no-wait
		nil,                  // args
	)
	if err != nil {
		return fmt.Errorf("queue declare: %w", err)
	}

	// Bind queue to receive all order events (orders.*)
	if err := s.channel.QueueBind(q.Name, "orders.*", "notifications", false, nil); err != nil {
		return fmt.Errorf("queue bind orders.*: %w", err)
	}

	// Bind queue to receive all payment events (payments.*)
	if err := s.channel.QueueBind(q.Name, "payments.*", "notifications", false, nil); err != nil {
		return fmt.Errorf("queue bind payments.*: %w", err)
	}

	// Start consuming
	msgs, err := s.channel.Consume(
		q.Name,               // queue
		"notification-consumer", // consumer tag
		false,                // auto-ack (false = manual ack for reliability)
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	s.logger.Info("rabbitmq notification subscriber started", slog.String("queue", q.Name))

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitmq channel closed")
			}
			if err := s.handleMessage(ctx, msg); err != nil {
				s.logger.Error("handle message",
					slog.String("routing_key", msg.RoutingKey),
					slog.String("error", err.Error()),
				)
			}
			msg.Ack(false)
		}
	}
}

func (s *Subscriber) resolveRecipient(fallback string) string {
	if NotifyEmail != "" {
		return NotifyEmail
	}
	return fallback
}

func (s *Subscriber) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	switch msg.RoutingKey {
	case "orders.created":
		var payload ordermodel.OrderCreatedPayload
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			return err
		}
		recipient := s.resolveRecipient(payload.UserID)
		s.email.Send(recipient, "Order Confirmed",
			fmt.Sprintf("Your order %s has been placed. Total: $%.2f",
				payload.OrderID, float64(payload.TotalCents)/100))

		return s.log(ctx, "orders.created", recipient,
			"Order Confirmed", fmt.Sprintf("Order %s placed", payload.OrderID))

	case "orders.paid":
		var payload ordermodel.OrderPaidPayload
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			return err
		}
		recipient := s.resolveRecipient(payload.OrderID)
		s.email.Send(recipient, "Payment Received",
			fmt.Sprintf("Payment %s received for order %s", payload.PaymentID, payload.OrderID))

		return s.log(ctx, "orders.paid", recipient,
			"Payment Received", fmt.Sprintf("Payment %s for order %s", payload.PaymentID, payload.OrderID))

	case "orders.cancelled":
		var payload ordermodel.OrderCancelledPayload
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			return err
		}
		recipient := s.resolveRecipient(payload.OrderID)
		s.email.Send(recipient, "Order Cancelled",
			fmt.Sprintf("Order %s has been cancelled. Reason: %s", payload.OrderID, payload.Reason))

		return s.log(ctx, "orders.cancelled", recipient,
			"Order Cancelled", fmt.Sprintf("Order %s cancelled: %s", payload.OrderID, payload.Reason))

	case "payments.completed":
		var payload paymentevent.PaymentEvent
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			return err
		}
		recipient := s.resolveRecipient(payload.OrderID)
		s.email.Send(recipient, "Payment Successful",
			fmt.Sprintf("Payment %s completed for order %s", payload.PaymentID, payload.OrderID))

		return s.log(ctx, "payments.completed", recipient,
			"Payment Successful", fmt.Sprintf("Payment %s completed", payload.PaymentID))

	case "payments.failed":
		var payload paymentevent.PaymentEvent
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			return err
		}
		recipient := s.resolveRecipient(payload.OrderID)
		s.email.Send(recipient, "Payment Failed",
			fmt.Sprintf("Payment failed for order %s", payload.OrderID))

		return s.log(ctx, "payments.failed", recipient,
			"Payment Failed", fmt.Sprintf("Payment failed for order %s", payload.OrderID))
	}
	return nil
}

func (s *Subscriber) log(ctx context.Context, eventType, recipient, subject, body string) error {
	return s.repo.Log(ctx, &model.Notification{
		ID:        uuid.New().String(),
		EventType: eventType,
		Recipient: recipient,
		Channel:   "email",
		Subject:   subject,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	})
}
