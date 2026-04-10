package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/parthasarathi/go-grpc-http/internal/order/model"
	"github.com/parthasarathi/go-grpc-http/internal/order/repository"
)

// Projector subscribes to order events and builds the Redis read model.
type Projector struct {
	js     nats.JetStreamContext
	reader repository.ReadRepository
	logger *slog.Logger
}

func NewProjector(js nats.JetStreamContext, reader repository.ReadRepository, logger *slog.Logger) *Projector {
	return &Projector{js: js, reader: reader, logger: logger}
}

// Start subscribes to all order events and projects them into the read model.
func (p *Projector) Start(ctx context.Context) error {
	sub, err := p.js.Subscribe("orders.*", func(msg *nats.Msg) {
		if err := p.handleMessage(ctx, msg); err != nil {
			p.logger.Error("projector error", slog.String("subject", msg.Subject), slog.String("error", err.Error()))
		}
		msg.Ack()
	}, nats.Durable("order-projector"), nats.DeliverAll())
	if err != nil {
		return fmt.Errorf("subscribe orders.*: %w", err)
	}

	<-ctx.Done()
	sub.Unsubscribe()
	return nil
}

func (p *Projector) handleMessage(ctx context.Context, msg *nats.Msg) error {
	switch msg.Subject {
	case "orders.created":
		return p.handleOrderCreated(ctx, msg.Data)
	case "orders.paid":
		return p.handleOrderPaid(ctx, msg.Data)
	case "orders.cancelled":
		return p.handleOrderCancelled(ctx, msg.Data)
	default:
		p.logger.Warn("unknown subject", slog.String("subject", msg.Subject))
		return nil
	}
}

func (p *Projector) handleOrderCreated(ctx context.Context, data []byte) error {
	var payload model.OrderCreatedPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	view := &model.OrderView{
		OrderID:    payload.OrderID,
		UserID:     payload.UserID,
		Items:      payload.Items,
		TotalCents: payload.TotalCents,
		Currency:   payload.Currency,
		Status:     string(model.StatusPending),
		CreatedAt:  payload.CreatedAt,
		UpdatedAt:  payload.CreatedAt,
	}

	if err := p.reader.Save(ctx, view); err != nil {
		return fmt.Errorf("save view: %w", err)
	}

	p.logger.Info("projected order.created", slog.String("order_id", payload.OrderID))
	return nil
}

func (p *Projector) handleOrderPaid(ctx context.Context, data []byte) error {
	var payload model.OrderPaidPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	view, err := p.reader.GetByID(ctx, payload.OrderID)
	if err != nil || view == nil {
		return fmt.Errorf("get view for paid event: %w", err)
	}

	view.Status = string(model.StatusPaid)
	view.PaymentID = payload.PaymentID
	view.UpdatedAt = time.Now().UTC()

	if err := p.reader.Save(ctx, view); err != nil {
		return fmt.Errorf("save view: %w", err)
	}

	p.logger.Info("projected order.paid", slog.String("order_id", payload.OrderID))
	return nil
}

func (p *Projector) handleOrderCancelled(ctx context.Context, data []byte) error {
	var payload model.OrderCancelledPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	view, err := p.reader.GetByID(ctx, payload.OrderID)
	if err != nil || view == nil {
		return fmt.Errorf("get view for cancelled event: %w", err)
	}

	view.Status = string(model.StatusCancelled)
	view.UpdatedAt = time.Now().UTC()

	if err := p.reader.Save(ctx, view); err != nil {
		return fmt.Errorf("save view: %w", err)
	}

	p.logger.Info("projected order.cancelled", slog.String("order_id", payload.OrderID))
	return nil
}
