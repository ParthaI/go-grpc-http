# Phase 1: Foundation - Setup Reference

## Commands Executed

### 1. Project Initialization

```bash
go mod init github.com/parthasarathi/go-grpc-http
```

**Why:** Every Go project needs a module file (`go.mod`) that declares the module path and tracks dependencies. This creates the root module so all packages within the project can import each other using `github.com/parthasarathi/go-grpc-http/...` as the base path.

```bash
go install github.com/bufbuild/buf/cmd/buf@latest
```

**Why:** `buf` is a modern replacement for the traditional `protoc` compiler. We use it because it provides proto linting (catches naming convention issues early), code generation with a simple YAML config (no long `protoc` flags), dependency management for third-party proto files (like Google API annotations), and breaking change detection against previous versions. Without `buf`, we would need to manually download `googleapis` proto files and manage complex `protoc` invocations.

```bash
mkdir -p proto/ gen/ cmd/ internal/ pkg/ migrations/ deployments/ scripts/
```

**Why:** This follows the standard Go project layout convention:
- `proto/` -- proto source files, organized by service and version (`user/v1/`)
- `gen/` -- generated code output from buf (committed to git so consumers don't need buf)
- `cmd/` -- one `main.go` per deployable binary (gateway, user-service, etc.)
- `internal/` -- private code that cannot be imported by other modules (enforced by Go)
- `pkg/` -- shared libraries importable by all services within this module
- `migrations/` -- SQL migration files per service
- `deployments/` -- Dockerfiles and deployment configs

### 2. Proto Code Generation

```bash
buf dep update
```

**Why:** Downloads the third-party proto dependencies declared in `buf.yaml` -- specifically `buf.build/googleapis/googleapis` which provides `google/api/annotations.proto` needed for the gRPC-gateway HTTP annotations (e.g., `option (google.api.http) = { get: "/api/v1/users/{user_id}" }`). Without this, our proto files would fail to compile because they import Google's HTTP annotation definitions.

```bash
buf lint
```

**Why:** Validates proto files against the `STANDARD` rule set -- catches issues like unused imports, incorrect naming conventions (proto uses `snake_case` for fields, `PascalCase` for messages), missing package declarations, and more. Running this before generation prevents generating code from malformed proto files.

```bash
buf generate
```

**Why:** Reads `buf.gen.yaml` and runs four code generation plugins:
1. `protocolbuffers/go` -- generates Go structs for all proto messages (`*.pb.go`)
2. `grpc/go` -- generates gRPC server/client interfaces and stubs (`*_grpc.pb.go`)
3. `grpc-ecosystem/gateway` -- generates HTTP reverse-proxy handlers that translate REST calls to gRPC (`*.pb.gw.go`). This is the core of the API Gateway pattern
4. `grpc-ecosystem/openapiv2` -- generates Swagger/OpenAPI specs from proto definitions (`*.swagger.json`) for API documentation

### 3. Go Dependency Installation

```bash
go get google.golang.org/grpc \
  google.golang.org/protobuf \
  github.com/grpc-ecosystem/grpc-gateway/v2/runtime \
  github.com/jackc/pgx/v5 \
  github.com/golang-jwt/jwt/v5 \
  github.com/google/uuid \
  golang.org/x/crypto \
  github.com/grpc-ecosystem/go-grpc-middleware/v2 \
  google.golang.org/genproto/googleapis/api
```

**Why:** Installs the Go libraries that the generated and hand-written code imports. Each module serves a specific role in the architecture (see the Modules section below for individual explanations).

```bash
go mod tidy
```

**Why:** Ensures `go.mod` and `go.sum` are in sync with the actual imports in the codebase. Removes any modules we fetched with `go get` but don't actually import, and adds any transitive dependencies that are needed. This is the standard way to keep the dependency graph clean.

### 4. Build and Verify

```bash
go build ./...
```

**Why:** Compiles all packages in the module without producing binaries. This is a quick check that all code compiles, all imports resolve, and there are no type errors. The `./...` pattern means "all packages recursively."

```bash
docker compose up -d
```

**Why:** Starts the PostgreSQL 16 container defined in `docker-compose.yml`. The user-service needs a running PostgreSQL instance to store user data. The `-d` flag runs it in detached (background) mode. The container is configured with port `5433` (not the default `5432`) to avoid conflicts with any local PostgreSQL installation.

```bash
go run ./cmd/user-service
```

**Why:** Starts the user-service gRPC server on port `50051`. On startup it connects to PostgreSQL, runs the auto-migration to create the `users` table if it doesn't exist, and registers the `UserService` gRPC server with auth, logging, and recovery interceptors.

```bash
HTTP_PORT=9090 go run ./cmd/gateway
```

**Why:** Starts the API gateway on port `9090` (we used `9090` instead of the default `8080` because another process was using that port). The gateway connects to the user-service at `localhost:50051` and registers the grpc-gateway HTTP handlers that translate incoming REST/JSON requests into gRPC calls.

### 5. End-to-End Verification

```bash
curl -s -X POST http://localhost:9090/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123","first_name":"John","last_name":"Doe"}'
```

**Why:** Tests the full request path: REST request hits the gateway, gateway translates it to a gRPC `Register` call, user-service hashes the password with bcrypt, generates a UUID, stores the user in PostgreSQL, and returns the response which the gateway translates back to JSON.

```bash
curl -s -X POST http://localhost:9090/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret123"}'
```

**Why:** Tests authentication flow. The user-service looks up the user by email, verifies the bcrypt password hash, generates a JWT token with 24-hour expiry, and returns it. This token is required for all subsequent authenticated requests.

```bash
curl -s http://localhost:9090/api/v1/users/{user_id} \
  -H "Authorization: Bearer {access_token}"
```

**Why:** Tests the JWT auth interceptor. The gRPC auth interceptor extracts the `Bearer` token from the `Authorization` metadata, validates it using the shared JWT secret, and injects the claims into the request context. If the token is missing or invalid, it returns `UNAUTHENTICATED`.

```bash
curl -s -X PUT http://localhost:9090/api/v1/users/{user_id} \
  -H "Authorization: Bearer {access_token}" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Jane","last_name":"Smith"}'
```

**Why:** Tests the full authenticated update flow. Also verifies the PostgreSQL UPDATE query with row-count checking (returns `NOT_FOUND` if the user ID doesn't exist).

---

## Go Modules Downloaded

### Direct Dependencies

**`google.golang.org/grpc`** -- v1.80.0

The core gRPC framework for Go. Provides the server and client runtime for defining, registering, and calling gRPC services. Every microservice in this project runs a gRPC server using this library. We need it because gRPC is the chosen inter-service communication protocol -- it provides strong typing via proto definitions, HTTP/2 transport, streaming support, and built-in interceptor chains for cross-cutting concerns like auth and logging.

**`google.golang.org/protobuf`** -- v1.36.11

The Go runtime library for Protocol Buffers. Required by the generated `*.pb.go` files to serialize/deserialize proto messages. Every proto message we define (RegisterRequest, LoginResponse, etc.) compiles to Go structs that depend on this library for marshaling to/from the wire format.

**`github.com/grpc-ecosystem/grpc-gateway/v2`** -- v2.28.0

Generates a reverse-proxy server that translates RESTful JSON API calls into gRPC calls. This is the core of our API Gateway pattern -- instead of manually writing an HTTP server that calls gRPC, we annotate proto files with `google.api.http` options and the gateway auto-generates the HTTP handlers. This means our proto files are the single source of truth for both the gRPC API and the REST API.

**`github.com/jackc/pgx/v5`** -- v5.9.1

A high-performance, pure Go PostgreSQL driver and connection pool. Chosen over the standard `database/sql` + `lib/pq` combination because pgx offers significantly better performance (native PostgreSQL protocol, no `database/sql` abstraction overhead), built-in connection pooling via `pgxpool`, support for PostgreSQL-specific types (JSONB, arrays, TIMESTAMPTZ), and `COPY` protocol support for bulk inserts. Each microservice uses its own pgxpool connection to its dedicated PostgreSQL database.

**`github.com/golang-jwt/jwt/v5`** -- v5.3.1

A Go implementation of JSON Web Tokens (RFC 7519). Used for stateless authentication -- the user-service generates a signed JWT on login containing the user ID and email, and the auth interceptor validates it on every subsequent request. We chose JWT because it's stateless (no session store needed), the gateway can validate tokens without calling the user-service, and the claims can carry user identity information that downstream services need.

**`github.com/google/uuid`** -- v1.6.0

Generates RFC 4122 UUIDs. Used as primary keys for all database entities (users, products, orders, etc.) instead of auto-incrementing integers. UUIDs are preferred in microservice architectures because they can be generated without coordinating with the database, they don't leak information about record count or creation order, and they're globally unique across services.

**`golang.org/x/crypto`** -- v0.49.0

Provides the `bcrypt` package for password hashing. Bcrypt is the industry standard for password storage -- it's intentionally slow (configurable cost factor) to resist brute-force attacks, automatically handles salt generation, and is immune to rainbow table attacks. We use it in the user-service to hash passwords on registration and verify them on login.

**`google.golang.org/genproto/googleapis/api`** -- v0.0.0-20260406

Contains the Go types for Google API annotations, specifically the `google.api.http` annotation used in proto files. The generated gateway code (`*.pb.gw.go`) imports these types to know how to map HTTP methods and URL paths to gRPC methods. Without this, the grpc-gateway generated code would not compile.

### Indirect Dependencies

These are pulled in automatically as transitive dependencies:

- **`github.com/jackc/pgpassfile`** -- Parses PostgreSQL `.pgpass` password files (used by pgx for credential lookup)
- **`github.com/jackc/pgservicefile`** -- Parses PostgreSQL service files (used by pgx for connection config)
- **`github.com/jackc/puddle/v2`** -- Generic resource pool used internally by pgxpool for connection management
- **`golang.org/x/net`** -- Extended networking libraries, required by gRPC for HTTP/2 transport
- **`golang.org/x/sync`** -- Concurrency primitives (semaphore, errgroup), used by pgxpool for connection limiting
- **`golang.org/x/sys`** -- Low-level OS interface, required by the crypto package for system calls
- **`golang.org/x/text`** -- Unicode text processing, required by gRPC for string handling
- **`google.golang.org/genproto/googleapis/rpc`** -- Go types for gRPC status codes and error details

### CLI Tools Installed

**`buf`** -- v1.67.0

Modern Protocol Buffers toolchain that replaces the traditional `protoc` compiler. We use it for three reasons: (1) dependency management -- it downloads third-party proto files (Google APIs) automatically, (2) linting -- enforces proto style conventions before code generation, (3) simplified codegen -- a single YAML config replaces complex `protoc` command lines with multiple plugin flags.

---

## Files Written

### Configuration Files

**`go.mod`** -- Declares the Go module path and all dependencies with exact versions. This is the root of the Go module system -- it ensures reproducible builds by pinning every dependency version.

**`buf.yaml`** -- Configures buf: declares `proto/` as the module path, lists proto dependencies (googleapis, grpc-gateway), and sets lint rules to `STANDARD`. This is the equivalent of a `package.json` for proto files.

**`buf.gen.yaml`** -- Tells buf which code generation plugins to run and where to output the results. Maps four remote plugins (protobuf/go, grpc/go, gateway, openapiv2) to their output directories.

**`docker-compose.yml`** -- Defines the PostgreSQL 16 container for the user-service database. Uses Alpine variant for smaller image size, maps port 5433 to avoid local conflicts, includes a healthcheck, and uses a named volume for data persistence across restarts.

**`Makefile`** -- Provides shorthand commands: `make proto` for code generation, `make build` for compiling all services, `make run-user` / `make run-gateway` for development, `make docker-up` / `make docker-down` for infrastructure.

**`.gitignore`** -- Excludes build artifacts (`bin/`), test binaries, `.env` files (contain secrets), and vendor directory from version control.

**`.env.example`** -- Documents all environment variables the services expect, with safe default values. Developers copy this to `.env` and customize for their environment.

### Proto Definitions

**`proto/common/v1/common.proto`** -- Shared message types (Pagination, PaginationResponse, Money) that will be reused across all service protos. The `/v1/` versioning allows breaking changes via `/v2/` in the future without affecting existing clients.

**`proto/user/v1/user.proto`** -- Defines the `UserService` with four RPCs (Register, Login, GetUser, UpdateUser). Each RPC has `google.api.http` annotations that tell grpc-gateway how to map HTTP methods/paths to gRPC calls. Also defines all request/response message types. This single file is the source of truth for the user API -- both gRPC and REST.

### Generated Code (via `buf generate`)

**`gen/go/common/v1/common.pb.go`** -- Go structs for Pagination, Money, etc. Auto-generated -- do not edit.

**`gen/go/user/v1/user.pb.go`** -- Go structs for all UserService request/response messages. Auto-generated.

**`gen/go/user/v1/user_grpc.pb.go`** -- The `UserServiceServer` interface that our server must implement, and the `UserServiceClient` interface for calling the service. Auto-generated.

**`gen/go/user/v1/user.pb.gw.go`** -- HTTP handlers that parse REST requests, convert them to proto messages, call the gRPC service, and return JSON responses. This is what the gateway registers to translate REST to gRPC. Auto-generated.

**`gen/openapiv2/user/v1/user.swagger.json`** -- OpenAPI 2.0 spec for the user API. Can be served via Swagger UI for interactive API documentation. Auto-generated.

### Shared Libraries (`pkg/`)

**`pkg/auth/jwt.go`** -- JWT token generation (called by user-service on login) and validation (called by the auth interceptor on every request). Encapsulates the signing key management, claims structure, expiry handling, and signature verification in one place so all services use consistent JWT handling.

**`pkg/auth/interceptor.go`** -- gRPC unary interceptor that runs before every RPC handler. It checks if the method is public (Register, Login are exempt), extracts the JWT from the `Authorization` metadata header, validates it, and injects the claims into the Go context. Downstream handlers can then call `ClaimsFromContext(ctx)` to get the authenticated user's ID and email.

**`pkg/database/postgres.go`** -- Creates a `pgxpool.Pool` connection pool from a database URL. Wraps the connection setup, config parsing, and initial ping into a single function. Every service calls this on startup to get a shared connection pool.

**`pkg/observability/logging.go`** -- Creates a structured JSON logger using Go's built-in `slog` package. Every log line includes the service name, is machine-parseable JSON (suitable for log aggregation tools like ELK or Loki), and uses the standard slog levels (INFO, ERROR, etc.).

**`pkg/interceptors/logging.go`** -- gRPC interceptor that logs every RPC call with the method name, response status code, and duration. Essential for debugging and monitoring -- you can see exactly which RPCs are being called, how long they take, and whether they succeeded or failed.

**`pkg/interceptors/recovery.go`** -- gRPC interceptor that catches panics in RPC handlers, logs the stack trace, and returns an `INTERNAL` error instead of crashing the entire service. Without this, a nil pointer dereference in any handler would kill the gRPC server process.

### User Service (`internal/user/`)

**`internal/user/model/user.go`** -- The domain model struct. Separates the internal representation from the proto-generated types, allowing the domain model to evolve independently of the API contract.

**`internal/user/repository/user_repository.go`** -- The repository interface. Defines the data access contract (Create, GetByID, GetByEmail, Update) that the service layer depends on. Using an interface here allows swapping the PostgreSQL implementation for an in-memory implementation in tests.

**`internal/user/repository/postgres.go`** -- PostgreSQL implementation of the repository. Contains the actual SQL queries. Uses parameterized queries (`$1`, `$2`) to prevent SQL injection. Returns `nil` for not-found cases (instead of errors) to let the service layer decide the appropriate error response.

**`internal/user/service/user_service.go`** -- Business logic layer. Handles email uniqueness checks, password hashing, UUID generation, JWT token creation, and coordinates between the repository and auth packages. This is where domain rules live -- the gRPC server layer just translates proto messages to/from service calls.

**`internal/user/server.go`** -- Implements the `UserServiceServer` gRPC interface generated from the proto file. Translates between proto request/response types and the service layer, and maps domain errors to appropriate gRPC status codes (AlreadyExists, NotFound, Unauthenticated, etc.).

### API Gateway (`internal/gateway/`)

**`internal/gateway/server.go`** -- Sets up the grpc-gateway HTTP multiplexer, registers the UserService reverse proxy handler (and later product, order, payment services), adds CORS middleware for browser access, and starts the HTTP server. This is the single entry point for all external REST clients.

### Service Entrypoints (`cmd/`)

**`cmd/user-service/main.go`** -- Wires everything together: reads config from environment variables, connects to PostgreSQL, runs the auto-migration, creates the JWT manager / repository / service / server chain, configures the gRPC server with interceptors (recovery, logging, auth), and handles graceful shutdown on SIGINT/SIGTERM.

**`cmd/gateway/main.go`** -- Reads gateway config from environment variables and starts the HTTP gateway server. Kept intentionally simple -- all logic lives in `internal/gateway/`.

### Database Migrations

**`migrations/user/001_create_users.up.sql`** -- Creates the `users` table with UUID primary key, unique email constraint, bcrypt password hash storage, name fields, and timestamps. Includes an index on email for fast login lookups.

**`migrations/user/001_create_users.down.sql`** -- Drops the users table. Used for rolling back the migration if needed.
