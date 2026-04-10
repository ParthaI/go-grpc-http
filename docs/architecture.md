# Go gRPC/HTTP Microservice Architecture Plan

## Context

Build a realistic, production-like learning/reference project demonstrating microservice architecture in Go using gRPC for inter-service communication. The architecture combines two key patterns: **API Gateway + gRPC** and **Event-driven + CQRS**, across 5 services + 1 gateway. Domain: **Order Management System** (users, products, orders, payments, notifications).

---

## Architecture Overview

```
                        CLIENTS (Browser, Mobile, CLI)
                                  |
                                  | REST/HTTP (JSON)
                                  v
                        +-------------------+
                        |   API GATEWAY     |
                        |  (grpc-gateway)   |
                        |  :8080 HTTP       |
                        +-----+---+---+-----+
                              |   |   |        gRPC (internal)
                 +------------+   |   +------------+
                 v                v                 v
          +------------+  +------------+   +--------------+
          | USER SVC   |  | PRODUCT SVC|   | PAYMENT SVC  |
          | :50051     |  | :50052     |   | :50054       |
          +-----+------+  +-----+------+   +------+-------+
                |               |                  |
             [PG:user]      [PG:product]       [PG:payment]
                                |
                          gRPC  | (stock check)
                                v
                        +----------------+
                        |  ORDER SVC     |
                        |  :50053        |
                        |  CQRS + Events |
                        +---+-------+----+
                            |       |
                     [PG:order]  [Redis:read-model]
                            |
                            | publishes events
                            v
                     +--------------+
                     |    NATS      |
                     |  JetStream   |
                     +--+-----+----+
                        |     |
              subscribes|     |subscribes
                        v     v
              +----------+  +-----------------+
              | PAYMENT  |  | NOTIFICATION SVC |
              | SVC      |  | (event-only,     |
              |          |  |  no REST routes)  |
              +----------+  +-----------------+
```

---

## Service Breakdown

### 1. api-gateway (`:8080` HTTP, `:8081` gRPC)

- **Pattern:** grpc-gateway
- **DB:** None (stateless)
- **Role:** REST-to-gRPC translation, JWT validation, rate limiting, CORS

### 2. user-service (`:50051`)

- **Pattern:** Standard CRUD
- **DB:** PostgreSQL
- **Role:** Registration, login, JWT issuance, profile CRUD

### 3. product-service (`:50052`)

- **Pattern:** Standard CRUD
- **DB:** PostgreSQL
- **Role:** Product catalog, inventory, pricing, stock reservation

### 4. order-service (`:50053`)

- **Pattern:** CQRS + Event Sourcing
- **DB:** PostgreSQL (write) + Redis (read)
- **Role:** Order placement, lifecycle, fulfillment tracking

### 5. payment-service (`:50054`)

- **Pattern:** CRUD + Event consumer/producer
- **DB:** PostgreSQL
- **Role:** Payment processing, refunds, ledger

### 6. notification-service (`:50055` internal)

- **Pattern:** Pure event consumer (no REST)
- **DB:** PostgreSQL
- **Role:** Email/SMS/push notifications, audit trail

---

## Design Patterns Covered

1. **API Gateway** -- grpc-gateway auto-generates REST from proto annotations
2. **CQRS** -- Order service has separate `OrderCommandService` and `OrderQueryService` gRPC services
3. **Event Sourcing** -- Order writes stored as events in `event_store` table, aggregate rebuilt from events
4. **Event-driven** -- NATS JetStream for async inter-service events (orders.created, payments.completed, etc.)
5. **Database per Service** -- Each service has its own PostgreSQL instance
6. **Circuit Breaker / Resilience** -- gRPC interceptors for retry, timeout, recovery
7. **Distributed Tracing** -- OpenTelemetry across all services + NATS messages
8. **JWT Auth** -- Gateway validates tokens, forwards user_id in gRPC metadata

---

## Event Flow (Order Placement)

```
1. Client POST /api/v1/orders -> Gateway -> gRPC PlaceOrder -> Order Service
2. Order Service calls Product Service (gRPC: ReserveStock)
3. Order Service calls User Service (gRPC: validate user)
4. Order Service writes OrderCreatedEvent to event_store (PostgreSQL)
5. Order Service publishes "orders.created" to NATS
6. [Async] Order projector subscribes -> updates Redis read model
7. [Async] Payment Service subscribes -> processes payment -> publishes "payments.completed"
8. [Async] Notification Service subscribes -> sends confirmation email
9. [Async] Order Service subscribes to "payments.completed" -> writes OrderPaidEvent -> updates read model
```

---

## End-to-End Data Flow: Place Order

### Phase 1: Synchronous Request (REST + gRPC)

```
Client                 Gateway              Order Svc
  |                       |                     |
  | POST /api/v1/orders   |                     |
  | + Bearer JWT token    |                     |
  |---------------------->|                     |
  |                       |                     |
  |                  [Validate JWT]              |
  |                  [Extract user_id]           |
  |                       |                     |
  |                       | gRPC: PlaceOrder    |
  |                       |-------------------->|
  |                       |                     |
```

```
Order Svc             Product Svc
  |                       |
  | gRPC: ReserveStock    |
  |---------------------->|
  | OK / Insufficient     |
  |<----------------------|
  |                       |
  | [Write OrderCreated   |
  |  to event_store (PG)] |
  |                       |
  | [Publish to NATS:     |
  |  "orders.created"]    |
```

```
Client                 Gateway              Order Svc
  |                       |                     |
  |                       | PlaceOrderResponse  |
  |                       |<--------------------|
  | 201 {order_id}        |                     |
  |<----------------------|                     |
```

### Phase 2: Asynchronous Events (NATS JetStream)

```
NATS                  Payment Svc
  |                       |
  | "orders.created"      |
  |---------------------->|
  |                       |
  |                  [Process payment]
  |                       |
  | "payments.completed"  |
  |<----------------------|
```

```
NATS                  Notification Svc
  |                       |
  | "orders.created"      |
  |---------------------->|
  |                  [Send confirmation email]
  |                       |
```

```
NATS                  Order Svc
  |                       |
  | "payments.completed"  |
  |---------------------->|
  |                       |
  |                  [Write OrderPaidEvent]
  |                  [Update Redis read model]
  |                       |
  | "orders.paid"         |
  |<----------------------|
```

```
NATS                  Notification Svc
  |                       |
  | "orders.paid"         |
  |---------------------->|
  |                  [Send payment receipt]
  |                       |
```

---

## Technology Stack

### Core

- **gRPC:** `google.golang.org/grpc`
- **REST Gateway:** `github.com/grpc-ecosystem/grpc-gateway/v2`
- **Proto management:** `buf` CLI (lint, codegen, breaking change detection)
- **Proto validation:** `github.com/bufbuild/protovalidate-go`

### Data

- **PostgreSQL:** `github.com/jackc/pgx/v5` (high-performance, pure Go)
- **SQL builder:** `github.com/Masterminds/squirrel`
- **Migrations:** `github.com/golang-migrate/migrate/v4`
- **Redis:** `github.com/redis/go-redis/v9`
- **Messaging:** `github.com/nats-io/nats.go` (JetStream)

### Infrastructure

- **Config:** `github.com/caarlos0/env/v11` (struct tags to env vars)
- **Logging:** `log/slog` (Go stdlib, structured JSON)
- **Tracing:** `go.opentelemetry.io/otel` + Jaeger
- **Metrics:** `github.com/prometheus/client_golang`

### Auth and Middleware

- **JWT:** `github.com/golang-jwt/jwt/v5`
- **gRPC middleware:** `github.com/grpc-ecosystem/go-grpc-middleware/v2`
- **Password hashing:** `golang.org/x/crypto/bcrypt`
- **UUID:** `github.com/google/uuid`

### Testing

- **Assertions:** `github.com/stretchr/testify`
- **Integration:** `github.com/testcontainers/testcontainers-go`

---

## Directory Structure

```
go-grpc-http/
|-- go.mod                              # Single Go module (monorepo)
|-- go.sum
|-- Makefile                            # Build, proto gen, docker, lint, test
|-- buf.yaml                            # Buf configuration (lint + breaking)
|-- buf.gen.yaml                        # Buf code generation config
|-- docker-compose.yml                  # Full infrastructure stack
|-- .env.example                        # Environment variable template
|-- .gitignore
|
|-- proto/                              # All .proto definitions
|   |-- common/
|   |   |-- v1/
|   |       |-- common.proto            # Shared types: Pagination, Money, Address
|   |
|   |-- user/
|   |   |-- v1/
|   |       |-- user.proto              # UserService RPCs + HTTP annotations
|   |
|   |-- product/
|   |   |-- v1/
|   |       |-- product.proto           # ProductService RPCs
|   |
|   |-- order/
|   |   |-- v1/
|   |       |-- order.proto             # OrderCommandService + OrderQueryService (CQRS)
|   |       |-- order_events.proto      # Event message types for NATS
|   |
|   |-- payment/
|   |   |-- v1/
|   |       |-- payment.proto           # PaymentService RPCs
|   |
|   |-- notification/
|       |-- v1/
|           |-- notification_events.proto
|
|-- gen/                                # Generated code from proto (git-committed)
|   |-- go/                             # Generated Go code
|   |   |-- common/v1/
|   |   |-- user/v1/
|   |   |-- product/v1/
|   |   |-- order/v1/
|   |   |-- payment/v1/
|   |   |-- notification/v1/
|   |-- openapiv2/                      # Generated Swagger/OpenAPI specs
|
|-- cmd/                                # Application entrypoints
|   |-- gateway/
|   |   |-- main.go                     # API Gateway entrypoint
|   |-- user-service/
|   |   |-- main.go
|   |-- product-service/
|   |   |-- main.go
|   |-- order-service/
|   |   |-- main.go
|   |-- payment-service/
|   |   |-- main.go
|   |-- notification-service/
|       |-- main.go
|
|-- internal/                           # Private application code
|   |-- gateway/
|   |   |-- server.go                   # HTTP mux setup, grpc-gateway registration
|   |   |-- middleware/
|   |       |-- auth.go                 # JWT validation middleware
|   |       |-- logging.go             # Request logging middleware
|   |       |-- ratelimit.go           # Rate limiting middleware
|   |       |-- cors.go               # CORS middleware
|   |
|   |-- user/
|   |   |-- server.go                   # gRPC server impl (UserServiceServer)
|   |   |-- service/
|   |   |   |-- user_service.go        # Business logic
|   |   |   |-- user_service_test.go
|   |   |-- repository/
|   |   |   |-- user_repository.go     # Interface
|   |   |   |-- postgres.go            # PostgreSQL implementation
|   |   |-- model/
|   |       |-- user.go                # Domain model
|   |
|   |-- product/
|   |   |-- server.go
|   |   |-- service/
|   |   |   |-- product_service.go
|   |   |   |-- product_service_test.go
|   |   |-- repository/
|   |   |   |-- product_repository.go
|   |   |   |-- postgres.go
|   |   |-- model/
|   |       |-- product.go
|   |
|   |-- order/                          # CQRS service -- more complex structure
|   |   |-- server.go                   # gRPC server impl
|   |   |-- command/                    # WRITE SIDE (CQRS)
|   |   |   |-- handler.go             # Command handlers (PlaceOrder, CancelOrder)
|   |   |   |-- handler_test.go
|   |   |   |-- place_order.go         # PlaceOrder command definition
|   |   |   |-- cancel_order.go        # CancelOrder command definition
|   |   |-- query/                      # READ SIDE (CQRS)
|   |   |   |-- handler.go             # Query handlers (GetOrder, ListOrders)
|   |   |   |-- handler_test.go
|   |   |-- event/                      # Event handling
|   |   |   |-- store.go               # Event store interface + Postgres impl
|   |   |   |-- publisher.go           # NATS event publisher
|   |   |   |-- subscriber.go          # NATS event subscriber (projections)
|   |   |   |-- projector.go           # Builds read models from events
|   |   |   |-- types.go               # Event type constants
|   |   |-- aggregate/
|   |   |   |-- order.go               # Order aggregate root
|   |   |   |-- order_test.go
|   |   |-- repository/
|   |   |   |-- write_repository.go    # Event store repository (write side)
|   |   |   |-- read_repository.go     # Redis read model repository
|   |   |   |-- postgres_write.go      # Postgres event store impl
|   |   |   |-- redis_read.go          # Redis read model impl
|   |   |-- model/
|   |       |-- order.go               # Write model (aggregate state)
|   |       |-- order_view.go          # Read model (denormalized projection)
|   |       |-- events.go              # Domain event types
|   |
|   |-- payment/
|   |   |-- server.go
|   |   |-- service/
|   |   |   |-- payment_service.go
|   |   |-- event/
|   |   |   |-- subscriber.go          # Subscribes to order events
|   |   |   |-- publisher.go           # Publishes payment events
|   |   |-- repository/
|   |   |   |-- payment_repository.go
|   |   |   |-- postgres.go
|   |   |-- model/
|   |       |-- payment.go
|   |
|   |-- notification/
|       |-- service/
|       |   |-- notification_service.go
|       |-- event/
|       |   |-- subscriber.go          # Subscribes to all relevant events
|       |-- sender/
|       |   |-- email.go               # Email sender (mock/SMTP)
|       |   |-- sms.go                 # SMS sender (mock)
|       |-- repository/
|       |   |-- notification_log.go
|       |   |-- postgres.go
|       |-- model/
|           |-- notification.go
|
|-- pkg/                                # Shared libraries (importable by all services)
|   |-- auth/
|   |   |-- jwt.go                      # JWT generation and validation
|   |   |-- jwt_test.go
|   |   |-- interceptor.go             # gRPC auth interceptor (unary + stream)
|   |
|   |-- database/
|   |   |-- postgres.go                 # PostgreSQL connection helper
|   |   |-- migrations.go              # Migration runner (golang-migrate)
|   |
|   |-- messaging/
|   |   |-- nats.go                     # NATS connection + publish/subscribe helpers
|   |   |-- nats_test.go
|   |
|   |-- observability/
|   |   |-- tracing.go                  # OpenTelemetry tracer setup
|   |   |-- metrics.go                  # Prometheus metrics setup
|   |   |-- logging.go                  # Structured logger (slog) setup
|   |
|   |-- health/
|   |   |-- health.go                   # gRPC health check implementation
|   |
|   |-- errors/
|   |   |-- errors.go                   # Domain error types -> gRPC status mapping
|   |
|   |-- interceptors/
|       |-- logging.go                  # gRPC logging interceptor
|       |-- recovery.go                 # gRPC panic recovery interceptor
|       |-- tracing.go                  # gRPC OpenTelemetry interceptor
|       |-- validation.go              # gRPC request validation interceptor
|
|-- migrations/                         # Database migrations per service
|   |-- user/
|   |   |-- 001_create_users.up.sql
|   |   |-- 001_create_users.down.sql
|   |-- product/
|   |   |-- 001_create_products.up.sql
|   |   |-- 001_create_products.down.sql
|   |-- order/
|   |   |-- 001_create_event_store.up.sql
|   |   |-- 001_create_event_store.down.sql
|   |   |-- 002_create_order_snapshots.up.sql
|   |   |-- 002_create_order_snapshots.down.sql
|   |-- payment/
|   |   |-- 001_create_payments.up.sql
|   |   |-- 001_create_payments.down.sql
|   |-- notification/
|       |-- 001_create_notification_log.up.sql
|       |-- 001_create_notification_log.down.sql
|
|-- deployments/                        # Deployment configs
|   |-- docker/
|       |-- gateway.Dockerfile
|       |-- user-service.Dockerfile
|       |-- product-service.Dockerfile
|       |-- order-service.Dockerfile
|       |-- payment-service.Dockerfile
|       |-- notification-service.Dockerfile
|
|-- scripts/                            # Helper scripts
    |-- seed.go                         # Database seeding
```

---

## Proto File Definitions

### proto/common/v1/common.proto
```protobuf
syntax = "proto3";
package common.v1;
option go_package = "github.com/parthasarathi/go-grpc-http/gen/go/common/v1;commonv1";

import "google/protobuf/timestamp.proto";

message Pagination {
  int32 page = 1;
  int32 page_size = 2;
}

message PaginationResponse {
  int32 total_count = 1;
  int32 page = 2;
  int32 page_size = 3;
}

message Money {
  int64 amount_cents = 1;    // Amount in smallest currency unit
  string currency = 2;       // ISO 4217 currency code
}
```

### proto/user/v1/user.proto
```protobuf
syntax = "proto3";
package user.v1;
option go_package = "github.com/parthasarathi/go-grpc-http/gen/go/user/v1;userv1";

import "google/api/annotations.proto";

service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse) {
    option (google.api.http) = { post: "/api/v1/users/register" body: "*" };
  }
  rpc Login(LoginRequest) returns (LoginResponse) {
    option (google.api.http) = { post: "/api/v1/users/login" body: "*" };
  }
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = { get: "/api/v1/users/{user_id}" };
  }
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse) {
    option (google.api.http) = { put: "/api/v1/users/{user_id}" body: "*" };
  }
}
```

### proto/product/v1/product.proto
```protobuf
syntax = "proto3";
package product.v1;
option go_package = "github.com/parthasarathi/go-grpc-http/gen/go/product/v1;productv1";

import "google/api/annotations.proto";

service ProductService {
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse) {
    option (google.api.http) = { post: "/api/v1/products" body: "*" };
  }
  rpc GetProduct(GetProductRequest) returns (GetProductResponse) {
    option (google.api.http) = { get: "/api/v1/products/{product_id}" };
  }
  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse) {
    option (google.api.http) = { get: "/api/v1/products" };
  }
  rpc UpdateInventory(UpdateInventoryRequest) returns (UpdateInventoryResponse) {
    option (google.api.http) = { put: "/api/v1/products/{product_id}/inventory" body: "*" };
  }
  // Internal RPCs -- no HTTP annotation (only called by order-service via gRPC)
  rpc ReserveStock(ReserveStockRequest) returns (ReserveStockResponse);
  rpc ReleaseStock(ReleaseStockRequest) returns (ReleaseStockResponse);
}
```

### proto/order/v1/order.proto (CQRS -- separate Command and Query services)
```protobuf
syntax = "proto3";
package order.v1;
option go_package = "github.com/parthasarathi/go-grpc-http/gen/go/order/v1;orderv1";

import "google/api/annotations.proto";

// Commands (Write side)
service OrderCommandService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse) {
    option (google.api.http) = {
      post: "/api/v1/orders"
      body: "*"
    };
  }
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse) {
    option (google.api.http) = {
      post: "/api/v1/orders/{order_id}/cancel"
      body: "*"
    };
  }
}

// Queries (Read side)
service OrderQueryService {
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse) {
    option (google.api.http) = {
      get: "/api/v1/orders/{order_id}"
    };
  }
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse) {
    option (google.api.http) = {
      get: "/api/v1/orders"
    };
  }
  rpc ListOrdersByUser(ListOrdersByUserRequest) returns (ListOrdersResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{user_id}/orders"
    };
  }
}
```

### proto/order/v1/order_events.proto
```protobuf
syntax = "proto3";
package order.v1;
option go_package = "github.com/parthasarathi/go-grpc-http/gen/go/order/v1;orderv1";

import "google/protobuf/timestamp.proto";

// These are serialized to NATS payloads, not used in gRPC
message OrderCreatedEvent {
  string order_id = 1;
  string user_id = 2;
  repeated OrderItem items = 3;
  int64 total_amount_cents = 4;
  string currency = 5;
  google.protobuf.Timestamp created_at = 6;
}

message OrderPaidEvent {
  string order_id = 1;
  string payment_id = 2;
  google.protobuf.Timestamp paid_at = 3;
}

message OrderCancelledEvent {
  string order_id = 1;
  string reason = 2;
  google.protobuf.Timestamp cancelled_at = 3;
}
```

### proto/payment/v1/payment.proto
```protobuf
syntax = "proto3";
package payment.v1;
option go_package = "github.com/parthasarathi/go-grpc-http/gen/go/payment/v1;paymentv1";

import "google/api/annotations.proto";

service PaymentService {
  rpc GetPayment(GetPaymentRequest) returns (GetPaymentResponse) {
    option (google.api.http) = { get: "/api/v1/payments/{payment_id}" };
  }
  rpc ListPaymentsByOrder(ListPaymentsByOrderRequest) returns (ListPaymentsResponse) {
    option (google.api.http) = { get: "/api/v1/orders/{order_id}/payments" };
  }
  // ProcessPayment is triggered internally via NATS event, not via HTTP
  rpc ProcessPayment(ProcessPaymentRequest) returns (ProcessPaymentResponse);
  rpc RefundPayment(RefundPaymentRequest) returns (RefundPaymentResponse) {
    option (google.api.http) = { post: "/api/v1/payments/{payment_id}/refund" body: "*" };
  }
}
```

---

## CQRS Detail (Order Service)

### Write Side (Command)

```
PlaceOrder HTTP POST -> Gateway -> gRPC -> OrderCommandService.PlaceOrder
  |
  v
command/handler.go: PlaceOrderHandler
  |
  +-- Call product-service gRPC: ReserveStock (synchronous)
  +-- Call user-service gRPC: GetUser (validate user exists)
  +-- Create Order aggregate (aggregate/order.go)
  |     aggregate.PlaceOrder() -> produces OrderCreatedEvent
  +-- Persist events to event_store table (repository/postgres_write.go)
  |     INSERT INTO event_store (aggregate_id, event_type, payload, version, created_at)
  +-- Publish OrderCreatedEvent to NATS subject "orders.created"
  +-- Return order_id to client
```

### Read Side (Query + Projection)

```
GetOrder HTTP GET -> Gateway -> gRPC -> OrderQueryService.GetOrder
  |
  v
query/handler.go: GetOrderHandler
  |
  +-- Read from Redis (repository/redis_read.go)
  |     GET order:{order_id}  -> JSON of OrderView
  +-- If cache miss: rebuild from event_store (fallback)
  +-- Return OrderView to client
```

### Projection Builder (event/projector.go)

```
NATS subscriber (running in order-service):
  subscribes to "orders.>"  (all order events)
  |
  v
For each event:
  +-- Deserialize event payload
  +-- Load current OrderView from Redis (or create new)
  +-- Apply event to projection:
  |     OrderCreated  -> create OrderView{status: "pending", items: [...]}
  |     OrderPaid     -> update OrderView{status: "paid", paid_at: ...}
  |     OrderCancelled -> update OrderView{status: "cancelled"}
  +-- Save updated OrderView to Redis
  |     SET order:{order_id} <json> EX 86400
  +-- Also maintain secondary indexes:
       ZADD user_orders:{user_id} <timestamp> <order_id>
```

### Order Aggregate (aggregate/order.go)

```go
type Order struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Items     []OrderItem
    Status    OrderStatus
    Version   int
    Changes   []DomainEvent  // Uncommitted events
}

func (o *Order) PlaceOrder(userID uuid.UUID, items []OrderItem) error {
    if len(items) == 0 {
        return ErrEmptyOrder
    }
    o.apply(OrderCreatedEvent{...})
    return nil
}

func (o *Order) Cancel(reason string) error {
    if o.Status != StatusPending {
        return ErrCannotCancel
    }
    o.apply(OrderCancelledEvent{...})
    return nil
}

func (o *Order) apply(event DomainEvent) {
    o.when(event)                          // Mutate state
    o.Changes = append(o.Changes, event)   // Track uncommitted
}

func (o *Order) when(event DomainEvent) {
    switch e := event.(type) {
    case OrderCreatedEvent:
        o.Status = StatusPending
        o.Items = e.Items
    case OrderPaidEvent:
        o.Status = StatusPaid
    case OrderCancelledEvent:
        o.Status = StatusCancelled
    }
    o.Version++
}
```

### Read Model (model/order_view.go)

```go
// Denormalized, optimized for reading -- stored in Redis as JSON
type OrderView struct {
    OrderID     string        `json:"order_id"`
    UserID      string        `json:"user_id"`
    UserEmail   string        `json:"user_email"`    // Denormalized
    Items       []ItemView    `json:"items"`
    TotalCents  int64         `json:"total_cents"`
    Currency    string        `json:"currency"`
    Status      string        `json:"status"`
    PaymentID   *string       `json:"payment_id,omitempty"`
    CreatedAt   time.Time     `json:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at"`
}
```

### Event Store Schema

```sql
CREATE TABLE event_store (
    id             BIGSERIAL PRIMARY KEY,
    aggregate_id   UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL DEFAULT 'order',
    event_type     VARCHAR(100) NOT NULL,
    payload        JSONB NOT NULL,
    metadata       JSONB,
    version        INT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(aggregate_id, version)  -- Optimistic concurrency
);

CREATE INDEX idx_event_store_aggregate ON event_store(aggregate_id, version);
CREATE INDEX idx_event_store_type ON event_store(event_type, created_at);
```

---

## Event Catalog

**`orders.created`**
- Published by: order-service
- Consumed by: payment-svc, notification-svc, order-svc (projector)
- Trigger: PlaceOrder command

**`orders.paid`**
- Published by: order-service
- Consumed by: notification-svc, order-svc (projector)
- Trigger: Payment completed event handler

**`orders.cancelled`**
- Published by: order-service
- Consumed by: notification-svc, product-svc (release stock), order-svc (projector)
- Trigger: CancelOrder command or payment failure

**`payments.completed`**
- Published by: payment-service
- Consumed by: order-service
- Trigger: Successful payment processing

**`payments.failed`**
- Published by: payment-service
- Consumed by: order-service
- Trigger: Failed payment processing

**`users.registered`**
- Published by: user-service
- Consumed by: notification-service
- Trigger: New user registration

> NATS JetStream is used (not core NATS) for at-least-once delivery with durable consumers. Each subscriber uses a durable consumer name for replay on restart.

---

## Cross-Cutting Concerns

### Authentication Flow

```go
// pkg/auth/interceptor.go -- gRPC unary interceptor
func UnaryAuthInterceptor(jwtManager *JWTManager) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler) (any, error) {
        // Skip auth for public methods (Register, Login)
        if isPublicMethod(info.FullMethod) {
            return handler(ctx, req)
        }
        claims, err := extractAndValidateToken(ctx, jwtManager)
        if err != nil {
            return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
        }
        ctx = context.WithValue(ctx, claimsKey{}, claims)
        return handler(ctx, req)
    }
}
```

Gateway validates JWT, then forwards user_id in gRPC metadata. Internal services trust metadata from the gateway (internal network only).

### OpenTelemetry Tracing

```go
// pkg/observability/tracing.go
func InitTracer(serviceName string) (*sdktrace.TracerProvider, error) {
    exporter, _ := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("jaeger:4317"),
        otlptracegrpc.WithInsecure(),
    )
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

Every gRPC call propagates trace context via `otelgrpc`. NATS messages carry trace context in message headers.

### Health Checks

Every gRPC service implements standard `grpc.health.v1.Health` service. Gateway exposes `/healthz` as HTTP.

### Structured Logging

```go
// pkg/observability/logging.go
func NewLogger(serviceName string) *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })).With(slog.String("service", serviceName))
}
```

### gRPC Server Bootstrap Pattern

```go
// cmd/order-service/main.go (conceptual)
func main() {
    cfg := config.Load()
    logger := observability.NewLogger("order-service")
    tp, _ := observability.InitTracer("order-service")
    defer tp.Shutdown(context.Background())

    db := database.NewPostgres(cfg.DatabaseURL)
    database.RunMigrations(db, "migrations/order")
    rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
    nc, _ := nats.Connect(cfg.NatsURL)
    js, _ := nc.JetStream()

    // Wire up CQRS components
    eventStore := event.NewPostgresStore(db)
    publisher := event.NewNATSPublisher(js)
    readRepo := repository.NewRedisReadRepo(rdb)
    cmdHandler := command.NewHandler(eventStore, publisher, productClient, userClient)
    queryHandler := query.NewHandler(readRepo, eventStore)

    // Start projection subscriber (background)
    projector := event.NewProjector(readRepo)
    subscriber := event.NewNATSSubscriber(js, projector)
    go subscriber.Start(context.Background())

    // gRPC server with interceptor chain
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            otelgrpc.UnaryServerInterceptor(),
            interceptors.LoggingInterceptor(logger),
            interceptors.RecoveryInterceptor(),
            interceptors.ValidationInterceptor(),
            auth.UnaryAuthInterceptor(jwtManager),
        ),
    )
    orderv1.RegisterOrderCommandServiceServer(grpcServer, server.NewCommandServer(cmdHandler))
    orderv1.RegisterOrderQueryServiceServer(grpcServer, server.NewQueryServer(queryHandler))
    grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer(db, rdb, nc))

    lis, _ := net.Listen("tcp", ":50053")
    grpcServer.Serve(lis)
}
```

---

## Infrastructure (docker-compose.yml)

```yaml
version: "3.8"
services:
  # --- Databases (one per service) ---
  postgres-user:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: userdb
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports: ["5433:5432"]
    volumes: ["pgdata-user:/var/lib/postgresql/data"]

  postgres-product:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: productdb
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports: ["5434:5432"]
    volumes: ["pgdata-product:/var/lib/postgresql/data"]

  postgres-order:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: orderdb
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports: ["5435:5432"]
    volumes: ["pgdata-order:/var/lib/postgresql/data"]

  postgres-payment:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: paymentdb
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports: ["5436:5432"]
    volumes: ["pgdata-payment:/var/lib/postgresql/data"]

  postgres-notification:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: notificationdb
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports: ["5437:5432"]
    volumes: ["pgdata-notification:/var/lib/postgresql/data"]

  # --- Caching ---
  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  # --- Messaging ---
  nats:
    image: nats:2-alpine
    command: ["-js", "-m", "8222"]
    ports:
      - "4222:4222"    # Client
      - "8222:8222"    # Monitoring

  # --- Observability ---
  jaeger:
    image: jaegertracing/jaeger:2
    ports:
      - "16686:16686"   # UI
      - "4317:4317"     # OTLP gRPC
      - "4318:4318"     # OTLP HTTP

  # --- Application Services ---
  gateway:
    build: { context: ., dockerfile: deployments/docker/gateway.Dockerfile }
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      USER_SERVICE_ADDR: user-service:50051
      PRODUCT_SERVICE_ADDR: product-service:50052
      ORDER_SERVICE_ADDR: order-service:50053
      PAYMENT_SERVICE_ADDR: payment-service:50054
      JWT_SECRET: ${JWT_SECRET}
      OTEL_EXPORTER_OTLP_ENDPOINT: jaeger:4317
    depends_on: [user-service, product-service, order-service, payment-service]

  user-service:
    build: { context: ., dockerfile: deployments/docker/user-service.Dockerfile }
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres-user:5432/userdb?sslmode=disable
      GRPC_PORT: "50051"
      JWT_SECRET: ${JWT_SECRET}
      NATS_URL: nats://nats:4222
      OTEL_EXPORTER_OTLP_ENDPOINT: jaeger:4317
    depends_on: [postgres-user, nats, jaeger]

  product-service:
    build: { context: ., dockerfile: deployments/docker/product-service.Dockerfile }
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres-product:5432/productdb?sslmode=disable
      GRPC_PORT: "50052"
      NATS_URL: nats://nats:4222
      OTEL_EXPORTER_OTLP_ENDPOINT: jaeger:4317
    depends_on: [postgres-product, nats]

  order-service:
    build: { context: ., dockerfile: deployments/docker/order-service.Dockerfile }
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres-order:5432/orderdb?sslmode=disable
      REDIS_ADDR: redis:6379
      GRPC_PORT: "50053"
      NATS_URL: nats://nats:4222
      USER_SERVICE_ADDR: user-service:50051
      PRODUCT_SERVICE_ADDR: product-service:50052
      OTEL_EXPORTER_OTLP_ENDPOINT: jaeger:4317
    depends_on: [postgres-order, redis, nats, user-service, product-service]

  payment-service:
    build: { context: ., dockerfile: deployments/docker/payment-service.Dockerfile }
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres-payment:5432/paymentdb?sslmode=disable
      GRPC_PORT: "50054"
      NATS_URL: nats://nats:4222
      OTEL_EXPORTER_OTLP_ENDPOINT: jaeger:4317
    depends_on: [postgres-payment, nats]

  notification-service:
    build: { context: ., dockerfile: deployments/docker/notification-service.Dockerfile }
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres-notification:5432/notificationdb?sslmode=disable
      NATS_URL: nats://nats:4222
      OTEL_EXPORTER_OTLP_ENDPOINT: jaeger:4317
    depends_on: [postgres-notification, nats]

volumes:
  pgdata-user:
  pgdata-product:
  pgdata-order:
  pgdata-payment:
  pgdata-notification:
```

---

## Build Tooling (Makefile)

```makefile
.PHONY: proto build test lint docker-up docker-down

proto:
	buf generate

proto-lint:
	buf lint

proto-breaking:
	buf breaking --against '.git#branch=main'

build:
	@for svc in gateway user-service product-service order-service payment-service notification-service; do \
		echo "Building $$svc..."; \
		go build -o bin/$$svc ./cmd/$$svc; \
	done

test:
	go test ./... -race -cover -count=1

test-integration:
	go test ./... -race -tags=integration -count=1

lint:
	golangci-lint run ./...

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

run-%:
	go run ./cmd/$*
```

### Dockerfile Pattern (multi-stage)

```dockerfile
# deployments/docker/order-service.Dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /order-service ./cmd/order-service

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /order-service /order-service
COPY migrations/order /migrations
EXPOSE 50053
ENTRYPOINT ["/order-service"]
```

### Buf Configuration

**buf.yaml:**
```yaml
version: v2
modules:
  - path: proto
deps:
  - buf.build/googleapis/googleapis
  - buf.build/grpc-ecosystem/grpc-gateway
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

**buf.gen.yaml:**
```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc/go
    out: gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc-ecosystem/gateway
    out: gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc-ecosystem/openapiv2
    out: gen/openapiv2
```

---

## Implementation Phases

### Phase 1: Foundation
- Initialize Go module, buf config, shared proto types
- Implement `pkg/` shared libraries (database, auth, observability, messaging)
- Build user-service (full CRUD + JWT) + gateway with grpc-gateway
- **Verify**: `curl POST /api/v1/users/register` works end-to-end

### Phase 2: Second Service + Inter-Service Calls
- Implement product-service with stock management
- Register in gateway
- **Verify**: REST CRUD for products works

### Phase 3: CQRS + Event-Driven Core
- Set up NATS JetStream
- Implement order-service with CQRS: event store, aggregate, command/query handlers, projector
- Wire NATS publisher/subscriber
- **Verify**: Place order via REST -> event stored -> Redis read model updated

### Phase 4: Payment + Notification
- Payment-service subscribes to `orders.created`, processes payment, publishes result
- Notification-service subscribes to all events, sends notifications
- Order-service subscribes to payment events, updates order status
- **Verify**: Full end-to-end order flow (place -> pay -> notify)

### Phase 5: Polish
- OpenTelemetry tracing across all services
- Prometheus metrics, health checks
- Integration tests with testcontainers
- Dockerfiles + docker-compose
- Makefile with all build/test/run targets

---

## Verification Plan

1. **Unit tests**: Each service's business logic with `testify`
2. **Integration tests**: `testcontainers-go` to spin up real Postgres/Redis/NATS
3. **End-to-end flow**:
   - Register user -> Login (get JWT) -> Create product -> Place order -> Verify payment -> Verify notification
   - Check Jaeger UI for distributed traces spanning all services
   - Check Redis for read model projections
4. **Docker Compose**: `docker compose up` should start everything and be testable via curl

---

## Key Design Decisions

- **Monorepo (single go.mod)** -- Simplifies proto sharing and dependency management for learning
- **NATS over Kafka** -- Much simpler to operate; JetStream provides needed persistence
- **PostgreSQL as event store** -- Keeps infra simple while demonstrating the pattern
- **Redis for read projections** -- Shows CQRS principle: different stores optimized for access patterns
- **Separate Command/Query gRPC services** -- Makes CQRS split visible at API level
- **Notification service has no REST** -- Demonstrates pure event-driven service pattern
- **buf over raw protoc** -- Modern proto tooling with linting and breaking change detection
