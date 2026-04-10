# Go gRPC-HTTP Microservices

A production-grade order management system built with Go, demonstrating 8 microservice design patterns across 16 Docker containers.

**[Live Architecture Diagram](https://parthai.github.io/go-grpc-http/architecture-v2.html)** — Click any component to explore how it works.

---

## Architecture

```
Browser (:3000)
    |
  Nginx (React 19 + TypeScript + Tailwind CSS)
    |
  API Gateway (:8080) — gRPC-Gateway (REST <-> gRPC)
    |
    |--- User Service (:50051)      -> PostgreSQL (userdb)
    |--- Product Service (:50052)   -> PostgreSQL (productdb)
    |--- Order Service (:50053)     -> PostgreSQL (event_store) + Redis (read model)
    |--- Payment Service (:50054)   -> PostgreSQL (paymentdb)
    |
  NATS JetStream (:4222) — Event Bus (orders.*, payments.*)
    |
  Bridge Service — NATS -> RabbitMQ forwarder
    |
  RabbitMQ (:5672) — Topic Exchange + Durable Queue
    |
  Notification Service -> PostgreSQL (notificationdb) + SMTP (Mailtrap)
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.25 |
| RPC | gRPC + Protocol Buffers |
| REST | gRPC-Gateway (auto-generated from proto) |
| Proto Tooling | Buf |
| Database | PostgreSQL 16 (5 instances, database-per-service) |
| Cache | Redis 7 (CQRS read model) |
| Event Bus | NATS JetStream (durable consumers, at-least-once) |
| Message Queue | RabbitMQ 3 (notification pipeline) |
| Auth | Per-user JWT (HMAC-SHA256, instant revocation) |
| Frontend | React 19, TypeScript, Tailwind CSS 4, Vite 8 |
| Web Server | Nginx 1.27 |
| Containers | Docker + Docker Compose (16 containers) |

## Design Patterns

| Pattern | Implementation |
|---------|---------------|
| **API Gateway** | gRPC-Gateway translates REST to gRPC. Single entry point for all clients. |
| **Database per Service** | 5 PostgreSQL instances. Each service owns its schema. No shared tables. |
| **CQRS** | Order writes go to event store (PostgreSQL). Reads served from Redis. |
| **Event Sourcing** | Order state derived from replaying immutable events. Full audit trail. |
| **Event-Driven** | NATS JetStream pub/sub decouples services. Durable consumer groups. |
| **Per-User JWT** | Unique signing secret per user. Change secret = instant revocation. No blacklist. |
| **Repository Pattern** | Interface-based data access. Swappable backends for testing. |
| **Health Check Protocol** | gRPC Health v1 on all services. Docker healthchecks for startup ordering. |

## Services

| Service | Port | Role |
|---------|------|------|
| **API Gateway** | :8080 | REST-to-gRPC reverse proxy, CORS middleware |
| **User Service** | :50051 | Registration, login, JWT issuance, profile management |
| **Product Service** | :50052 | Product catalog, inventory, atomic stock reservation |
| **Order Service** | :50053 | CQRS + Event Sourcing, command/query split, projector |
| **Payment Service** | :50054 | Event-driven payment processing (auto-triggers on order) |
| **Bridge Service** | — | NATS to RabbitMQ event forwarder (separate container) |
| **Notification Service** | — | RabbitMQ consumer, SMTP email sender, audit log |
| **Frontend** | :3000 | React SPA served by Nginx |

## Quick Start

### Prerequisites

- Docker + Docker Compose
- Go 1.25+ (for local development)
- Node.js 22+ (for frontend development)

### Start Everything

```bash
docker compose up -d --build
```

This starts all 16 containers:
- **Frontend:** http://localhost:3000
- **API Gateway:** http://localhost:8080
- **RabbitMQ Management:** http://localhost:15672 (guest/guest)
- **NATS Monitor:** http://localhost:8222

### Local Development

```bash
# Start infrastructure only (databases, Redis, NATS, RabbitMQ)
make docker-up

# Run services individually (separate terminals)
make run-user
make run-product
make run-order
make run-payment
make run-notification
make run-bridge
make run-gateway

# Frontend dev server
cd frontend && npm run dev
```

### Proto Code Generation

```bash
# Generate Go stubs + gateway + OpenAPI from proto files
make proto

# Lint proto files
make proto-lint
```

### Build & Test

```bash
# Build all services
make build

# Run tests
make test
```

## API Endpoints

### Authentication
```
POST /api/v1/users/register    — Create account
POST /api/v1/users/login       — Get JWT token
GET  /api/v1/users/{id}        — Get profile (auth required)
PUT  /api/v1/users/{id}        — Update profile (auth required)
```

### Products
```
POST /api/v1/products          — Create product (auth required)
GET  /api/v1/products          — List products
GET  /api/v1/products/{id}     — Get product details
PUT  /api/v1/products/{id}     — Update product (auth required)
```

### Orders
```
POST /api/v1/orders                    — Place order (auth required)
POST /api/v1/orders/{id}/cancel        — Cancel order (auth required)
GET  /api/v1/orders/{id}               — Get order details
GET  /api/v1/users/{id}/orders         — List user's orders
```

### Payments
```
GET  /api/v1/payments/{id}             — Get payment details
GET  /api/v1/orders/{id}/payments      — List payments for order
POST /api/v1/payments/{id}/refund      — Refund payment (auth required)
```

### Health
```
GET  /healthz                          — Gateway health check
```

## Order Flow

1. User places order via `POST /api/v1/orders`
2. Order Service reserves stock via sync gRPC to Product Service
3. Order aggregate produces `OrderCreatedEvent`, persisted to event store
4. Event published to NATS `orders.created`
5. **Parallel async processing:**
   - Projector writes order to Redis (status: pending)
   - Payment Service creates payment, publishes `payments.completed`
   - Bridge forwards to RabbitMQ, Notification Service sends email
6. Order Service receives `payments.completed`, produces `OrderPaidEvent`
7. Projector updates Redis (status: paid)
8. Client polls — sees order is **paid**

## Project Structure

```
cmd/                        — Service entrypoints
  gateway/                  — API Gateway
  user-service/             — User Service
  product-service/          — Product Service
  order-service/            — Order Service
  payment-service/          — Payment Service
  notification-service/     — Notification Service
  bridge-service/           — NATS-to-RabbitMQ Bridge
internal/                   — Private business logic per service
  user/                     — Repository, service, gRPC server
  product/                  — Repository, service, gRPC server
  order/                    — Aggregate, command handler, query handler,
                              event store, projector, payment subscriber
  payment/                  — Repository, event subscriber/publisher
  notification/             — RabbitMQ subscriber, email sender, bridge
  gateway/                  — HTTP server setup
pkg/                        — Shared packages
  auth/                     — JWT manager, auth interceptor, token resolver
  database/                 — PostgreSQL connection pool helper
  messaging/                — RabbitMQ connection helper
  interceptors/             — Logging, recovery interceptors
  observability/            — Structured logger (slog)
  health/                   — gRPC health check server
proto/                      — Protocol Buffer definitions
  user/v1/                  — UserService (5 RPCs)
  product/v1/               — ProductService (7 RPCs)
  order/v1/                 — OrderCommandService (2) + OrderQueryService (2)
  payment/v1/               — PaymentService (3 RPCs)
  common/v1/                — Shared types (Pagination, Money)
gen/                        — Auto-generated code (do not edit)
  go/                       — Go stubs, gRPC clients, gateway handlers
  openapiv2/                — Swagger/OpenAPI specs
migrations/                 — SQL migration files
deployments/docker/         — Dockerfiles (multi-stage builds)
frontend/                   — React 19 + TypeScript + Tailwind CSS
docs/                       — Architecture diagram, Q&A, setup guides
```

## Docker Containers

### Infrastructure (8)

| Container | Image | Port | Purpose |
|-----------|-------|------|---------|
| postgres-user | `postgres:16-alpine` | :5433 | User accounts, auth tokens |
| postgres-product | `postgres:16-alpine` | :5434 | Product catalog, inventory |
| postgres-order | `postgres:16-alpine` | :5435 | Event store (append-only) |
| postgres-payment | `postgres:16-alpine` | :5436 | Payment records |
| postgres-notification | `postgres:16-alpine` | :5437 | Notification audit log |
| redis | `redis:7-alpine` | :6379 | CQRS read model |
| nats | `nats:2-alpine` | :4222 | Event bus (JetStream) |
| rabbitmq | `rabbitmq:3-management-alpine` | :5672 :15672 | Notification queue |

### Application (8)

| Container | Dockerfile | Port | Purpose |
|-----------|-----------|------|---------|
| frontend | `frontend/Dockerfile` | :3000 | React SPA (Node build + Nginx serve) |
| gateway | `deployments/docker/gateway.Dockerfile` | :8080 | REST-to-gRPC proxy |
| user-service | `deployments/docker/user-service.Dockerfile` | :50051 | Auth, JWT, profiles |
| product-service | `deployments/docker/product-service.Dockerfile` | :50052 | Catalog, stock |
| order-service | `deployments/docker/order-service.Dockerfile` | :50053 | CQRS + Event Sourcing |
| payment-service | `deployments/docker/payment-service.Dockerfile` | :50054 | Event-driven payments |
| bridge-service | `deployments/docker/bridge-service.Dockerfile` | — | NATS to RabbitMQ forwarder |
| notification-service | `deployments/docker/notification-service.Dockerfile` | — | Email + audit log |

All Go services use **multi-stage builds**: `golang:1.25-alpine` (build) -> `alpine:3.20` (runtime, ~35MB).

## Environment Variables

Copy `.env.example` and configure:

```bash
# Database (each service has its own)
DATABASE_URL=postgres://postgres:postgres@localhost:5433/userdb?sslmode=disable

# NATS
NATS_URL=nats://localhost:4222

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# SMTP (optional — leave empty for mock email logging)
SMTP_HOST=sandbox.smtp.mailtrap.io
SMTP_PORT=587
SMTP_USER=your-username
SMTP_PASSWORD=your-password
SMTP_FROM=notifications@example.com
NOTIFY_EMAIL=recipient@example.com
```

## License

MIT
