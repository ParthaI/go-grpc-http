# Phase 2: Product Service - Setup Reference

## Commands Executed

### 1. Proto Definition and Code Generation

```bash
buf lint
buf generate
```

**Why:** We defined `proto/product/v1/product.proto` with 7 RPCs -- 5 public (HTTP-annotated) and 2 internal (ReserveStock, ReleaseStock with no HTTP annotation). The internal RPCs are only callable via gRPC from other services (order-service), not through the REST gateway. Running `buf generate` produces the Go stubs, gateway handlers, and Swagger spec.

### 2. Start Product Database

```bash
docker compose up -d
```

**Why:** Adds a second PostgreSQL container (`postgres-product` on port 5434) dedicated to the product-service. This follows the Database per Service pattern -- each service owns its data and no other service can access it directly.

### 3. Build and Run

```bash
go build ./...
go run ./cmd/user-service &
go run ./cmd/product-service &
HTTP_PORT=9090 go run ./cmd/gateway &
```

**Why:** Three processes needed: user-service (for JWT auth), product-service (new), and gateway (now registers both services). The gateway was updated to proxy product REST calls to the product gRPC service.

### 4. Install grpcurl

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

**Why:** grpcurl is a command-line gRPC client (like curl for gRPC). We needed it to test the internal `ReserveStock` and `ReleaseStock` RPCs that have no HTTP endpoint -- they can only be called via gRPC directly. In production, these would be called by the order-service.

### 5. End-to-End Verification

```bash
# Create product (authenticated)
curl -s -X POST http://localhost:9090/api/v1/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Mechanical Keyboard","description":"Cherry MX Blue","price_cents":12999,"currency":"USD","stock_quantity":50,"sku":"KB-MECH-001"}'

# Get product (public - no auth needed)
curl -s http://localhost:9090/api/v1/products/{product_id}

# List products with pagination (public)
curl -s "http://localhost:9090/api/v1/products"

# Update product details (authenticated)
curl -s -X PUT http://localhost:9090/api/v1/products/{product_id} \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Keyboard Pro","description":"Updated","price_cents":14999,"currency":"USD"}'

# Update inventory quantity (authenticated)
curl -s -X PUT http://localhost:9090/api/v1/products/{product_id}/inventory \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"quantity":75}'

# Reserve stock via internal gRPC (service-to-service)
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"product_id":"...","quantity":5,"order_id":"order-1"}' \
  localhost:50052 product.v1.ProductService/ReserveStock

# Release stock via internal gRPC
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"product_id":"...","quantity":3,"order_id":"order-1"}' \
  localhost:50052 product.v1.ProductService/ReleaseStock
```

**Why:** Tests all 7 RPCs: 5 through the REST gateway (CreateProduct, GetProduct, ListProducts, UpdateProduct, UpdateInventory) and 2 through direct gRPC (ReserveStock, ReleaseStock). Stock math was verified: 75 - 5 + 3 = 73.

---

## Go Modules Downloaded

No new direct dependencies were added in Phase 2. The product-service uses the same libraries already installed in Phase 1 (grpc, pgx, jwt, uuid). One CLI tool was added:

**`grpcurl`** (CLI tool, not a project dependency)

**Why:** Needed to test the internal gRPC RPCs (ReserveStock, ReleaseStock) that have no REST endpoint. These RPCs are intentionally hidden from the gateway because they should only be called by other microservices (the order-service in Phase 3), not by external clients.

---

## Files Written

### Proto Definition

**`proto/product/v1/product.proto`**

**Why:** Defines the ProductService with 7 RPCs. The first 5 have `google.api.http` annotations so grpc-gateway generates REST endpoints. The last 2 (ReserveStock, ReleaseStock) deliberately have no HTTP annotation -- this demonstrates the pattern where some RPCs are internal-only, callable by other services via gRPC but not exposed to external clients through REST.

### Generated Code (via `buf generate`)

**`gen/go/product/v1/product.pb.go`** -- Go structs for all product request/response messages.

**`gen/go/product/v1/product_grpc.pb.go`** -- The `ProductServiceServer` interface and `ProductServiceClient` for gRPC. The order-service will import and use `ProductServiceClient` to call ReserveStock in Phase 3.

**`gen/go/product/v1/product.pb.gw.go`** -- HTTP reverse-proxy handlers for the 5 public RPCs. Registered in the gateway.

### Product Domain Layer

**`internal/product/model/product.go`**

**Why:** Domain model struct with all product fields. Separate from proto types so the internal representation can differ from the API contract (e.g., `SKU` field name vs proto's `sku`).

**`internal/product/repository/product_repository.go`**

**Why:** Repository interface with 7 methods matching the 7 RPCs. `ReserveStock` uses an atomic SQL conditional update (`WHERE stock_quantity >= $1`) to prevent overselling -- this is a key concurrency pattern for inventory management.

**`internal/product/repository/postgres.go`**

**Why:** PostgreSQL implementation. Key design decisions:
- `ReserveStock` uses `UPDATE ... WHERE stock_quantity >= $1 RETURNING stock_quantity` -- this is an atomic operation that checks stock and decrements in a single query, preventing race conditions where two orders could oversell the same product.
- `ReleaseStock` uses `UPDATE ... SET stock_quantity = stock_quantity + $1` to return reserved stock when an order is cancelled.
- `List` uses `COUNT(*)` + `LIMIT/OFFSET` for pagination, ordered by `created_at DESC`.

**`internal/product/service/product_service.go`**

**Why:** Business logic layer. Coordinates UUID generation, validation, and repository calls. Thin for now but will grow as business rules increase (e.g., minimum stock thresholds, price change validation).

**`internal/product/server.go`**

**Why:** Implements the `ProductServiceServer` gRPC interface. Translates between proto types and service layer, maps errors to gRPC status codes. Returns `success: false` (not an error) for insufficient stock on ReserveStock -- this lets the caller (order-service) handle the business logic of a failed reservation.

### Service Entrypoint

**`cmd/product-service/main.go`**

**Why:** Wires up product-service: connects to `postgres-product:5434`, runs auto-migration, creates the gRPC server with the same interceptor chain as user-service (recovery, logging, auth). Listens on port `50052`.

### Database Migration

**`migrations/product/001_create_products.up.sql`**

**Why:** Creates the products table with a `UNIQUE` constraint on `sku` (prevents duplicate product SKUs), a `BIGINT` for `price_cents` (avoids floating-point money issues), and indexes on `sku` (fast lookups) and `created_at DESC` (fast listing).

**`migrations/product/001_create_products.down.sql`**

**Why:** Drops the products table for migration rollback.

### Modified Files

**`internal/gateway/server.go`**

**Why:** Added `ProductServiceAddr` to the Config struct and registered the product service grpc-gateway handler. The gateway now proxies both user and product REST calls to their respective gRPC services.

**`cmd/gateway/main.go`**

**Why:** Added `PRODUCT_SERVICE_ADDR` env var to gateway config (defaults to `localhost:50052`).

**`pkg/auth/interceptor.go`**

**Why:** Added `GetProduct` and `ListProducts` to the public methods list. These endpoints should be accessible without authentication (browsing products doesn't require login). CreateProduct, UpdateProduct, UpdateInventory, ReserveStock, and ReleaseStock still require JWT auth.

**`docker-compose.yml`**

**Why:** Added `postgres-product` container on port `5434` with its own volume. Follows Database per Service -- the product database is completely isolated from the user database.

**`Makefile`**

**Why:** Added `run-product` target and `product-service` to the build list.
