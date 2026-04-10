# Phase 4: Payment + Notification Services - Setup Reference

## End-to-End Async Event Chain

```
Place Order
  |-> order-service writes OrderCreatedEvent to event_store
  |-> publishes "orders.created" to NATS
  |
  |-> [projector] builds Redis read model (status: pending)
  |-> [payment-service] subscribes, creates payment, processes it
  |     |-> publishes "payments.completed" to NATS
  |
  |-> [notification-service] subscribes to "orders.created"
  |     |-> sends "Order Confirmed" email, logs to DB
  |
  |-> [order-service] subscribes to "payments.completed"
  |     |-> writes OrderPaidEvent to event_store
  |     |-> publishes "orders.paid" to NATS
  |     |-> [projector] updates Redis (status: paid, paymentId set)
  |
  |-> [notification-service] subscribes to "payments.completed"
  |     |-> sends "Payment Successful" email
  |
  |-> [notification-service] subscribes to "orders.paid"
        |-> sends "Payment Received" email
```

## Verified Results

| Check | Result |
|-------|--------|
| Order total | 23997 cents (3 x $79.99) |
| Order status | `paid` (auto-updated by payment event) |
| Payment record | `completed`, amount matches order |
| Stock reserved | 15 -> 12 |
| Event store | v1=order.created, v2=order.paid |
| Notifications sent | 3: Order Confirmed, Payment Successful, Payment Received |

## Commands Executed

```bash
# Install dependencies (no new ones needed)
go mod tidy

# Generate payment proto
buf lint && buf generate

# Start all infrastructure (7 containers)
docker compose up -d

# Start all 6 services
go run ./cmd/user-service &
go run ./cmd/product-service &
go run ./cmd/order-service &
go run ./cmd/payment-service &
go run ./cmd/notification-service &
HTTP_PORT=9090 go run ./cmd/gateway &
```

## Go Modules Downloaded

No new dependencies in Phase 4. Payment and notification services use the same libraries: `pgx` (PostgreSQL), `nats.go` (NATS JetStream), `uuid` (ID generation).

## Files Written

### Payment Service

**`proto/payment/v1/payment.proto`** -- 3 RPCs: GetPayment, ListPaymentsByOrder (public), RefundPayment (authenticated). Payment creation happens via NATS event subscription, not via REST.

**`internal/payment/model/payment.go`** -- Payment domain model with status lifecycle: pending -> completed/failed -> refunded.

**`internal/payment/repository/postgres.go`** -- CRUD for payments table. `GetByOrderID` returns all payments for an order (supports retry scenarios).

**`internal/payment/event/subscriber.go`** -- Subscribes to `orders.created` with durable consumer "payment-processor". On each event: creates a payment record, simulates processing (always succeeds in dev), updates status to completed, publishes `payments.completed` to NATS.

**`internal/payment/event/publisher.go`** -- Publishes `payments.completed` and `payments.failed` events to NATS. Uses the `PaymentEvent` struct with payment_id, order_id, status, and timestamp.

**`internal/payment/server.go`** -- gRPC server implementing GetPayment, ListPaymentsByOrder, and RefundPayment. Refund enforces business rule: only completed payments can be refunded.

**`cmd/payment-service/main.go`** -- Connects to PostgreSQL (port 5436), NATS, creates PAYMENTS stream, starts event subscriber in background, runs gRPC server on port 50054.

### Notification Service (Pure Event Consumer)

**`internal/notification/model/notification.go`** -- Notification log entry: event type, recipient, channel, subject, body.

**`internal/notification/repository/postgres.go`** -- Logs every notification sent to the `notification_log` table for audit trail.

**`internal/notification/sender/email.go`** -- Mock email sender that logs to stdout. In production, would use SMTP/SendGrid/SES.

**`internal/notification/event/subscriber.go`** -- Subscribes to both `orders.*` and `payments.*` with separate durable consumers. Handles 5 event types:
- `orders.created` -> "Order Confirmed" email
- `orders.paid` -> "Payment Received" email
- `orders.cancelled` -> "Order Cancelled" email
- `payments.completed` -> "Payment Successful" email
- `payments.failed` -> "Payment Failed" email

Each notification is both sent (mock) and logged to PostgreSQL.

**`cmd/notification-service/main.go`** -- No gRPC server (pure event consumer). Connects to PostgreSQL (port 5437) and NATS, starts subscriber.

### Order Service Update

**`internal/order/event/payment_subscriber.go`** -- Subscribes to `payments.*` with durable consumer "order-payment-handler". On `payments.completed`: loads order aggregate from event store, calls `MarkPaid()`, persists OrderPaidEvent, publishes `orders.paid`. On `payments.failed`: cancels the order and releases stock.

### Modified Files

- **`internal/gateway/server.go`** -- Registers payment service handler
- **`cmd/gateway/main.go`** -- Added `PAYMENT_SERVICE_ADDR` config
- **`cmd/order-service/main.go`** -- Starts payment event subscriber in background
- **`pkg/auth/interceptor.go`** -- Added GetPayment and ListPaymentsByOrder as public
- **`docker-compose.yml`** -- Added postgres-payment (5436) and postgres-notification (5437)
- **`Makefile`** -- Added run-payment and run-notification targets

### Database Migrations

- **`migrations/payment/001_create_payments.up.sql`** -- payments table with order_id index
- **`migrations/notification/001_create_notification_log.up.sql`** -- notification_log table with event_type index
