# Phase 5: Polish - Setup Reference

## What Was Added

### 1. Multi-Stage Dockerfiles (6 files)

All services get production-ready Docker images using the same two-stage pattern:

```dockerfile
# Stage 1: Build with full Go toolchain
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download          # cached layer for dependencies
COPY . .
RUN CGO_ENABLED=0 go build -o /service ./cmd/service

# Stage 2: Minimal runtime (no Go toolchain, ~10MB)
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /service /service
ENTRYPOINT ["/service"]
```

**Why multi-stage:** The builder image is ~1GB (Go toolchain), but the final image is ~15MB (just the binary + CA certs). `CGO_ENABLED=0` produces a fully static binary that runs on Alpine without glibc.

Files created:
- `deployments/docker/gateway.Dockerfile`
- `deployments/docker/user-service.Dockerfile`
- `deployments/docker/product-service.Dockerfile`
- `deployments/docker/order-service.Dockerfile`
- `deployments/docker/payment-service.Dockerfile`
- `deployments/docker/notification-service.Dockerfile`

### 2. gRPC Health Checks

Every gRPC service implements the standard `grpc.health.v1.Health/Check` RPC.

**`pkg/health/health.go`** -- Generic health server that accepts checker functions. Each service passes its own dependency checks:
- user-service: PostgreSQL ping
- product-service: PostgreSQL ping
- order-service: PostgreSQL ping + Redis ping
- payment-service: PostgreSQL ping

**Why:** Kubernetes, Docker health checks, and load balancers use the gRPC Health Check protocol to determine if a service is ready to receive traffic. Without this, a service with a dead database connection would still accept requests and return errors.

### 3. Gateway `/healthz` Endpoint

The API gateway exposes `GET /healthz` returning `{"status":"ok"}` for HTTP-based health monitoring (load balancers, uptime monitors).

### 4. Domain Error Helper

**`pkg/errors/errors.go`** -- Maps domain error messages to gRPC status codes automatically. Pattern-matches on error strings: "not found" -> NotFound, "already registered" -> AlreadyExists, "insufficient" -> FailedPrecondition, etc.

### 5. Full Docker Compose with All Services

Updated `docker-compose.yml` now includes all 13 containers:
- 5 PostgreSQL instances (one per service)
- 1 Redis (order read model)
- 1 NATS JetStream (event bus)
- 6 application services (built from Dockerfiles)

Services have proper `depends_on` with health check conditions so databases are ready before services start.

### 6. Makefile Improvements

New targets:
- `make docker-build` -- Build all 6 Docker images locally
- `make docker-all` -- Start everything (infra + services) via Docker Compose
- `make docker-up` -- Start infrastructure only (for local development)
- `SERVICES` variable for DRY service list

## Commands

```bash
# Build all Go binaries
make build

# Build all Docker images
make docker-build

# Start everything in Docker (production-like)
make docker-all

# Start infrastructure only (for local dev with go run)
make docker-up

# Run tests
make test

# Check health
curl http://localhost:8080/healthz
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

## Files Created

- `deployments/docker/*.Dockerfile` -- 6 multi-stage Dockerfiles
- `pkg/health/health.go` -- gRPC health check server
- `pkg/errors/errors.go` -- Domain-to-gRPC error mapping

## Files Modified

- `cmd/user-service/main.go` -- Registered health server
- `cmd/product-service/main.go` -- Registered health server
- `cmd/order-service/main.go` -- Registered health server (PG + Redis)
- `cmd/payment-service/main.go` -- Registered health server
- `internal/gateway/server.go` -- Added `/healthz` HTTP endpoint
- `pkg/auth/interceptor.go` -- Added health check to public methods
- `docker-compose.yml` -- Added all 6 app services with proper depends_on
- `Makefile` -- Added docker-build, docker-all targets

## Verified

- Gateway `/healthz` returns `{"status":"ok"}`
- All 4 gRPC services return `SERVING` on health check
- Full E2E flow works: register -> login -> create product -> place order -> auto-payment -> order status=paid -> stock decremented
