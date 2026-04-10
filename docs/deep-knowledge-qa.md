# Go gRPC-HTTP Microservices — Deep Knowledge Q&A

A comprehensive question bank covering every technology, pattern, and design decision in this project. Organized by topic — from fundamentals to advanced architecture.

---

## Table of Contents

1. [Project Overview & Architecture](#1-project-overview--architecture)
2. [gRPC Fundamentals](#2-grpc-fundamentals)
3. [Protocol Buffers (Protobuf)](#3-protocol-buffers-protobuf)
4. [gRPC-Gateway (REST ↔ gRPC)](#4-grpc-gateway-rest--grpc)
5. [Buf Build Tool](#5-buf-build-tool)
6. [Go Language & Patterns](#6-go-language--patterns)
7. [PostgreSQL & Database Design](#7-postgresql--database-design)
8. [Authentication & JWT](#8-authentication--jwt)
9. [CQRS (Command Query Responsibility Segregation)](#9-cqrs-command-query-responsibility-segregation)
10. [Event Sourcing](#10-event-sourcing)
11. [NATS JetStream & Event-Driven Architecture](#11-nats-jetstream--event-driven-architecture)
12. [Redis](#12-redis)
13. [gRPC Interceptors & Middleware](#13-grpc-interceptors--middleware)
14. [Observability & Logging](#14-observability--logging)
15. [Docker & Containerization](#15-docker--containerization)
16. [Frontend (React + TypeScript)](#16-frontend-react--typescript)
17. [Microservice Communication Patterns](#17-microservice-communication-patterns)
18. [Error Handling & Resilience](#18-error-handling--resilience)
19. [Concurrency & Performance](#19-concurrency--performance)
20. [Testing Strategies](#20-testing-strategies)
21. [Scenario-Based / System Design Questions](#21-scenario-based--system-design-questions)

---

## 1. Project Overview & Architecture

### Q: What is this project and what problem does it solve?

**A:** This is a production-grade order management microservice system built with Go. It models an e-commerce workflow — users register/login, browse products, place orders, make payments, and receive notifications. It demonstrates 8 advanced microservice patterns:

1. **API Gateway Pattern** (gRPC-Gateway)
2. **Microservices** with database-per-service
3. **CQRS** (Command Query Responsibility Segregation)
4. **Event Sourcing**
5. **Event-Driven Architecture** (NATS JetStream)
6. **Per-User JWT Authentication**
7. **Repository Pattern**
8. **Health Check Protocol**

### Q: How many services exist and what does each one do?

**A:** There are 6 backend services + 1 frontend:

| Service | Port | Role |
|---------|------|------|
| **API Gateway** | `:8080` | REST-to-gRPC reverse proxy, single external entry point |
| **User Service** | `:50051` | Registration, login, JWT issuance, profile management |
| **Product Service** | `:50052` | Product catalog, inventory, stock reservation |
| **Order Service** | `:50053` | Order placement/cancellation, CQRS + Event Sourcing |
| **Payment Service** | `:50054` | Payment processing (event-driven, no manual trigger) |
| **Notification Service** | No port | Event consumer only — mock email/SMS, audit log |
| **Frontend** | `:3000` | React SPA served via Nginx |

### Q: Draw the complete request flow for "User places an order."

**A:**
```
Browser → Nginx(:3000) → API Gateway(:8080) → Order Service(:50053)
                                                    │
                                    ┌───────────────┼───────────────┐
                                    ▼               ▼               ▼
                            Product Service   PostgreSQL      NATS JetStream
                            (GetProduct,      (event_store)   (orders.created)
                             ReserveStock)                         │
                                                    ┌──────────────┼──────────────┐
                                                    ▼              ▼              ▼
                                              Projector     Payment Svc    Notification Svc
                                              (Redis write)  (process $)   (mock email)
                                                              │
                                                              ▼
                                                        NATS (payments.completed)
                                                              │
                                                    ┌─────────┼─────────┐
                                                    ▼                   ▼
                                              Order Svc            Notification Svc
                                              (OrderPaidEvent)     (email: "paid")
                                              → Projector
                                              (Redis: status=paid)
```

**Timeline:**
1. **T+0ms:** Browser sends `POST /api/v1/orders` to Nginx
2. **T+10ms:** Gateway translates REST→gRPC, calls `OrderCommandService.PlaceOrder`
3. **T+20ms:** Order service enriches items via `ProductService.GetProduct`, reserves stock via `ProductService.ReserveStock`
4. **T+22ms:** Order aggregate produces `OrderCreatedEvent`, appended to PostgreSQL event_store
5. **T+25ms:** Event published to NATS `orders.created`, response sent to client (status: `pending`)
6. **T+30ms:** (Async) Projector writes denormalized order to Redis
7. **T+35ms:** (Async) Payment service creates payment record, simulates processing, publishes `payments.completed`
8. **T+40ms:** (Async) Notification service logs "Order Confirmed" email
9. **T+50ms:** (Async) Order service receives `payments.completed`, produces `OrderPaidEvent`
10. **T+55ms:** (Async) Projector updates Redis (status: `paid`, payment_id set)
11. **T+60ms:** Client polls `GET /api/v1/orders/{id}` → reads from Redis → sees `paid`

### Q: What is the technology stack used?

**A:**

| Layer | Technology | Version |
|-------|-----------|---------|
| Language | Go | 1.25.0 |
| RPC Framework | gRPC | 1.80.0 |
| Serialization | Protocol Buffers | 1.36.11 |
| REST Gateway | grpc-gateway | 2.28.0 |
| Proto Tooling | buf | 1.67.0 |
| Primary Database | PostgreSQL | 16 Alpine |
| Cache / Read Model | Redis | 7 Alpine |
| Message Broker | NATS JetStream | 2 Alpine |
| JWT | golang-jwt | 5.3.1 |
| Password Hashing | bcrypt (x/crypto) | 0.49.0 |
| DB Driver | pgx | 5.9.1 |
| Frontend | React + TypeScript | 19 + 6.0 |
| Build Tool (FE) | Vite | 8.0 |
| CSS | Tailwind CSS | 4.2 |
| Containerization | Docker + Compose | Multi-stage |
| Web Server (FE) | Nginx | 1.27 Alpine |

### Q: Why was Go chosen for this project?

**A:** Go is ideal for microservices because:
- **First-class gRPC support** — Google created both Go and gRPC
- **Fast compilation** — builds in seconds, great for Docker multi-stage
- **Static binary** — single file deployment (`CGO_ENABLED=0`), tiny Alpine images
- **Built-in concurrency** — goroutines and channels for async event handling
- **Strong standard library** — `log/slog`, `net/http`, `context`, `crypto`
- **Low memory footprint** — each service runs in ~30-40MB containers

---

## 2. gRPC Fundamentals

### Q: What is gRPC and why use it over REST?

**A:** gRPC (Google Remote Procedure Call) is an RPC framework that uses HTTP/2 and Protocol Buffers. Advantages over REST:

| Feature | gRPC | REST |
|---------|------|------|
| Protocol | HTTP/2 (multiplexed, binary) | HTTP/1.1 (text-based) |
| Serialization | Protobuf (binary, ~10x smaller) | JSON (text, verbose) |
| Contract | Proto file (strongly typed) | OpenAPI/Swagger (optional) |
| Code generation | Auto-generated client + server stubs | Manual or tool-dependent |
| Streaming | Bidirectional streaming built-in | SSE/WebSocket (bolted on) |
| Performance | ~7-10x faster serialization | Slower JSON parse/serialize |

In this project, gRPC is used for **inter-service communication** (fast, typed, binary) while REST is exposed to **external clients** via gRPC-Gateway.

### Q: What is HTTP/2 and how does gRPC use it?

**A:** HTTP/2 is a binary protocol with:
- **Multiplexing:** Multiple requests/responses on a single TCP connection (no head-of-line blocking)
- **Header compression** (HPACK): Reduces overhead for repeated headers
- **Server push:** Server can proactively send responses
- **Binary framing:** Smaller, faster to parse than text

gRPC uses HTTP/2 for:
- Multiplexed RPC calls over one connection (no connection-per-request overhead)
- Bidirectional streaming (both client and server push frames)
- Efficient header compression for metadata (auth tokens, trace IDs)

### Q: Explain the gRPC service definition model.

**A:** A gRPC service is defined in a `.proto` file:

```protobuf
service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}
```

This generates:
1. **Server interface** (`UserServiceServer`) — you implement this in Go
2. **Client stub** (`UserServiceClient`) — auto-generated, call remote methods like local functions
3. **Registration function** (`RegisterUserServiceServer`) — wires implementation to gRPC server

### Q: What are the 4 types of gRPC communication?

**A:**
1. **Unary RPC** — Client sends one request, server sends one response (like REST). All RPCs in this project are unary.
2. **Server streaming** — Client sends one request, server streams multiple responses (e.g., real-time feeds)
3. **Client streaming** — Client streams multiple requests, server sends one response (e.g., file upload)
4. **Bidirectional streaming** — Both client and server stream simultaneously (e.g., chat)

### Q: How does gRPC handle errors? What status codes exist?

**A:** gRPC uses status codes (not HTTP status codes):

```go
import "google.golang.org/grpc/status"
import "google.golang.org/grpc/codes"

// Return an error
return nil, status.Errorf(codes.NotFound, "user %s not found", id)

// Extract code from error
st, _ := status.FromError(err)
fmt.Println(st.Code())    // codes.NotFound
fmt.Println(st.Message()) // "user xyz not found"
```

Common codes used in this project:
| Code | Meaning | Used When |
|------|---------|-----------|
| `OK` (0) | Success | Default |
| `InvalidArgument` (3) | Bad request | Missing fields, invalid email |
| `NotFound` (5) | Not found | User/product/order doesn't exist |
| `AlreadyExists` (6) | Duplicate | Duplicate email, duplicate SKU |
| `Unauthenticated` (16) | No/invalid auth | Missing or invalid JWT |
| `Internal` (13) | Server error | DB failure, panic recovery |
| `FailedPrecondition` (9) | Business rule | Insufficient stock |

### Q: How is a gRPC server created and started in Go?

**A:** From this project's pattern:

```go
// 1. Create gRPC server with interceptors
grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        interceptors.RecoveryInterceptor(logger),
        interceptors.LoggingInterceptor(logger),
        interceptors.UnaryAuthInterceptor(jwtManager, tokenResolver, publicMethods),
    ),
)

// 2. Register service implementations
userv1.RegisterUserServiceServer(grpcServer, userServer)

// 3. Register health check
grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

// 4. Start listening
lis, _ := net.Listen("tcp", ":50051")
grpcServer.Serve(lis)
```

### Q: What is `grpc.ChainUnaryInterceptor` and why chain interceptors?

**A:** `ChainUnaryInterceptor` wraps multiple interceptors in order, creating a middleware pipeline:

```
Request → Recovery → Logging → Auth → Handler → Response
```

Each interceptor receives `(ctx, req, info, handler)` and can:
- Modify context (add claims, trace IDs)
- Short-circuit (return error before reaching handler)
- Measure timing (log duration after handler returns)
- Catch panics (recovery interceptor)

**Order matters:** Recovery must be first (catches panics from logging/auth/handler), logging second (measures everything including auth), auth third (protects handler).

---

## 3. Protocol Buffers (Protobuf)

### Q: What are Protocol Buffers and why use them?

**A:** Protocol Buffers (protobuf) is Google's language-neutral, platform-neutral, binary serialization format. Advantages:

- **~10x smaller** than JSON (binary encoding, field numbers instead of names)
- **~5-100x faster** parsing (no string matching, known schema)
- **Strongly typed** — compile-time errors for schema violations
- **Backwards compatible** — add fields without breaking old clients
- **Cross-language** — generates code for Go, Java, Python, TypeScript, etc.

### Q: Explain protobuf field numbering and wire types.

**A:** Each field has a unique number used in binary encoding:

```protobuf
message User {
  string id = 1;           // field number 1
  string email = 2;        // field number 2
  string first_name = 5;   // field numbers don't need to be sequential
}
```

**Wire types** determine how bytes are encoded:
| Wire Type | Encoding | Used For |
|-----------|----------|----------|
| 0 | Varint | int32, int64, uint32, bool, enum |
| 1 | 64-bit | fixed64, double |
| 2 | Length-delimited | string, bytes, repeated, nested messages |
| 5 | 32-bit | fixed32, float |

**Binary format:** `(field_number << 3) | wire_type` followed by value bytes.

### Q: What is the difference between `proto2` and `proto3`? Which does this project use?

**A:** This project uses **`proto3`** (`syntax = "proto3";`).

Key differences:
| Feature | proto2 | proto3 |
|---------|--------|--------|
| Default values | Explicit `default` keyword | Zero values (0, "", false) |
| Required fields | `required` keyword | Not supported (all optional) |
| Optional fields | `optional` keyword | All fields are optional by default |
| Presence tracking | Tracks if field was set | Only with `optional` keyword |
| Enum zero value | Any | Must be 0 |
| Unknown fields | Preserved | Preserved (since 3.5) |

### Q: What are the proto files in this project and what do they define?

**A:**

```
proto/
├── common/v1/common.proto    — Shared types: Pagination, Money
├── user/v1/user.proto        — UserService: Register, Login, GetUser, UpdateUser, GetAuthToken
├── product/v1/product.proto  — ProductService: CRUD + ReserveStock, ReleaseStock
├── order/v1/order.proto      — OrderCommandService + OrderQueryService
└── payment/v1/payment.proto  — PaymentService: Get, List, Refund
```

### Q: What does `google.api.http` annotation do in proto files?

**A:** It maps gRPC methods to REST HTTP endpoints for grpc-gateway:

```protobuf
rpc Register(RegisterRequest) returns (RegisterResponse) {
  option (google.api.http) = {
    post: "/api/v1/users/register"
    body: "*"
  };
}

rpc GetProduct(GetProductRequest) returns (GetProductResponse) {
  option (google.api.http) = {
    get: "/api/v1/products/{product_id}"
  };
}
```

**Mapping rules:**
- `post`/`put`/`delete` + `body: "*"` → JSON body maps to request message
- `get` → query parameters map to request fields
- `{product_id}` → URL path parameter maps to field `product_id`

### Q: What is versioning in proto packages (`v1`) and why is it important?

**A:** Package versioning (`user.v1`, `product.v1`) enables:
- **Breaking changes** in `v2` without affecting `v1` clients
- **Parallel operation** — old and new clients work simultaneously
- **Gradual migration** — services can upgrade to `v2` independently
- **API lifecycle** — deprecate `v1` after all clients migrate

Convention: `package <domain>.<version>;` e.g., `package user.v1;`

### Q: What code gets generated from a single `.proto` file?

**A:** For `user.proto`, `buf generate` produces 4 files:

| File | Plugin | Contents |
|------|--------|----------|
| `user.pb.go` | protoc-gen-go | Go structs for all messages (User, RegisterRequest, etc.) |
| `user_grpc.pb.go` | protoc-gen-go-grpc | Server interface + client stub + registration function |
| `user.pb.gw.go` | protoc-gen-grpc-gateway | HTTP reverse-proxy handlers for REST endpoints |
| `user.swagger.json` | protoc-gen-openapiv2 | OpenAPI 2.0 spec for documentation |

---

## 4. gRPC-Gateway (REST ↔ gRPC)

### Q: What is gRPC-Gateway and why is it needed?

**A:** gRPC-Gateway is a reverse proxy that translates RESTful JSON HTTP requests into gRPC calls. It's needed because:

- **Browsers can't speak gRPC** natively (need HTTP/1.1 + JSON)
- **External clients** (mobile apps, third-party APIs) expect REST
- **Internal services** benefit from gRPC's speed and type safety
- **Single source of truth** — proto files define both APIs

### Q: How does gRPC-Gateway work under the hood?

**A:**
```
Browser                 gRPC-Gateway              gRPC Service
   │                        │                         │
   │  POST /api/v1/orders   │                         │
   │  {"user_id":"x",...}   │                         │
   │ ─────────────────────► │                         │
   │                        │  1. Parse JSON body     │
   │                        │  2. Map to PlaceOrder   │
   │                        │     Request message     │
   │                        │  3. Serialize to proto  │
   │                        │  ────────────────────►  │
   │                        │                         │ Handle RPC
   │                        │  ◄────────────────────  │
   │                        │  4. Deserialize proto   │
   │                        │  5. Convert to JSON     │
   │  {"order_id":"y",...}  │                         │
   │ ◄───────────────────── │                         │
```

**In code (gateway/server.go):**
```go
mux := runtime.NewServeMux()

// Register each service's gateway handler
userv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, userAddr, opts)
productv1.RegisterProductServiceHandlerFromEndpoint(ctx, mux, productAddr, opts)
orderv1.RegisterOrderCommandServiceHandlerFromEndpoint(ctx, mux, orderAddr, opts)
orderv1.RegisterOrderQueryServiceHandlerFromEndpoint(ctx, mux, orderAddr, opts)
paymentv1.RegisterPaymentServiceHandlerFromEndpoint(ctx, mux, paymentAddr, opts)
```

### Q: What is `RegisterXxxHandlerFromEndpoint` vs `RegisterXxxHandlerServer`?

**A:**
- **`FromEndpoint`** — Gateway connects to service via gRPC network call (used in this project). Gateway and service are separate processes.
- **`HandlerServer`** — Gateway calls service in-process (same binary). Faster but less flexible.

This project uses `FromEndpoint` because gateway and services run in separate Docker containers.

### Q: How does gRPC-Gateway handle path parameters, query parameters, and request bodies?

**A:**

**Path parameters:**
```protobuf
rpc GetProduct(GetProductRequest) returns (...) {
  option (google.api.http) = { get: "/api/v1/products/{product_id}" };
}
message GetProductRequest { string product_id = 1; }
```
`GET /api/v1/products/abc123` → `GetProductRequest{product_id: "abc123"}`

**Query parameters:**
```protobuf
rpc ListProducts(ListProductsRequest) returns (...) {
  option (google.api.http) = { get: "/api/v1/products" };
}
message ListProductsRequest { int32 page_size = 1; string page_token = 2; }
```
`GET /api/v1/products?page_size=10&page_token=xyz` → `ListProductsRequest{page_size: 10, page_token: "xyz"}`

**Request body:**
```protobuf
rpc PlaceOrder(PlaceOrderRequest) returns (...) {
  option (google.api.http) = { post: "/api/v1/orders" body: "*" };
}
```
`body: "*"` means the entire JSON body maps to the request message.

### Q: How does the gateway handle CORS?

**A:** A custom HTTP middleware wraps the gRPC-Gateway multiplexer:

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(w, r)
    })
}

http.ListenAndServe(":8080", corsMiddleware(mux))
```

This allows the React frontend at `:3000` to call the gateway at `:8080`.

---

## 5. Buf Build Tool

### Q: What is Buf and why use it instead of raw `protoc`?

**A:** Buf is a modern protobuf build tool that replaces `protoc`. Advantages:

| Feature | protoc | Buf |
|---------|--------|-----|
| Plugin management | Manual download + PATH setup | Remote plugins (no local install) |
| Dependency management | Manual or `go get` | `buf.yaml` deps + `buf.lock` |
| Linting | Third-party tools | 50+ built-in STANDARD rules |
| Breaking change detection | None | `buf breaking` detects API-breaking changes |
| Config | CLI flags | Declarative YAML |

### Q: Explain the 3 Buf config files in this project.

**A:**

**`buf.yaml`** — Module definition + dependencies:
```yaml
version: v2
modules:
  - path: proto                              # Where .proto files live
deps:
  - buf.build/googleapis/googleapis          # google.api.http annotations
  - buf.build/grpc-ecosystem/grpc-gateway    # gateway options
lint:
  use: [STANDARD]                            # 50+ lint rules
breaking:
  use: [FILE]                                # Detect breaking changes per-file
```

**`buf.gen.yaml`** — Code generation plugins:
```yaml
plugins:
  - remote: buf.build/protocolbuffers/go      # → *.pb.go (messages)
    out: gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc/go                 # → *_grpc.pb.go (server/client)
    out: gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc-ecosystem/gateway  # → *.pb.gw.go (REST handlers)
    out: gen/go
    opt: [paths=source_relative]
  - remote: buf.build/grpc-ecosystem/openapiv2  # → *.swagger.json
    out: gen/openapiv2
```

**`buf.lock`** — Pinned dependency versions (auto-generated by `buf dep update`).

### Q: What does `paths=source_relative` mean?

**A:** It tells the Go protobuf generator to place output files relative to the source `.proto` file's path, rather than using the `go_package` option's full import path. This means `proto/user/v1/user.proto` generates `gen/go/user/v1/user.pb.go` (mirrors the source structure).

### Q: What does `buf lint` check?

**A:** With `use: [STANDARD]`, it enforces 50+ rules including:
- Package names must be lowercase (`user.v1`, not `User.V1`)
- Service names must be PascalCase and end with `Service`
- RPC names must be PascalCase
- Request/Response messages must be named `<RpcName>Request`/`<RpcName>Response`
- Enum zero value must be `UNSPECIFIED`
- Field names must be snake_case
- No imports outside declared dependencies

---

## 6. Go Language & Patterns

### Q: What is the project's Go module structure?

**A:**
```
go.mod: module github.com/example/go-grpc-http

cmd/               — Service entrypoints (main.go per service)
  gateway/
  user-service/
  product-service/
  order-service/
  payment-service/
  notification-service/
internal/          — Private packages (not importable by external modules)
  user/            — User service business logic
  product/         — Product service business logic
  order/           — Order service (CQRS + Event Sourcing)
  payment/         — Payment service
  notification/    — Notification service
  gateway/         — Gateway server setup
pkg/               — Shared packages (importable by all services)
  database/        — PostgreSQL connection helper
  observability/   — Structured logger factory
  interceptors/    — gRPC middleware (logging, recovery, auth)
gen/               — Auto-generated protobuf code (DO NOT EDIT)
```

### Q: Why `internal/` vs `pkg/`? What's the Go visibility rule?

**A:**
- **`internal/`** — Go enforces that packages under `internal/` can only be imported by code within the same module. Each service's logic is here (e.g., `internal/user/` cannot be imported by `internal/product/`). This enforces encapsulation.
- **`pkg/`** — Shared code importable by all services in the module (database helpers, interceptors, logging). Convention, not enforced by Go.

### Q: Explain the Repository Pattern as used in this project.

**A:** Each service separates business logic from data access via interfaces:

```go
// Interface (internal/user/repository/user_repository.go)
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    GetByID(ctx context.Context, id string) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
}

// PostgreSQL implementation (internal/user/repository/postgres.go)
type postgresUserRepository struct {
    pool *pgxpool.Pool
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
    row := r.pool.QueryRow(ctx, "SELECT ... FROM users WHERE id = $1", id)
    // scan into model.User
}
```

**Benefits:**
- Swap PostgreSQL for in-memory (testing) without changing business logic
- Service layer depends on interface, not concrete implementation
- Each implementation is independently testable

### Q: What is Dependency Injection and how is it done here?

**A:** Dependencies are passed via constructors, not created internally:

```go
// Constructor accepts dependencies
func NewUserService(repo repository.UserRepository, jwtManager *JWTManager) *UserService {
    return &UserService{repo: repo, jwtManager: jwtManager}
}

// In main.go, wire everything together
pool := database.NewPostgresPool(ctx, databaseURL)
repo := repository.NewPostgresUserRepository(pool)
jwtManager := auth.NewJWTManager()
svc := service.NewUserService(repo, jwtManager)
srv := server.NewServer(svc)
```

**Benefits:**
- Testable: inject mock repository
- Explicit: all dependencies visible in constructor signature
- No global state or init() magic

### Q: What is `context.Context` and how is it used throughout the project?

**A:** `context.Context` carries request-scoped values, cancellation signals, and deadlines:

```go
// Passing auth claims through context
ctx = context.WithValue(ctx, "user_id", claims.UserID)

// Using context for database queries (automatic cancellation)
row := pool.QueryRow(ctx, "SELECT ...")

// Passing through gRPC calls (automatic propagation)
resp, err := productClient.GetProduct(ctx, &productv1.GetProductRequest{...})
```

In this project, context flows: HTTP request → gRPC-Gateway → gRPC interceptors → handler → database/NATS/Redis calls. If the client disconnects, all downstream operations are cancelled.

### Q: What is `pgxpool.Pool` and why use connection pooling?

**A:** `pgxpool.Pool` from the `jackc/pgx` driver maintains a pool of PostgreSQL connections:

```go
func NewPostgresPool(ctx context.Context, connStr string) *pgxpool.Pool {
    config, _ := pgxpool.ParseConfig(connStr)
    config.MaxConns = 10
    config.MinConns = 2
    pool, _ := pgxpool.NewWithConfig(ctx, config)
    return pool
}
```

**Why pooling?**
- Creating a TCP + TLS connection per query is expensive (~10-50ms)
- Pool reuses connections (query latency drops to ~1ms)
- Controls max connections (prevents overwhelming PostgreSQL)
- Handles reconnection on transient failures

**Why `pgx` over `database/sql`?**
- Pure Go (no CGO, no libpq dependency) — important for Alpine Docker images
- Native PostgreSQL type support (JSONB, arrays, UUID)
- Connection pooling built-in (no separate library)
- ~2-3x faster than `database/sql` for common operations

---

## 7. PostgreSQL & Database Design

### Q: Why does each service have its own PostgreSQL instance?

**A:** This is the **database-per-service** pattern:

**Advantages:**
- **Loose coupling** — services can evolve schemas independently
- **Independent scaling** — user DB can be read-replicated without affecting order DB
- **Fault isolation** — order DB crash doesn't affect product service
- **Technology freedom** — could swap product DB for MongoDB without affecting others
- **No distributed transactions** — events handle cross-service consistency

**Trade-offs:**
- 5 PostgreSQL instances use more resources
- Cross-service queries impossible (must use API calls or events)
- Data consistency is eventual (not immediate)

### Q: Explain the database schema for each service.

**A:**

**User Service (`userdb`):**
```sql
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,        -- UUID
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,        -- bcrypt
    auth_token TEXT NOT NULL DEFAULT '',-- per-user JWT signing secret
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- Indexes: email (lookups), auth_token WHERE != '' (JWT verification)
```

**Product Service (`productdb`):**
```sql
CREATE TABLE products (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    price_cents BIGINT NOT NULL,       -- $9.99 = 999 cents (avoids float rounding)
    currency VARCHAR(3) DEFAULT 'USD',
    stock_quantity INT NOT NULL DEFAULT 0,
    sku VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- Indexes: sku (unique lookups), created_at DESC (listing)
```

**Order Service (`orderdb`):**
```sql
CREATE TABLE event_store (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id VARCHAR(36) NOT NULL,      -- order_id
    aggregate_type VARCHAR(50) DEFAULT 'order',
    event_type VARCHAR(100) NOT NULL,       -- "order.created", "order.paid"
    payload JSONB NOT NULL,                 -- full event data
    version INT NOT NULL,                   -- per-aggregate version
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(aggregate_id, version)           -- optimistic concurrency
);
```

**Payment Service (`paymentdb`):**
```sql
CREATE TABLE payments (
    id VARCHAR(36) PRIMARY KEY,
    order_id VARCHAR(36) NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(20) DEFAULT 'pending', -- pending/completed/failed/refunded
    method VARCHAR(50) DEFAULT 'card',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- Indexes: order_id, status
```

**Notification Service (`notificationdb`):**
```sql
CREATE TABLE notification_log (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    channel VARCHAR(50) DEFAULT 'email',
    subject VARCHAR(500) NOT NULL,
    body TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Q: Why store prices as `BIGINT` cents instead of `DECIMAL` or `FLOAT`?

**A:**
- **Float:** `0.1 + 0.2 = 0.30000000000000004` — floating point arithmetic causes rounding errors
- **DECIMAL:** Works but slower arithmetic, more storage, and ORM serialization issues
- **Integer cents:** Exact arithmetic, fast, no rounding. `$9.99 = 999 cents`. Convert to dollars only at display time: `price_cents / 100.0`

This is the standard pattern used by Stripe, Square, and most payment systems.

### Q: What is a partial index and why is it used for `auth_token`?

**A:**
```sql
CREATE INDEX idx_users_auth_token ON users(auth_token) WHERE auth_token != '';
```

A **partial index** only indexes rows matching the `WHERE` clause. Benefits:
- Smaller index (skips users without auth_token)
- Faster lookups (less data to scan)
- No wasted space for empty values

### Q: How do migrations work in this project?

**A:** Two approaches are used:

1. **SQL migration files** in `migrations/` — For documentation and manual execution
2. **Inline migrations in code** — Each service runs `CREATE TABLE IF NOT EXISTS` on startup

```go
// cmd/user-service/main.go
migrations := []string{
    `CREATE TABLE IF NOT EXISTS users (...)`,
    `CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
}
for _, m := range migrations {
    pool.Exec(ctx, m)
}
```

**Rationale:** Development convenience — no separate migration CLI tool needed. For production, you'd use a tool like `golang-migrate` or `goose`.

---

## 8. Authentication & JWT

### Q: What is JWT and how is it structured?

**A:** JSON Web Token (JWT) is a compact, URL-safe token format:

```
eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoiYWJjMTIzIiwiZW1haWwiOiJ0ZXN0QGV4YW1wbGUuY29tIiwiZXhwIjoxNzEyODA3NjAwfQ.signature
```

Three parts (base64url encoded, separated by `.`):
1. **Header:** `{"alg": "HS256"}` — signing algorithm
2. **Payload:** `{"user_id": "abc123", "email": "test@example.com", "exp": 1712807600}` — claims
3. **Signature:** `HMAC-SHA256(base64(header) + "." + base64(payload), secret)`

### Q: What is the per-user JWT secret pattern and why is it superior?

**A:** Traditional JWT uses one shared secret for all users. This project uses a **unique secret per user**.

**Traditional (shared secret):**
```
JWT_SECRET = "my-app-secret"
All tokens signed with same secret
Revocation: Need token blacklist (Redis set of revoked JWTs)
If secret leaks: ALL users compromised
```

**Per-user secret (this project):**
```
user.auth_token = base64(32 random bytes)  // unique per user
JWT signed with user's auth_token as HMAC key
Revocation: UPDATE users SET auth_token = new_random() WHERE id = ?
             → ALL old JWTs for that user instantly invalid
If one token leaks: Only that user affected
No blacklist needed — auth_token change = instant revocation
```

**Trade-off:** Requires a DB/gRPC call to resolve `auth_token` for verification (but it's cached per request, not per token validation).

### Q: Walk through the complete login → authenticated request flow.

**A:**

**Step 1: Login**
```
POST /api/v1/users/login  {"email": "john@example.com", "password": "secret123"}

Server:
1. SELECT * FROM users WHERE email = 'john@example.com'
2. bcrypt.CompareHashAndPassword(user.password_hash, "secret123") ✓
3. claims = {user_id: "uuid-123", email: "john@example.com", exp: now+24h}
4. token = jwt.Sign(claims, user.auth_token)  // sign with user's unique secret
5. Return: {access_token: "eyJ...", user_id: "uuid-123", expires_at: "2025-04-11T..."}
```

**Step 2: Authenticated Request**
```
POST /api/v1/orders  Authorization: Bearer eyJ...
                     {"user_id": "uuid-123", "items": [...]}

Auth Interceptor:
1. Extract "Bearer eyJ..." from Authorization header
2. jwt.ParseUnverified(token) → claims.user_id = "uuid-123"
3. tokenResolver.ResolveAuthToken("uuid-123")
   → gRPC call to UserService.GetAuthToken("uuid-123")
   → returns user's auth_token from DB
4. jwt.Validate(token, auth_token) → verify HMAC signature
5. If valid: inject claims into context, proceed to handler
   If invalid: return codes.Unauthenticated
```

### Q: How does token revocation work without a blacklist?

**A:**
```
1. User changes password or admin revokes access
2. UPDATE users SET auth_token = encode(gen_random_bytes(32), 'base64') WHERE id = ?
3. New auth_token generated (old one overwritten)
4. Next request with old JWT:
   - Interceptor resolves NEW auth_token from DB
   - Tries to verify old JWT with NEW auth_token
   - HMAC signature doesn't match → verification fails
   - Return: codes.Unauthenticated
5. User must re-login to get new JWT signed with new auth_token
```

**No blacklist, no Redis cache of revoked tokens, no expiry scanning.**

### Q: What is bcrypt and why is it used for password hashing?

**A:** bcrypt is a deliberately slow, adaptive hashing algorithm:

```go
// Hash (on registration)
hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// bcrypt.DefaultCost = 10 → ~100ms per hash

// Verify (on login)
err := bcrypt.CompareHashAndPassword(hash, []byte(password))
```

**Why bcrypt over SHA-256/MD5?**
- **Intentionally slow** — 100ms vs 1μs. Brute-force attacker tries ~10/sec vs ~1B/sec
- **Built-in salt** — each hash has unique random salt (same password → different hash)
- **Adaptive cost** — increase cost factor as hardware gets faster
- **Resistant to rainbow tables** — salt makes precomputed tables useless

### Q: What are the public (unauthenticated) vs protected (authenticated) methods?

**A:**

**Public (no JWT needed):**
- Health checks: `/grpc.health.v1.Health/Check`
- Auth: `Register`, `Login`
- Read-only: `GetProduct`, `ListProducts`, `GetOrder`, `ListOrdersByUser`, `GetPayment`, `ListPaymentsByOrder`
- Internal: `ReserveStock`, `ReleaseStock`, `GetAuthToken` (gRPC-only, no REST)

**Protected (JWT required):**
- User: `GetUser`, `UpdateUser`
- Product: `CreateProduct`, `UpdateProduct`, `UpdateInventory`
- Order: `PlaceOrder`, `CancelOrder`
- Payment: `RefundPayment`

---

## 9. CQRS (Command Query Responsibility Segregation)

### Q: What is CQRS and why does the order service use it?

**A:** CQRS separates read and write operations into different models:

```
                    ┌─────────────────────┐
                    │   Order Service      │
                    │                      │
    Commands ──────►│  Command Handler     │──── PostgreSQL (event_store)
    (PlaceOrder,    │  (write model)       │     append-only events
     CancelOrder)   │                      │
                    │                      │
    Queries ───────►│  Query Handler       │──── Redis (denormalized views)
    (GetOrder,      │  (read model)        │     fast key-value lookups
     ListOrders)    │                      │
                    └─────────────────────┘
```

**Why CQRS for orders?**
- **Writes are complex:** Place order involves stock reservation, event creation, publishing
- **Reads are simple:** "Give me order X" is a key-value lookup
- **Different scaling needs:** Reads are ~100x more frequent than writes
- **Different data models:** Write model is event-based (append-only), read model is denormalized (fast)

### Q: How is the read model built and kept in sync?

**A:** A **projector** subscribes to events and builds the Redis read model:

```go
// Projector subscribes to NATS
nats.Subscribe("orders.*", func(msg *nats.Msg) {
    switch eventType {
    case "orders.created":
        // Create new order in Redis (status: pending)
        redis.Set("order:"+orderID, orderJSON)
        // Add to user's order list
        redis.RPush("user_orders:"+userID, orderID)

    case "orders.paid":
        // Update existing order in Redis
        order := redis.Get("order:" + orderID)
        order.Status = "paid"
        order.PaymentID = paymentID
        redis.Set("order:"+orderID, orderJSON)

    case "orders.cancelled":
        order := redis.Get("order:" + orderID)
        order.Status = "cancelled"
        redis.Set("order:"+orderID, orderJSON)
    }
    msg.Ack()
})
```

**Sync guarantees:**
- **Eventually consistent** — read model may lag by milliseconds
- **Durable consumer** — if projector crashes, events replay on restart
- **Idempotent** — re-processing the same event produces the same result (upsert)

### Q: What happens if the read model (Redis) becomes corrupted?

**A:** The read model can be completely rebuilt from the event store:

1. Flush Redis
2. Read all events from PostgreSQL `event_store` ordered by `created_at`
3. Replay each event through the projector
4. Redis is now consistent with event store

This is a major advantage of event sourcing — the event store is the source of truth, the read model is a disposable projection.

### Q: What are the trade-offs of CQRS?

**A:**

| Advantage | Trade-off |
|-----------|-----------|
| Independent scaling of reads/writes | Increased complexity (2 models, projector) |
| Optimized read model (fast queries) | Eventual consistency (reads may lag) |
| Different data stores per side | Need to maintain projector code |
| Simpler command handlers (no read concerns) | Debugging requires tracing events |
| Write model can be event-sourced | More infrastructure (Redis + PostgreSQL + NATS) |

---

## 10. Event Sourcing

### Q: What is Event Sourcing and how does it differ from CRUD?

**A:**

**CRUD (traditional):**
```
UPDATE orders SET status = 'paid', payment_id = 'pay-123' WHERE id = 'ord-456';
-- Previous state lost forever
```

**Event Sourcing:**
```
INSERT INTO event_store (aggregate_id, event_type, payload, version)
VALUES ('ord-456', 'order.paid', '{"payment_id":"pay-123","paid_at":"..."}', 2);
-- All previous states preserved
```

**Key difference:** In event sourcing, you never update or delete. You only append events. Current state is derived by replaying all events for an aggregate.

### Q: How does the Order aggregate work?

**A:**

```go
type OrderAggregate struct {
    ID        string
    UserID    string
    Items     []OrderItem
    Status    string
    Events    []Event  // uncommitted events
    Version   int
}

// Command: Place Order
func (a *OrderAggregate) PlaceOrder(userID string, items []OrderItem, total int64, currency string) {
    a.apply(OrderCreatedEvent{
        OrderID: a.ID, UserID: userID, Items: items,
        TotalCents: total, Currency: currency,
    })
}

// Command: Mark Paid
func (a *OrderAggregate) MarkPaid(paymentID string) {
    if a.Status != "pending" { return error }
    a.apply(OrderPaidEvent{OrderID: a.ID, PaymentID: paymentID})
}

// Apply event to aggregate state
func (a *OrderAggregate) apply(event Event) {
    switch e := event.(type) {
    case OrderCreatedEvent:
        a.Status = "pending"
        a.UserID = e.UserID
        a.Items = e.Items
    case OrderPaidEvent:
        a.Status = "paid"
    case OrderCancelledEvent:
        a.Status = "cancelled"
    }
    a.Version++
    a.Events = append(a.Events, event)
}
```

**Rebuilding state from events:**
```go
func LoadAggregate(events []Event) *OrderAggregate {
    agg := &OrderAggregate{}
    for _, e := range events {
        agg.apply(e)  // replay each event in order
    }
    return agg
}
```

### Q: What is optimistic concurrency control in the event store?

**A:** The `UNIQUE(aggregate_id, version)` constraint prevents concurrent conflicting writes:

```sql
-- Two concurrent writers try to append version 3 for the same order:
-- Writer A: INSERT INTO event_store (aggregate_id, version, ...) VALUES ('ord-1', 3, ...)  ✓
-- Writer B: INSERT INTO event_store (aggregate_id, version, ...) VALUES ('ord-1', 3, ...)  ✗ UNIQUE violation!
```

**Flow:**
1. Load aggregate: read events, replay → current version = 2
2. Produce new event: version = 3
3. INSERT event with version = 3
4. If another writer already inserted version 3 → unique constraint violation → retry

**No locks needed** — the database constraint handles concurrency.

### Q: What are the advantages of Event Sourcing?

**A:**
1. **Complete audit trail** — every state change is an immutable event
2. **Time travel** — reconstruct state at any point by replaying events up to that timestamp
3. **Event replay** — rebuild read models, fix projector bugs, backfill new projections
4. **Debugging** — "how did this order end up cancelled?" → look at event sequence
5. **Decoupling** — events are consumed by any interested service (payment, notification)
6. **No data loss** — never delete or overwrite; only append

---

## 11. NATS JetStream & Event-Driven Architecture

### Q: What is NATS and what is JetStream?

**A:**
- **NATS** — Lightweight, high-performance message broker (pub/sub, request/reply)
- **JetStream** — NATS's persistence layer (adds durable subscriptions, at-least-once delivery, message history)

**NATS alone:** Messages are fire-and-forget (if no subscriber is listening, message is lost)
**NATS + JetStream:** Messages persisted to disk, consumers can replay from any point

### Q: What subjects (topics) exist in this project?

**A:**

| Subject | Publisher | Subscribers | Event |
|---------|-----------|-------------|-------|
| `orders.created` | Order Command Handler | Projector, Payment Subscriber, Notification | New order placed |
| `orders.paid` | Order Payment Subscriber | Projector, Notification | Payment confirmed |
| `orders.cancelled` | Order Command Handler | Projector, Notification | Order cancelled |
| `payments.completed` | Payment Service | Order Payment Subscriber, Notification | Payment processed |
| `payments.failed` | Payment Service | Notification | Payment failed |

### Q: What is a durable consumer and why is it critical?

**A:** A durable consumer maintains its position in the message stream across restarts:

```go
// Durable subscriber
nats.Subscribe("orders.*", handler,
    nats.Durable("order-projector"),  // Name persisted in JetStream
    nats.DeliverAll(),                // Start from first message
)
```

**Without durable:** If projector crashes at message #50 and restarts, it starts from the latest message (misses #50-#99).

**With durable:** JetStream remembers that `order-projector` last acknowledged message #49. On restart, it delivers #50 onwards.

**Consumer groups in this project:**
- `order-projector` — builds Redis read model
- `payment-processor` — creates payments from order events
- `notification-orders` — sends order notifications
- `notification-payments` — sends payment notifications
- `order-payment-subscriber` — marks orders as paid

### Q: What delivery guarantees does NATS JetStream provide?

**A:**

| Guarantee | NATS Core | NATS JetStream |
|-----------|-----------|---------------|
| At-most-once | ✓ (default) | - |
| At-least-once | ✗ | ✓ (with Ack) |
| Exactly-once | ✗ | ✓ (with dedup) |

**This project uses at-least-once:**
- Message delivered to subscriber
- Subscriber processes and calls `msg.Ack()`
- If subscriber crashes before Ack, message is redelivered
- Handlers must be **idempotent** (processing twice = same result)

### Q: How is idempotency achieved in event handlers?

**A:** Several strategies:

1. **Upsert (Redis projector):** `SET order:{id}` overwrites previous value — safe to replay
2. **Status checks:** Payment subscriber checks if payment already exists for order before creating
3. **Unique constraints:** Event store's `UNIQUE(aggregate_id, version)` prevents duplicate events
4. **Notification log:** INSERT with UUID primary key — duplicate UUIDs rejected

### Q: What is the difference between pub/sub and request/reply in NATS?

**A:**
- **Pub/Sub:** Publisher broadcasts to subject, all subscribers receive (used in this project for events)
- **Request/Reply:** Client sends request, one server responds (used for RPC-like patterns)

This project uses **pub/sub exclusively** — events are broadcast to all interested consumers.

---

## 12. Redis

### Q: Why is Redis used and what role does it play?

**A:** Redis serves as the **CQRS read model** for the order service:

- **Write model:** PostgreSQL event_store (source of truth)
- **Read model:** Redis (denormalized, fast-access order views)

**Why Redis over PostgreSQL for reads?**
- **Sub-millisecond reads** — in-memory key-value lookup
- **Simple data model** — `order:{id}` → JSON blob (no JOINs)
- **Disposable** — can be rebuilt from event store at any time
- **Scales horizontally** — Redis Cluster for high throughput

### Q: What data structure does the Redis read model use?

**A:**
```
KEY: order:{order_id}
VALUE: JSON {
    "order_id": "uuid",
    "user_id": "uuid",
    "items": [{"product_id":"...","product_name":"...","quantity":2,"price_cents":999}],
    "total_cents": 1998,
    "currency": "USD",
    "status": "pending" | "paid" | "cancelled",
    "payment_id": "uuid" (optional),
    "created_at": "2025-...",
    "updated_at": "2025-..."
}

KEY: user_orders:{user_id}
VALUE: LIST ["order-id-1", "order-id-2", ...]
```

### Q: What happens if Redis goes down?

**A:**
1. **Read queries fail** — `GetOrder` and `ListOrdersByUser` return errors
2. **Write commands still work** — PlaceOrder writes to PostgreSQL event store and NATS (no Redis dependency on write path)
3. **Recovery:** When Redis comes back, projector replays events from NATS (durable consumer) and rebuilds the read model
4. **Full rebuild:** Can also replay all events from PostgreSQL event_store if NATS history is exhausted

---

## 13. gRPC Interceptors & Middleware

### Q: What are gRPC interceptors and how do they compare to HTTP middleware?

**A:** Interceptors are the gRPC equivalent of HTTP middleware — they wrap RPC handlers:

| HTTP Middleware | gRPC Interceptor |
|-----------------|-----------------|
| `func(next http.Handler) http.Handler` | `grpc.UnaryServerInterceptor` |
| Wraps HTTP handlers | Wraps RPC handlers |
| Access to `http.Request` / `http.ResponseWriter` | Access to `context`, `interface{}` (request), `grpc.UnaryServerInfo` |
| Chains via `middleware(next)` | Chains via `grpc.ChainUnaryInterceptor(a, b, c)` |

### Q: Explain each interceptor in this project.

**A:**

**1. Recovery Interceptor (first in chain):**
```go
func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        defer func() {
            if r := recover(); r != nil {
                logger.Error("panic recovered",
                    slog.Any("panic", r),
                    slog.String("stack", string(debug.Stack())),
                )
                err = status.Error(codes.Internal, "internal error")
            }
        }()
        return handler(ctx, req)
    }
}
```
- **Purpose:** Catches panics in any downstream interceptor or handler
- **Why first:** Must wrap everything to prevent gRPC server crash
- **Returns:** `codes.Internal` to client (hides internal details)

**2. Logging Interceptor (second):**
```go
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        start := time.Now()
        resp, err := handler(ctx, req)
        st, _ := status.FromError(err)
        logger.Info("gRPC call",
            slog.String("method", info.FullMethod),
            slog.String("code", st.Code().String()),
            slog.Duration("duration", time.Since(start)),
        )
        return resp, err
    }
}
```
- **Purpose:** Logs every RPC: method name, status code, duration
- **Why second:** Measures time including auth interceptor

**3. Auth Interceptor (third):**
```go
func UnaryAuthInterceptor(jwtManager, tokenResolver, publicMethods) grpc.UnaryServerInterceptor {
    return func(ctx, req, info, handler) {
        if isPublicMethod(info.FullMethod) {
            return handler(ctx, req)  // skip auth
        }
        token := extractBearerToken(ctx)
        claims := jwtManager.ParseUnverified(token)
        authToken := tokenResolver.ResolveAuthToken(claims.UserID)
        if err := jwtManager.Validate(token, authToken); err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        ctx = context.WithValue(ctx, "claims", claims)
        return handler(ctx, req)
    }
}
```
- **Purpose:** Validates JWT, injects claims into context
- **Why third:** Only runs for protected methods

### Q: Why does interceptor order matter?

**A:**
```
Request → [Recovery] → [Logging] → [Auth] → [Handler]

If Auth panics:    Recovery catches it ✓, Logging records it ✓
If Handler panics: Recovery catches it ✓, Logging records it ✓, Auth already passed ✓
If Recovery is last: panic in Logging/Auth crashes the server!
```

Wrong order `[Logging, Auth, Recovery]` — a panic in Auth would crash the server before Recovery sees it.

---

## 14. Observability & Logging

### Q: What is `log/slog` and why was it chosen over third-party loggers?

**A:** `log/slog` is Go's standard library structured logger (added in Go 1.21):

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

logger.Info("user registered",
    slog.String("user_id", "abc-123"),
    slog.String("email", "john@example.com"),
    slog.Duration("duration", 45*time.Millisecond),
)
// Output: {"time":"2025-04-10T...","level":"INFO","msg":"user registered","user_id":"abc-123","email":"john@example.com","duration":"45ms"}
```

**Why `slog` over `zap`/`logrus`?**
- Standard library — no external dependency
- JSON output — structured, machine-parseable
- Type-safe — `slog.String`, `slog.Int`, `slog.Duration` (not `map[string]interface{}`)
- Performance — comparable to `zap` (the fastest Go logger)

### Q: What is the gRPC Health Check Protocol?

**A:** It's a standardized proto service (`grpc.health.v1.Health`) that all gRPC services implement:

```protobuf
service Health {
    rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
    rpc Watch(HealthCheckRequest) returns (stream HealthCheckResponse);
}
```

In this project:
```go
healthServer := health.NewServer(
    func(ctx context.Context) error { return pool.Ping(ctx) },      // PostgreSQL
    func(ctx context.Context) error { return rdb.Ping(ctx).Err() }, // Redis (order service)
)
grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
```

Used by:
- Docker Compose healthchecks
- Load balancers
- Gateway's `/healthz` endpoint

---

## 15. Docker & Containerization

### Q: Explain the multi-stage Docker build pattern used here.

**A:**

```dockerfile
# Stage 1: Build (large, has compiler + tools)
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download          # Cache dependencies
COPY . .
RUN CGO_ENABLED=0 go build -o /user-service ./cmd/user-service

# Stage 2: Runtime (tiny, only binary)
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /user-service /user-service
EXPOSE 50051
ENTRYPOINT ["/user-service"]
```

**Why multi-stage?**
- Builder image: ~1.3GB (Go compiler, tools, source code)
- Runtime image: ~32-40MB (only binary + ca-certificates)
- **97% smaller** deployment image
- Faster pulls, less disk, smaller attack surface

### Q: What does `CGO_ENABLED=0` do and why is it important?

**A:** It disables CGO (Go's C interop layer), producing a **fully static binary**:

- Without: Binary links to `libc` (needs glibc or musl at runtime)
- With `CGO_ENABLED=0`: Binary has zero runtime dependencies
- Critical for `alpine` images which use `musl` (not `glibc`)
- The `pgx` driver is pure Go — no C dependencies

### Q: How does Docker Compose orchestrate 14 containers?

**A:**

**Infrastructure tier (starts first, with healthchecks):**
- 5 PostgreSQL instances (one per service)
- 1 Redis instance
- 1 NATS instance (with JetStream enabled)

**Application tier (waits for infrastructure):**
- 5 backend services + 1 gateway
- Each depends on its PostgreSQL instance being healthy

**Frontend tier (waits for gateway):**
- Nginx serving React SPA

**Startup order controlled by:**
```yaml
depends_on:
  postgres-user:
    condition: service_healthy  # Wait for pg_isready to pass
```

### Q: How does the frontend Nginx container proxy API calls?

**A:**
```nginx
# Serve React SPA
location / {
    try_files $uri /index.html;   # SPA fallback (client-side routing)
}

# Proxy API calls to gateway
location /api {
    proxy_pass http://gateway:8080;  # Docker DNS resolves "gateway" container
}

# Cache static assets aggressively
location ~ ^/assets {
    expires 365d;
    add_header Cache-Control "public, immutable";
}

# Enable gzip compression
gzip on;
gzip_types text/plain text/css application/json application/javascript;
```

### Q: What is the `try_files $uri /index.html` directive?

**A:** It's the SPA fallback pattern:
1. If requested file exists (e.g., `/assets/index-abc123.js`) → serve it
2. If not → serve `/index.html` → React Router handles the route client-side

Without this, refreshing `/orders/abc-123` would return 404 (Nginx has no such file).

---

## 16. Frontend (React + TypeScript)

### Q: What is the frontend architecture?

**A:** Feature-based organization:

```
src/
├── types/api.ts          — All API TypeScript interfaces
├── lib/
│   ├── api-client.ts     — HTTP client (auto-attaches JWT)
│   ├── format.ts         — Currency/date/status formatters
│   └── product-images.ts — Product image mappings
├── hooks/useAsync.ts     — Generic async data fetching hook
├── context/AuthContext.tsx — Global auth state (token, userId, login/logout)
├── layouts/AppLayout.tsx  — Navigation + Outlet wrapper
├── components/ui/         — Reusable UI components (Button, Input, Card, Badge, Alert)
└── features/
    ├── auth/              — LoginPage, RegisterPage
    ├── dashboard/         — DashboardPage (stats, recent orders)
    ├── products/          — ProductsPage, ProductDetailPage
    ├── orders/            — OrdersPage, OrderDetailPage
    └── payments/          — Payment API helpers
```

### Q: How does the API client auto-attach JWT tokens?

**A:**
```typescript
class ApiClient {
    private getHeaders(): Record<string, string> {
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        const token = localStorage.getItem('token');
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }
        return headers;
    }

    async get<T>(url: string): Promise<T> {
        const res = await fetch(url, { headers: this.getHeaders() });
        if (!res.ok) throw new ApiError(res.status, await res.text());
        return res.json();
    }

    async post<T>(url: string, body: unknown): Promise<T> {
        const res = await fetch(url, {
            method: 'POST',
            headers: this.getHeaders(),
            body: JSON.stringify(body),
        });
        if (!res.ok) throw new ApiError(res.status, await res.text());
        return res.json();
    }
}
```

Every API call automatically includes the JWT from localStorage.

### Q: What is the `useAsync` hook and why is it useful?

**A:**
```typescript
function useAsync<T>(asyncFn: () => Promise<T>) {
    const [data, setData] = useState<T | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const execute = async () => {
        setLoading(true);
        setError(null);
        try {
            const result = await asyncFn();
            setData(result);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    return { data, loading, error, execute };
}
```

**Usage:**
```typescript
const { data: products, loading, error, execute } = useAsync(
    () => api.get<Product[]>('/api/v1/products')
);

useEffect(() => { execute(); }, []);

if (loading) return <Spinner />;
if (error) return <Alert variant="error">{error}</Alert>;
return <ProductGrid products={products} />;
```

Eliminates boilerplate loading/error state management in every component.

### Q: How does the AuthContext work?

**A:**
```typescript
const AuthContext = createContext<AuthState>(null);

function AuthProvider({ children }) {
    const [token, setToken] = useState(localStorage.getItem('token'));
    const [userId, setUserId] = useState(localStorage.getItem('userId'));
    const [email, setEmail] = useState(localStorage.getItem('email'));

    const login = (accessToken: string, userId: string, email: string) => {
        localStorage.setItem('token', accessToken);
        localStorage.setItem('userId', userId);
        localStorage.setItem('email', email);
        setToken(accessToken);
        setUserId(userId);
        setEmail(email);
    };

    const logout = () => {
        localStorage.clear();
        setToken(null);
        setUserId(null);
        setEmail(null);
    };

    const isAuthenticated = !!token;

    return (
        <AuthContext.Provider value={{ token, userId, email, isAuthenticated, login, logout }}>
            {children}
        </AuthContext.Provider>
    );
}

// Usage in any component:
const { isAuthenticated, userId, logout } = useAuth();
```

---

## 17. Microservice Communication Patterns

### Q: What are the two communication styles in this project?

**A:**

**1. Synchronous (gRPC — request/response):**
- Gateway → Services (REST translated to gRPC)
- Order Service → Product Service (GetProduct, ReserveStock)
- Auth Interceptor → User Service (GetAuthToken)
- **Used for:** Operations requiring immediate response

**2. Asynchronous (NATS JetStream — pub/sub):**
- Order Service → NATS → Payment Service (order.created)
- Payment Service → NATS → Order Service (payments.completed)
- All services → NATS → Notification Service
- **Used for:** Operations that can be processed later (eventual consistency)

### Q: What is the Saga pattern and is it used here?

**A:** The order placement is an **implicit choreography-based saga**:

```
PlaceOrder (Saga):
Step 1: Reserve stock (sync gRPC) → compensate: release stock
Step 2: Create event (PostgreSQL) → no compensate needed (append-only)
Step 3: Publish event (NATS) → subscribers process asynchronously

CancelOrder (Compensating Transaction):
Step 1: Produce OrderCancelledEvent
Step 2: Release stock (sync gRPC) → undo Step 1's reservation
Step 3: Publish event → payment service refunds, notification sends email
```

It's not a formal saga framework, but follows the pattern: a sequence of local transactions with compensating actions on failure.

### Q: How does stock reservation handle partial failures?

**A:**
```go
// In PlaceOrder command handler:
var reserved []ReservedItem

for _, item := range order.Items {
    err := productClient.ReserveStock(ctx, item.ProductID, item.Quantity)
    if err != nil {
        // Rollback all previously reserved items
        for _, r := range reserved {
            productClient.ReleaseStock(ctx, r.ProductID, r.Quantity)
        }
        return nil, status.Error(codes.FailedPrecondition, "insufficient stock")
    }
    reserved = append(reserved, item)
}
```

**Pattern:** Reserve forward, rollback backward on any failure.

---

## 18. Error Handling & Resilience

### Q: How does error handling flow from database to client?

**A:**
```
PostgreSQL error (e.g., unique constraint violation)
    ↓
Repository: wraps in domain error → "email already exists"
    ↓
Service: translates to gRPC status → status.Error(codes.AlreadyExists, "email already exists")
    ↓
Interceptors: log the error
    ↓
gRPC-Gateway: translates to HTTP → 409 Conflict {"message": "email already exists"}
    ↓
Frontend: displays error in Alert component
```

**gRPC → HTTP status code mapping (automatic by gRPC-Gateway):**
| gRPC Code | HTTP Status |
|-----------|-------------|
| OK | 200 |
| InvalidArgument | 400 |
| Unauthenticated | 401 |
| NotFound | 404 |
| AlreadyExists | 409 |
| FailedPrecondition | 412 |
| Internal | 500 |

### Q: What happens if a downstream service is unavailable?

**A:**

| Scenario | Behavior |
|----------|----------|
| Product Service down | PlaceOrder fails immediately (sync dependency) |
| Payment Service down | Order placed successfully (status: pending), payment processes when service recovers (NATS replay) |
| Notification Service down | No emails sent, but all business logic unaffected (fire-and-forget) |
| Redis down | Read queries fail, write commands still work |
| NATS down | Events not published, but event_store has them (can replay later) |
| User Service down | Login/register fail, JWT verification fails for protected routes |

### Q: How does the Recovery interceptor prevent server crashes?

**A:**
```go
defer func() {
    if r := recover(); r != nil {
        logger.Error("panic recovered",
            slog.Any("panic", r),
            slog.String("stack", string(debug.Stack())),
        )
        err = status.Error(codes.Internal, "internal error")
    }
}()
```

Without this:
- A nil pointer dereference in any handler would crash the entire gRPC server
- All other connections would be terminated
- Service would need to restart

With recovery:
- Panic is caught, logged with stack trace
- Client gets `codes.Internal` error
- Server continues serving other requests

---

## 19. Concurrency & Performance

### Q: How does Go handle concurrent gRPC requests?

**A:** gRPC server in Go spawns a **new goroutine per RPC call** automatically:

```go
// grpcServer.Serve(lis) internally does:
for {
    conn, _ := lis.Accept()
    go handleConnection(conn)  // Each connection in its own goroutine
    // Each RPC within a connection also gets its own goroutine
}
```

- Goroutines are lightweight (~4KB stack, vs ~1MB for OS threads)
- Go runtime multiplexes goroutines onto OS threads (M:N scheduling)
- 100K concurrent RPCs use ~400MB of goroutine stacks

### Q: How is `pgxpool` connection pooling tuned?

**A:**
```go
config.MaxConns = 10  // Max open connections to PostgreSQL
config.MinConns = 2   // Keep at least 2 idle connections ready
```

**Why these values?**
- PostgreSQL's default `max_connections` = 100
- 5 services × 10 max = 50 connections (leaves headroom)
- 2 min connections = warm pool (no connection creation latency for first queries)

**What happens at max?**
- 11th concurrent query blocks until a connection is released
- `pgxpool` returns `context.DeadlineExceeded` if context times out while waiting

### Q: How does the atomic stock reservation prevent overselling?

**A:**
```sql
UPDATE products 
SET stock_quantity = stock_quantity - $1
WHERE id = $2 AND stock_quantity >= $1
RETURNING stock_quantity;
```

**Why this is safe:**
- `WHERE stock_quantity >= $1` — only succeeds if sufficient stock
- `stock_quantity - $1` — atomic decrement (no read-modify-write race)
- PostgreSQL row-level lock during UPDATE — concurrent updates serialize
- If affected rows = 0 → insufficient stock → return error

**Without `WHERE` check:**
```sql
-- BAD: race condition
SELECT stock_quantity FROM products WHERE id = ?;   -- returns 5
-- Another request also reads 5
UPDATE products SET stock_quantity = 5 - 3 WHERE id = ?;  -- sets to 2
UPDATE products SET stock_quantity = 5 - 3 WHERE id = ?;  -- sets to 2 (should be -1!)
```

---

## 20. Testing Strategies

### Q: How is the codebase structured for testability?

**A:**

1. **Interfaces everywhere** — Repository, Service, TokenResolver all have interfaces
2. **Dependency injection** — constructors accept interfaces, not concrete types
3. **Separation of concerns** — handler ≠ service ≠ repository
4. **No global state** — all state passed via constructors or context

```go
// Easy to test with mock
type mockUserRepo struct {}
func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    return &User{ID: id, Email: "test@test.com"}, nil
}

svc := service.NewUserService(&mockUserRepo{}, jwtManager)
// Now test service logic without real database
```

### Q: What testing strategies would you apply to this project?

**A:**

| Layer | Strategy | Tools |
|-------|----------|-------|
| Repository | Integration tests with real PostgreSQL | `testcontainers-go` + `pgxpool` |
| Service | Unit tests with mock repositories | Go interfaces + `gomock` or hand-written mocks |
| gRPC Handler | Integration tests with `bufconn` | In-memory gRPC connections |
| Event handlers | Integration tests with embedded NATS | `nats-server` test helper |
| API (E2E) | HTTP tests against gateway | `net/http/httptest` + real services |
| Frontend | Component tests + API mocking | Vitest + MSW (Mock Service Worker) |

### Q: What is `bufconn` and how is it used for gRPC testing?

**A:** `bufconn` creates an in-memory gRPC connection (no TCP, no port):

```go
lis := bufconn.Listen(1024 * 1024)
grpcServer := grpc.NewServer()
userv1.RegisterUserServiceServer(grpcServer, myServer)
go grpcServer.Serve(lis)

// Create client that connects in-memory
conn, _ := grpc.DialContext(ctx, "bufnet",
    grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return lis.Dial()
    }),
    grpc.WithInsecure(),
)
client := userv1.NewUserServiceClient(conn)

// Test like it's a real network call
resp, err := client.Register(ctx, &userv1.RegisterRequest{...})
```

**Benefits:** Fast (no network), no port conflicts, full gRPC behavior (interceptors, metadata).

---

## 21. Scenario-Based / System Design Questions

### Q: How would you add a new "Inventory Service" that tracks stock independently?

**A:**
1. Create `proto/inventory/v1/inventory.proto` with `InventoryService`
2. Generate code: `buf generate`
3. Create `internal/inventory/` with repository, service, server
4. Add `migrations/inventory/001_create_inventory.sql`
5. Add PostgreSQL instance in `docker-compose.yml`
6. Register with gateway in `internal/gateway/server.go`
7. Product service calls `InventoryService.ReserveStock` instead of self-managing stock
8. Publish inventory events to NATS (`inventory.reserved`, `inventory.released`)

### Q: How would you add distributed tracing (OpenTelemetry)?

**A:**
1. Add `go.opentelemetry.io/otel` dependency
2. Create trace provider in `pkg/observability/tracing.go`
3. Add `otelgrpc.UnaryServerInterceptor()` to interceptor chain
4. Add `otelgrpc.UnaryClientInterceptor()` to all gRPC client connections
5. Trace ID propagated automatically via gRPC metadata
6. Export traces to Jaeger/Zipkin container added to `docker-compose.yml`

### Q: How would you handle service discovery in production (no hardcoded addresses)?

**A:** Options:
1. **DNS-based (Kubernetes):** Service names resolve via kube-dns (`user-service:50051`)
2. **Consul/etcd:** Register on startup, health check, resolve via API
3. **Envoy sidecar:** Service mesh handles routing, load balancing, retries
4. **gRPC built-in:** Use `grpc.WithResolvers()` + custom name resolver

### Q: If the event store grows to billions of rows, how would you optimize?

**A:**
1. **Partitioning:** Partition `event_store` by `aggregate_id` hash (distribute across disks)
2. **Snapshots:** Periodically save aggregate state → only replay events after snapshot
3. **Archiving:** Move old events to cold storage (S3), keep recent in PostgreSQL
4. **Read model:** Already using Redis (fast reads), add more projections as needed
5. **Sharding:** Split aggregates across multiple PostgreSQL instances

### Q: How would you make this system handle 10,000 orders/second?

**A:**
1. **Horizontal scaling:** Run N instances of each service behind load balancer
2. **Connection pooling:** Increase `pgxpool.MaxConns`, use PgBouncer
3. **NATS clustering:** Run 3+ NATS servers for high availability
4. **Redis Cluster:** Shard read model across multiple Redis nodes
5. **Event store optimization:** Batch event inserts, use `COPY` for bulk writes
6. **Async everywhere:** Make stock reservation async (event-driven instead of sync gRPC)
7. **Caching:** Cache product data (rarely changes), cache user auth_tokens

### Q: What would break if you removed NATS and used synchronous calls everywhere?

**A:**
1. **Tight coupling:** Order service must know about payment, notification services
2. **Cascading failures:** If notification service is slow, PlaceOrder becomes slow
3. **No fault tolerance:** If payment service is down, PlaceOrder fails immediately
4. **No replay:** Lost events can't be recovered (no message history)
5. **Scaling bottleneck:** Every order blocks on payment + notification processing
6. **Circular dependencies:** Order → Payment → Order (for marking paid) creates deadlock risk

### Q: How would you add rate limiting to the API gateway?

**A:**
Add HTTP middleware before the gRPC-Gateway mux:

```go
func rateLimitMiddleware(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(100), 200) // 100 req/s, burst 200
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

http.ListenAndServe(":8080", rateLimitMiddleware(corsMiddleware(mux)))
```

For per-user rate limiting, use `sync.Map` keyed by JWT user_id.

### Q: How would you deploy this to Kubernetes?

**A:**
1. **Deployments:** One per service (replicas: 2+)
2. **Services:** ClusterIP for internal gRPC, LoadBalancer for gateway
3. **ConfigMaps:** Environment variables (DATABASE_URL, service addresses)
4. **Secrets:** Database passwords, JWT secrets
5. **PersistentVolumeClaims:** PostgreSQL data
6. **Ingress:** Nginx Ingress for external HTTPS → gateway
7. **Health probes:** `livenessProbe` and `readinessProbe` using gRPC health check
8. **HPA:** Horizontal Pod Autoscaler based on CPU/memory
9. **StatefulSet:** For PostgreSQL (stable network identity, ordered scaling)
10. **Helm chart:** Template all the above for different environments

---

## Bonus: Quick Reference Commands

```bash
# Start everything
docker compose up -d --build

# Generate proto code
make proto

# Lint proto files
make proto-lint

# Build all services
make build

# Run tests
make test

# Start infrastructure only (for local development)
make docker-up

# Stop everything
make docker-down

# Register a user
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret","first_name":"John","last_name":"Doe"}'

# Login
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret"}'

# Place an order (with JWT)
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"<user_id>","items":[{"product_id":"a0000001-0000-0000-0000-000000000001","quantity":2}],"currency":"USD"}'

# Health check
curl http://localhost:8080/healthz
```
