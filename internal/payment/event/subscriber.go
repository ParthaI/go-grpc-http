package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	ordermodel "github.com/parthasarathi/go-grpc-http/internal/order/model"
	"github.com/parthasarathi/go-grpc-http/internal/payment/model"
	"github.com/parthasarathi/go-grpc-http/internal/payment/repository"
)

// Subscriber listens to order events and processes payments.
type Subscriber struct {
	js     nats.JetStreamContext
	repo   *repository.Repository
	pub    *Publisher
	logger *slog.Logger
}

func NewSubscriber(js nats.JetStreamContext, repo *repository.Repository, pub *Publisher, logger *slog.Logger) *Subscriber {
	return &Subscriber{js: js, repo: repo, pub: pub, logger: logger}
}

// Start subscribes to orders.created and processes payments.
func (s *Subscriber) Start(ctx context.Context) error {
	sub, err := s.js.Subscribe("orders.created", func(msg *nats.Msg) {
		if err := s.handleOrderCreated(ctx, msg.Data); err != nil {
			s.logger.Error("handle order.created", slog.String("error", err.Error()))
		}
		msg.Ack()
	}, nats.Durable("payment-processor"), nats.DeliverAll())
	if err != nil {
		return fmt.Errorf("subscribe orders.created: %w", err)
	}

	<-ctx.Done()
	sub.Unsubscribe()
	return nil
}

func (s *Subscriber) handleOrderCreated(ctx context.Context, data []byte) error {
	var payload ordermodel.OrderCreatedPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	s.logger.Info("processing payment",
		slog.String("order_id", payload.OrderID),
		slog.Int64("amount", payload.TotalCents),
	)

	now := time.Now().UTC()
	payment := &model.Payment{
		ID:          uuid.New().String(),
		OrderID:     payload.OrderID,
		AmountCents: payload.TotalCents,
		Currency:    payload.Currency,
		Status:      model.StatusPending,
		Method:      "card",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return fmt.Errorf("create payment: %w", err)
	}

	// Simulate payment processing (always succeeds in dev)
	payment.Status = model.StatusCompleted
	if err := s.repo.UpdateStatus(ctx, payment.ID, model.StatusCompleted); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// Publish payment result
	if err := s.pub.PublishCompleted(payment.ID, payload.OrderID); err != nil {
		s.logger.Error("publish payments.completed", slog.String("error", err.Error()))
	}

	s.logger.Info("payment completed",
		slog.String("payment_id", payment.ID),
		slog.String("order_id", payload.OrderID),
	)
	return nil
}
