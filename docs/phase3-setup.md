# Phase 3: CQRS + Event-Driven Order Service - Setup Reference

## Commands Executed

### 1. Install New Dependencies

```bash
go get github.com/nats-io/nats.go github.com/redis/go-redis/v9
```

**Why:** Two new infrastructure clients needed for the CQRS pattern. NATS JetStream is the event bus that decouples the write side from the read side -- when the order-service writes an event, NATS delivers it asynchronously to the projector (and later to payment-service and notification-service). Redis is the read model store -- it holds pre-computed, denormalized order views optimized for fast queries, separate from the PostgreSQL event store which is optimized for appending events.

### 2. Proto Generation

```bash
buf lint && buf generate
```

**Why:** Generated code for `proto/order/v1/order.proto` which defines two separate gRPC services -- `OrderCommandService` (PlaceOrder, CancelOrder) and `OrderQueryService` (GetOrder, ListOrdersByUser). Having two services in one proto file is the key CQRS pattern: commands mutate state through events, queries read from a separate denormalized store.

### 3. Start Infrastructure

```bash
docker compose up -d
```

**Why:** Adds three new containers: `postgres-order` (event store), `redis` (read model), and `nats` with JetStream enabled (`-js` flag). NATS JetStream provides durable message delivery with at-least-once semantics -- if the projector crashes, it replays unacknowledged messages on restart.

### 4. End-to-End Verification

```bash
# Place order (write side: event store + NATS + stock reservation)
curl -X POST http://localhost:9090/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_id":"...","items":[{"product_id":"...","quantity":2}]}'

# Get order (read side: Redis projection)
curl http://localhost:9090/api/v1/orders/{order_id}

# Cancel order (write side: new event + stock release)
curl -X POST http://localhost:9090/api/v1/orders/{order_id}/cancel \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"reason":"changed my mind"}'

# Verify event store (PostgreSQL)
psql -d orderdb -c "SELECT event_type, version FROM event_store WHERE aggregate_id='...'"
```

**Why:** Verifies the complete CQRS cycle:
1. PlaceOrder -> enriches items with product prices -> reserves stock via gRPC -> writes OrderCreatedEvent to event store -> publishes to NATS -> projector builds Redis read model
2. GetOrder -> reads from Redis (not from event store) -> returns denormalized view with product names and prices
3. CancelOrder -> loads aggregate from event store -> applies business rule (only pending orders) -> writes OrderCancelledEvent -> releases stock -> publishes to NATS -> projector updates Redis status to "cancelled"
4. Event store shows the full audit trail: version 1 = order.created, version 2 = order.cancelled

---

## Go Modules Downloaded

**`github.com/nats-io/nats.go`** -- v1.50.0

**Why:** Go client for NATS messaging system. Used for publishing domain events (orders.created, orders.cancelled) and subscribing to them. JetStream mode provides persistent streams with durable consumers, ensuring events aren't lost if a subscriber is temporarily offline. We chose NATS over Kafka because it's much simpler to operate, has a tiny Docker image (~20MB), and JetStream provides the persistence guarantees we need without Kafka's operational complexity.

**`github.com/redis/go-redis/v9`** -- v9.18.0

**Why:** Go client for Redis. Used as the CQRS read model store. The projector writes denormalized `OrderView` objects as JSON to Redis, and the query handler reads them. Redis is ideal here because: (1) read queries are simple key lookups (O(1)), (2) sorted sets enable listing orders by user sorted by time, (3) TTL-based expiration handles cache lifecycle. The read model is eventually consistent with the write model -- there's a brief delay between writing an event and the projector updating Redis.

### Indirect Dependencies

- `github.com/nats-io/nkeys` -- NATS authentication keys
- `github.com/nats-io/nuid` -- NATS unique ID generation
- `go.uber.org/atomic` -- Atomic operations used by NATS client

---

## Files Written

### Proto Definition

**`proto/order/v1/order.proto`**

**Why:** Defines two separate gRPC services in one file -- this is the visible manifestation of CQRS at the API level:
- `OrderCommandService` has `PlaceOrder` and `CancelOrder` -- these write events, they don't return the full order state
- `OrderQueryService` has `GetOrder` and `ListOrdersByUser` -- these read from the Redis projection, not from the event store
Both services run in the same process but could be split into separate deployments later for independent scaling (reads usually outnumber writes 10:1).

### Domain Model

**`internal/order/model/order.go`**

**Why:** Defines two separate models for the two sides of CQRS:
- `Order` -- the write-side aggregate state, reconstructed by replaying events
- `OrderView` -- the read-side denormalized projection stored in Redis as JSON. Contains pre-joined data (product name, total) so queries never need to join across services

**`internal/order/model/events.go`**

**Why:** Defines event types and their JSON payloads. Events are the source of truth in event sourcing -- the `event_store` table records every state change that ever happened. The payloads are serialized as JSON (not protobuf) for PostgreSQL JSONB storage and human-readable debugging.

### Aggregate Root

**`internal/order/aggregate/order.go`**

**Why:** The aggregate root is the core DDD pattern. It enforces business invariants:
- `PlaceOrder()` validates items aren't empty, calculates total, produces `OrderCreatedEvent`
- `Cancel()` checks status is "pending" (you can't cancel a paid order), produces `OrderCancelledEvent`
- `MarkPaid()` checks status is "pending" (for Phase 4), produces `OrderPaidEvent`

Events are tracked as `Changes` (uncommitted) until persisted. `LoadFromEvents()` rebuilds the aggregate by replaying stored events -- this is event sourcing: the current state is derived from the event history, not stored directly.

### Event Infrastructure

**`internal/order/event/store.go`**

**Why:** PostgreSQL-backed event store. `Append()` inserts events within a transaction with optimistic concurrency -- the `UNIQUE(aggregate_id, version)` constraint prevents concurrent writes from corrupting the event stream. If two commands try to modify the same order simultaneously, one fails with a unique violation.

**`internal/order/event/publisher.go`**

**Why:** Publishes domain events to NATS JetStream. Maps event types to NATS subjects: `order.created` -> `orders.created`, etc. Publishing is fire-and-forget after events are persisted -- even if NATS is temporarily down, the events are safely in PostgreSQL and can be replayed later.

**`internal/order/event/projector.go`**

**Why:** The projector is the bridge between write and read sides. It subscribes to `orders.*` on NATS with a durable consumer ("order-projector"), so it receives every event exactly once (even across restarts). For each event it:
1. Deserializes the payload
2. Creates or updates the `OrderView` in Redis
3. Acknowledges the message

This is the eventual consistency mechanism -- there's a small delay (typically <100ms) between writing an event and the read model being updated.

### Repositories

**`internal/order/repository/read_repository.go`**

**Why:** Interface for the read side. Only 3 methods: `GetByID`, `GetByUserID`, `Save`. The read side is intentionally simple -- complex queries are handled by pre-computing the answer in the projector.

**`internal/order/repository/redis_read.go`**

**Why:** Redis implementation of the read repository. Uses two data structures:
- `SET order:{id}` -- JSON blob of the full `OrderView` (with 24h TTL)
- `ZADD user_orders:{user_id}` -- sorted set of order IDs scored by timestamp, enabling `ListOrdersByUser` sorted by recency

Uses Redis pipelines to atomically update both the view and the index in one round trip.

### Command Handler (Write Side)

**`internal/order/command/handler.go`**

**Why:** Orchestrates the write side. `PlaceOrder()`:
1. Fetches product details via gRPC (enriches items with names and prices)
2. Reserves stock via gRPC (atomic decrement in product-service)
3. Creates aggregate and produces events
4. Persists events to PostgreSQL event store
5. Publishes events to NATS
6. On any failure, rolls back stock reservations

`CancelOrder()` loads the aggregate from the event store (replays events), applies the cancel business rule, persists the new event, releases stock, and publishes.

### Query Handler (Read Side)

**`internal/order/query/handler.go`**

**Why:** Reads from Redis. Thin layer that delegates to the read repository. The query side never touches PostgreSQL or NATS -- it only reads pre-computed views from Redis.

### gRPC Server

**`internal/order/server.go`**

**Why:** Implements both `OrderCommandServiceServer` and `OrderQueryServiceServer`. Two separate structs (`CommandServer` and `QueryServer`) each with their own handler -- making the CQRS split explicit in the code structure.

### Service Entrypoint

**`cmd/order-service/main.go`**

**Why:** Wires the most complex service: connects to PostgreSQL (event store), Redis (read model), and NATS (event bus). Creates the JetStream stream "ORDERS" with subject pattern `orders.*`. Starts the projector in a background goroutine. Registers both command and query gRPC servers.

### Database Migration

**`migrations/order/001_create_event_store.up.sql`**

**Why:** Creates the event store table with `UNIQUE(aggregate_id, version)` for optimistic concurrency. The `JSONB` payload column stores event data in a queryable format. Indexes on `(aggregate_id, version)` for fast aggregate loading and `(event_type, created_at)` for event replay.

### Modified Files

**`internal/gateway/server.go`** -- Registers both `OrderCommandService` and `OrderQueryService` handlers. Two registrations for one service because CQRS exposes separate gRPC services for commands and queries.

**`cmd/gateway/main.go`** -- Added `ORDER_SERVICE_ADDR` config.

**`pkg/auth/interceptor.go`** -- Added `GetOrder` and `ListOrdersByUser` as public (browsable without auth). Added `ReserveStock` and `ReleaseStock` as public to allow internal service-to-service calls without JWT (in production, use mTLS instead).

**`docker-compose.yml`** -- Added `postgres-order` (port 5435), `redis` (port 6379), and `nats` with JetStream (port 4222 client, 8222 monitoring).

**`Makefile`** -- Added `run-order` target and order-service to build list.

---

## Verified End-to-End Results

| Test | Result |
|------|--------|
| Place order (2x Headphones @ $199) | `totalCents: 39800` (correct: 2 x 19900) |
| Product name enriched | `productName: "Wireless Headphones"` |
| Stock reserved | 20 -> 18 |
| Read model in Redis | Full order view with items, prices, status |
| List orders by user | Returns all user's orders |
| Cancel order | Status changed to "cancelled" |
| Read model updated after cancel | `status: "cancelled"`, `updatedAt` changed |
| Stock released after cancel | 18 -> 20 |
| Event store audit trail | v1=order.created, v2=order.cancelled |
