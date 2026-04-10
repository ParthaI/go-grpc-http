package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/parthasarathi/go-grpc-http/internal/order/aggregate"
	"github.com/parthasarathi/go-grpc-http/internal/order/repository"
	paymentevent "github.com/parthasarathi/go-grpc-http/internal/payment/event"
)

// PaymentSubscriber listens to payment events and updates order status.
type PaymentSubscriber struct {
	js         nats.JetStreamContext
	eventStore *Store
	publisher  *Publisher
	reader     repository.ReadRepository
	logger     *slog.Logger
}

func NewPaymentSubscriber(js nats.JetStreamContext, store *Store, pub *Publisher, reader repository.ReadRepository, logger *slog.Logger) *PaymentSubscriber {
	return &PaymentSubscriber{js: js, eventStore: store, publisher: pub, reader: reader, logger: logger}
}

func (s *PaymentSubscriber) Start(ctx context.Context) error {
	sub, err := s.js.Subscribe("payments.*", func(msg *nats.Msg) {
		if err := s.handlePaymentEvent(ctx, msg); err != nil {
			s.logger.Error("handle payment event", slog.String("subject", msg.Subject), slog.String("error", err.Error()))
		}
		msg.Ack()
	}, nats.Durable("order-payment-handler"), nats.DeliverAll())
	if err != nil {
		return fmt.Errorf("subscribe payments.*: %w", err)
	}

	<-ctx.Done()
	sub.Unsubscribe()
	return nil
}

func (s *PaymentSubscriber) handlePaymentEvent(ctx context.Context, msg *nats.Msg) error {
	var payload paymentevent.PaymentEvent
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	switch msg.Subject {
	case "payments.completed":
		return s.markOrderPaid(ctx, payload.OrderID, payload.PaymentID)
	case "payments.failed":
		return s.cancelOrder(ctx, payload.OrderID, "payment failed")
	}
	return nil
}

func (s *PaymentSubscriber) markOrderPaid(ctx context.Context, orderID, paymentID string) error {
	events, err := s.eventStore.Load(ctx, orderID)
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}
	if len(events) == 0 {
		return fmt.Errorf("order %s not found", orderID)
	}

	order, err := aggregate.LoadFromEvents(events)
	if err != nil {
		return fmt.Errorf("rebuild aggregate: %w", err)
	}

	if err := order.MarkPaid(paymentID); err != nil {
		s.logger.Warn("cannot mark paid", slog.String("order_id", orderID), slog.String("error", err.Error()))
		return nil
	}

	if err := s.eventStore.Append(ctx, order.Changes); err != nil {
		return fmt.Errorf("persist events: %w", err)
	}

	if err := s.publisher.Publish(order.Changes); err != nil {
		s.logger.Error("publish order.paid", slog.String("error", err.Error()))
	}

	s.logger.Info("order marked as paid", slog.String("order_id", orderID), slog.String("payment_id", paymentID))
	return nil
}

func (s *PaymentSubscriber) cancelOrder(ctx context.Context, orderID, reason string) error {
	events, err := s.eventStore.Load(ctx, orderID)
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}

	order, err := aggregate.LoadFromEvents(events)
	if err != nil {
		return fmt.Errorf("rebuild aggregate: %w", err)
	}

	if err := order.Cancel(reason); err != nil {
		s.logger.Warn("cannot cancel", slog.String("order_id", orderID), slog.String("error", err.Error()))
		return nil
	}

	if err := s.eventStore.Append(ctx, order.Changes); err != nil {
		return fmt.Errorf("persist events: %w", err)
	}

	s.publisher.Publish(order.Changes)
	s.logger.Info("order cancelled due to payment failure", slog.String("order_id", orderID))
	return nil
}
