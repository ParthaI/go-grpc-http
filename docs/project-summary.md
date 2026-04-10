# Go gRPC/HTTP Microservice Project - Complete Summary

## Overview

A production-like **order management system** built with Go microservices using gRPC for inter-service communication, grpc-gateway for REST API exposure, and a React frontend served via Nginx. Demonstrates 8 microservice design patterns across 6 backend services with a fully containerized deployment.

## Architecture

```
                Browser / Client
                      |
                      | :3000
                      v
              +---------------+
              |    NGINX      |
              | (React SPA +  |
              |  API proxy)   |
              +-------+-------+
                      |
              /api/*  | proxy
                      v
              +---------------+
              |   GATEWAY     | :8080
              |  (grpc-gw)    |
              +--+--+--+--+--+
                 |  |  |  |     gRPC (internal)
          +------+  |  |  +------+
          v         v  v         v
     +--------+ +------+ +--------+
     | USER   | | PROD | | PAYMENT|
     | :50051 | | :50052 | :50054 |
     +---+----+ +--+---+ +---+----+
         |         |          |
      [PG:user] [PG:prod]  [PG:pay]
                   |
              gRPC | (stock)
                   v
            +-----------+
            |   ORDER   | :50053
            |  CQRS +   |
            |  Events   |
            +--+-----+--+
               |     |
        [PG:order] [Redis]
               |
          NATS JetStream
            /       \
      PAYMENT    NOTIFICATION
      SERVICE      SERVICE
     (events)   (events only)
```

## Frontend

### Tech Stack

- **Vite** -- Build tool and dev server
- **React 19** + **TypeScript** -- UI framework
- **Tailwind CSS** -- Utility-first styling
- **React Router** -- Client-side routing
- **Nginx 1.27** -- Production server (Docker)

### Project Structure (Feature-Based)

```
frontend/src/
|-- types/                    # Shared TypeScript interfaces
|   |-- api.ts                # All API request/response types
|
|-- lib/                      # Utilities
|   |-- api-client.ts         # Centralized HTTP client (auto-attaches JWT)
|   |-- format.ts             # Currency, date, status formatting
|
|-- hooks/                    # Custom React hooks
|   |-- useAsync.ts           # Generic async state management
|
|-- context/                  # React Context providers
|   |-- AuthContext.tsx        # Auth state (token, userId, login/logout)
|
|-- layouts/                  # Page layouts
|   |-- AppLayout.tsx         # Nav bar + Outlet (shared across all pages)
|
|-- components/ui/            # Reusable UI kit
|   |-- Button.tsx            # Primary/secondary/danger variants + loading
|   |-- Input.tsx             # Labeled input with error state
|   |-- Card.tsx              # Content card with optional title
|   |-- Badge.tsx             # Status badge (pending/paid/cancelled)
|   |-- Alert.tsx             # Success/error/info alerts
|   |-- index.ts              # Barrel exports
|
|-- features/                 # Domain modules (self-contained)
|   |-- auth/
|   |   |-- api.ts            # register(), login(), getUser()
|   |   |-- components/
|   |   |   |-- LoginPage.tsx
|   |   |   |-- RegisterPage.tsx
|   |   |-- index.ts
|   |
|   |-- dashboard/
|   |   |-- components/
|   |   |   |-- DashboardPage.tsx  # Stats, recent orders, account
|   |   |-- index.ts
|   |
|   |-- products/
|   |   |-- api.ts            # list(), get(), create()
|   |   |-- components/
|   |   |   |-- ProductsPage.tsx   # Product grid + create form
|   |   |-- index.ts
|   |
|   |-- orders/
|   |   |-- api.ts            # get(), listByUser(), place(), cancel()
|   |   |-- components/
|   |   |   |-- OrdersPage.tsx      # Order list + place order form
|   |   |   |-- OrderDetailPage.tsx # Order details + payment table
|   |   |-- index.ts
|   |
|   |-- payments/
|       |-- api.ts            # get(), listByOrder(), refund()
|
|-- App.tsx                   # Router configuration
|-- main.tsx                  # Entry point
|-- index.css                 # Tailwind import
```

### Pages

- **Login** -- Email/password form, stores JWT in localStorage
- **Register** -- Creates account, shows auth_token once
- **Dashboard** -- Stats cards (products, orders, total spent), recent orders, account info
- **Products** -- Grid of product cards with prices/stock, create product form
- **Orders** -- List with status badges, place order form with product dropdown
- **Order Detail** -- Order info, items breakdown, payment table, cancel/refresh actions

### Docker (Nginx)

```
frontend/
|-- Dockerfile              # Multi-stage: Node 22 builds, Nginx 1.27 serves
|-- nginx/default.conf      # SPA fallback + API reverse proxy + gzip + asset caching
|-- .dockerignore
```

Nginx configuration:
- **`/`** -- Serves React SPA (`try_files $uri /index.html`)
- **`/api/*`** -- Reverse proxies to `gateway:8080`
- **`/healthz`** -- Reverse proxies to `gateway:8080`
- **`/assets/*`** -- 1-year cache with immutable header
- **Gzip** enabled for text, CSS, JSON, JS

## Backend Services

### 1. API Gateway (`:8080`)

- **Role:** Single entry point for all external clients
- **Pattern:** grpc-gateway (auto-generates REST from proto annotations)
- **Features:** CORS middleware, `/healthz` endpoint
- **Files:** `cmd/gateway/main.go`, `internal/gateway/server.go`

### 2. User Service (`:50051`)

- **Role:** Registration, login, JWT issuance, profile management
- **Pattern:** Standard CRUD
- **DB:** PostgreSQL (port 5433)
- **Auth:** Per-user JWT signing with base64-encoded `auth_token`
- **RPCs:** Register, Login, GetUser, UpdateUser, GetAuthToken (internal)
- **Files:** `cmd/user-service/`, `internal/user/`

### 3. Product Service (`:50052`)

- **Role:** Product catalog, inventory, pricing, stock reservation
- **Pattern:** Standard CRUD + atomic stock operations
- **DB:** PostgreSQL (port 5434)
- **RPCs:** CreateProduct, GetProduct, ListProducts, UpdateProduct, UpdateInventory, ReserveStock (internal), ReleaseStock (internal)
- **Files:** `cmd/product-service/`, `internal/product/`

### 4. Order Service (`:50053`)

- **Role:** Order placement, lifecycle management
- **Pattern:** CQRS + Event Sourcing
- **DB:** PostgreSQL (event store, port 5435) + Redis (read model)
- **Write side:** Command handlers -> aggregate -> event store -> NATS publish
- **Read side:** NATS projector -> Redis denormalized views -> query handlers
- **RPCs:** PlaceOrder, CancelOrder (commands), GetOrder, ListOrdersByUser (queries)
- **Files:** `cmd/order-service/`, `internal/order/`

### 5. Payment Service (`:50054`)

- **Role:** Payment processing, refunds
- **Pattern:** Event-driven consumer + gRPC server
- **DB:** PostgreSQL (port 5436)
- **Events:** Subscribes to `orders.created`, publishes `payments.completed`/`payments.failed`
- **RPCs:** GetPayment, ListPaymentsByOrder, RefundPayment
- **Files:** `cmd/payment-service/`, `internal/payment/`

### 6. Notification Service (no port)

- **Role:** Email/SMS notifications, audit trail
- **Pattern:** Pure event consumer (no REST endpoints)
- **DB:** PostgreSQL (port 5437)
- **Events:** Subscribes to `orders.*` and `payments.*`, sends mock emails, logs all notifications
- **Files:** `cmd/notification-service/`, `internal/notification/`

## Design Patterns Implemented

1. **API Gateway** -- grpc-gateway auto-generates REST endpoints from proto HTTP annotations
2. **CQRS** -- Separate `OrderCommandService` and `OrderQueryService` gRPC services with different data stores
3. **Event Sourcing** -- Order state derived from replaying events in `event_store` table, not stored directly
4. **Event-Driven Architecture** -- NATS JetStream decouples services via async events with at-least-once delivery
5. **Database per Service** -- 5 isolated PostgreSQL instances, no shared databases
6. **Per-User JWT Authentication** -- Each user's `auth_token` (32-byte base64) is the JWT signing secret
7. **Health Checks** -- gRPC `Health/Check` protocol on all services + HTTP `/healthz` on gateway
8. **Aggregate Root (DDD)** -- Order aggregate enforces business invariants before producing events

## Authentication Flow

```
Register -> generates 32-byte random auth_token (base64)
Login    -> signs JWT with user's auth_token (per-user secret)

Verification (3-step):
  1. Parse JWT unverified -> extract user_id
  2. Resolve auth_token:
       user-service: DBTokenResolver (direct SQL query)
       other services: GRPCTokenResolver (calls GetAuthToken RPC)
  3. Verify JWT signature with user's auth_token

Token invalidation: change auth_token in DB -> all existing JWTs instantly invalid
```

## Event Flow (Order Placement)

```
1. Browser -> Nginx :3000 -> proxy /api/* -> Gateway :8080 -> gRPC PlaceOrder -> Order Service
2. Order Service -> gRPC GetProduct (enrich prices) -> Product Service
3. Order Service -> gRPC ReserveStock (atomic) -> Product Service
4. Order Service -> write OrderCreatedEvent to event_store (PostgreSQL)
5. Order Service -> publish "orders.created" to NATS
6. [Async] Projector -> build Redis read model (status: pending)
7. [Async] Payment Service -> process payment -> publish "payments.completed"
8. [Async] Notification Service -> send "Order Confirmed" email
9. [Async] Order Service -> subscribe "payments.completed" -> write OrderPaidEvent
10. [Async] Projector -> update Redis (status: paid, paymentId set)
11. [Async] Notification Service -> send "Payment Successful" + "Payment Received"
```

## NATS Event Catalog

- **`orders.created`** -- Published by order-service, consumed by payment-svc, notification-svc, order projector
- **`orders.paid`** -- Published by order-service (after payment), consumed by notification-svc, order projector
- **`orders.cancelled`** -- Published by order-service, consumed by notification-svc, product-svc (stock release)
- **`payments.completed`** -- Published by payment-service, consumed by order-service
- **`payments.failed`** -- Published by payment-service, consumed by order-service

## Technology Stack

### Backend

- **gRPC:** `google.golang.org/grpc` v1.80.0
- **REST Gateway:** `github.com/grpc-ecosystem/grpc-gateway/v2` v2.28.0
- **Proto management:** `buf` CLI v1.67.0
- **PostgreSQL:** `github.com/jackc/pgx/v5` v5.9.1
- **Redis:** `github.com/redis/go-redis/v9` v9.18.0
- **Messaging:** `github.com/nats-io/nats.go` v1.50.0 (JetStream)
- **JWT:** `github.com/golang-jwt/jwt/v5` v5.3.1 (per-user signing)
- **Password hashing:** `golang.org/x/crypto` (bcrypt)
- **UUID:** `github.com/google/uuid` v1.6.0
- **Logging:** `log/slog` (stdlib, structured JSON)

### Frontend

- **React:** v19 + TypeScript
- **Vite:** v8 (build + dev server)
- **Tailwind CSS:** v4
- **React Router:** v7

### Infrastructure

- **Docker:** Multi-stage builds (Go 1.25 + Alpine, Node 22 + Nginx 1.27)
- **PostgreSQL:** 16 Alpine (5 instances)
- **Redis:** 7 Alpine
- **NATS:** 2 Alpine (JetStream enabled)
- **Nginx:** 1.27 Alpine (frontend reverse proxy)

## Full Project Structure

```
go-grpc-http/
|-- cmd/                            # 6 backend service entrypoints
|   |-- gateway/main.go
|   |-- user-service/main.go
|   |-- product-service/main.go
|   |-- order-service/main.go
|   |-- payment-service/main.go
|   |-- notification-service/main.go
|
|-- proto/                          # 5 proto definitions
|   |-- common/v1/common.proto
|   |-- user/v1/user.proto
|   |-- product/v1/product.proto
|   |-- order/v1/order.proto
|   |-- payment/v1/payment.proto
|
|-- gen/                            # 18 generated files (Go stubs + Swagger)
|   |-- go/{common,user,product,order,payment}/v1/
|   |-- openapiv2/
|
|-- internal/                       # Private backend service code
|   |-- gateway/                    # HTTP mux + CORS + /healthz
|   |-- user/                       # model, repository, service, server
|   |-- product/                    # model, repository, service, server
|   |-- order/                      # CQRS: aggregate, command, query, event, repository
|   |-- payment/                    # model, repository, event, server
|   |-- notification/               # model, repository, sender, event
|
|-- pkg/                            # Shared Go libraries
|   |-- auth/                       # JWT, interceptor, DBTokenResolver, GRPCTokenResolver
|   |-- database/                   # PostgreSQL connection pool
|   |-- errors/                     # Domain-to-gRPC error mapping
|   |-- health/                     # gRPC health check server
|   |-- interceptors/               # Logging, recovery interceptors
|   |-- observability/              # Structured JSON logger
|
|-- migrations/                     # SQL migrations per service
|   |-- user/                       # 001_create_users, 002_add_auth_token
|   |-- product/                    # 001_create_products
|   |-- order/                      # 001_create_event_store
|   |-- payment/                    # 001_create_payments
|   |-- notification/               # 001_create_notification_log
|
|-- frontend/                       # React application
|   |-- src/
|   |   |-- types/                  # TypeScript API interfaces
|   |   |-- lib/                    # API client, formatting utilities
|   |   |-- hooks/                  # Custom React hooks
|   |   |-- context/                # Auth context provider
|   |   |-- layouts/                # App layout with nav bar
|   |   |-- components/ui/          # Reusable UI kit (Button, Input, Card, Badge, Alert)
|   |   |-- features/              # Domain modules
|   |   |   |-- auth/              # Login, Register pages + API
|   |   |   |-- dashboard/         # Dashboard page
|   |   |   |-- products/          # Products page + API
|   |   |   |-- orders/            # Orders page, Order detail + API
|   |   |   |-- payments/          # Payment API
|   |   |-- App.tsx                # Router
|   |   |-- main.tsx               # Entry point
|   |-- nginx/default.conf         # Nginx config (SPA + API proxy)
|   |-- Dockerfile                 # Multi-stage: Node build + Nginx serve
|   |-- vite.config.ts             # Vite + Tailwind + dev proxy
|   |-- package.json
|
|-- deployments/docker/             # 6 backend Dockerfiles
|-- docs/                           # Architecture + phase setup docs
|-- docker-compose.yml              # 14 containers
|-- Makefile                        # Build, test, run, docker targets
|-- buf.yaml / buf.gen.yaml         # Proto tooling config
```

## Docker Containers (14 total)

```
CONTAINER                           IMAGE                  PORT      SIZE
go-grpc-http-frontend               Nginx + React SPA      :3000     50MB
go-grpc-http-gateway                Go binary              :8080     27MB
go-grpc-http-user-service           Go binary              :50051    32MB
go-grpc-http-product-service        Go binary              :50052    32MB
go-grpc-http-order-service          Go binary              :50053    40MB
go-grpc-http-payment-service        Go binary              :50054    34MB
go-grpc-http-notification-service   Go binary              -         24MB
go-grpc-http-postgres-user          PostgreSQL 16          :5433     -
go-grpc-http-postgres-product       PostgreSQL 16          :5434     -
go-grpc-http-postgres-order         PostgreSQL 16          :5435     -
go-grpc-http-postgres-payment       PostgreSQL 16          :5436     -
go-grpc-http-postgres-notification  PostgreSQL 16          :5437     -
go-grpc-http-redis                  Redis 7                :6379     -
go-grpc-http-nats                   NATS 2 (JetStream)     :4222     -
```

## Project Stats

- **Backend files:** 84 hand-written + 18 generated = 102
- **Frontend files:** 22 (TypeScript + config)
- **Total project files:** ~124
- **Backend services:** 6
- **Proto files:** 5
- **gRPC RPCs:** 16 (11 public REST + 5 internal gRPC-only)
- **NATS events:** 5 event types
- **Docker containers:** 14 (5 PostgreSQL + 1 Redis + 1 NATS + 6 app + 1 frontend)
- **Database tables:** 6 (users, products, event_store, payments, notification_log + Redis keys)
- **Frontend pages:** 6 (login, register, dashboard, products, orders, order detail)

## How to Run

### Full Docker (recommended)

```bash
# Build and start all 14 containers
docker compose up -d --build

# Access the application
# Frontend (React + Nginx):  http://localhost:3000
# API Gateway (direct):      http://localhost:8080
# NATS Monitoring:           http://localhost:8222

# Stop everything
docker compose down -v
```

### Local Development

```bash
# Start infrastructure only (databases, Redis, NATS)
make docker-up

# Start backend services (in separate terminals)
make run-user
make run-product
make run-order
make run-payment
make run-notification
HTTP_PORT=9090 make run-gateway

# Start frontend dev server (hot reload, proxy to :9090)
cd frontend && npm run dev
# Frontend: http://localhost:3000
```

### Quick Test (via curl)

```bash
# Register
curl -X POST http://localhost:3000/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret","first_name":"Test","last_name":"User"}'

# Login (get JWT)
curl -X POST http://localhost:3000/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret"}'

# Create product (with JWT)
curl -X POST http://localhost:3000/api/v1/products \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Widget","price_cents":999,"currency":"USD","stock_quantity":10,"sku":"WDG-001"}'

# Place order (triggers full async chain: stock -> payment -> notification)
curl -X POST http://localhost:3000/api/v1/orders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"<user_id>","items":[{"product_id":"<product_id>","quantity":2}]}'

# Check order (auto-updates to "paid" after ~3s)
curl http://localhost:3000/api/v1/orders/<order_id>

# Health check
curl http://localhost:3000/healthz
```

## API Reference

### Public Endpoints (no auth required)

```
GET  /healthz                              # Gateway health check
GET  /api/v1/products                      # List products
GET  /api/v1/products/{product_id}         # Get product
GET  /api/v1/orders/{order_id}             # Get order
GET  /api/v1/users/{user_id}/orders        # List user's orders
GET  /api/v1/payments/{payment_id}         # Get payment
GET  /api/v1/orders/{order_id}/payments    # List order's payments
POST /api/v1/users/register                # Register (returns auth_token)
POST /api/v1/users/login                   # Login (returns JWT)
```

### Authenticated Endpoints (requires `Authorization: Bearer <jwt>`)

```
GET  /api/v1/users/{user_id}                       # Get user profile
PUT  /api/v1/users/{user_id}                        # Update user profile
POST /api/v1/products                               # Create product
PUT  /api/v1/products/{product_id}                  # Update product
PUT  /api/v1/products/{product_id}/inventory        # Update stock
POST /api/v1/orders                                 # Place order
POST /api/v1/orders/{order_id}/cancel               # Cancel order
POST /api/v1/payments/{payment_id}/refund           # Refund payment
```

### Internal gRPC-Only RPCs (service-to-service, no REST)

```
product.v1.ProductService/ReserveStock     # Called by order-service
product.v1.ProductService/ReleaseStock     # Called by order-service
user.v1.UserService/GetAuthToken           # Called by product/order/payment for JWT verification
```

## Implementation Phases

- **Phase 1:** Foundation -- Go module, buf, user-service, gateway, shared libraries
- **Phase 2:** Product service -- catalog, inventory, atomic stock operations
- **Phase 3:** CQRS order service -- event sourcing, aggregate, NATS projector, Redis read model
- **Phase 4:** Payment + notification -- async event chain, auto-payment processing, mock emails
- **Phase 5:** Polish -- Dockerfiles, health checks, full docker-compose, error helpers
- **Phase 6:** Frontend -- React + TypeScript + Tailwind, Nginx Docker, feature-based architecture

See `docs/phase{1-5}-setup.md` for detailed setup references with command explanations and rationale.
