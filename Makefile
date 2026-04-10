.PHONY: proto proto-lint build test run-user run-product run-order run-payment run-notification run-gateway docker-up docker-down docker-build docker-all

BUF := $(shell go env GOPATH)/bin/buf
SERVICES := gateway user-service product-service order-service payment-service notification-service bridge-service

proto:
	$(BUF) generate

proto-lint:
	$(BUF) lint

build:
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		go build -o bin/$$svc ./cmd/$$svc; \
	done

test:
	go test ./... -race -cover -count=1

run-user:
	go run ./cmd/user-service

run-product:
	go run ./cmd/product-service

run-order:
	go run ./cmd/order-service

run-payment:
	go run ./cmd/payment-service

run-notification:
	go run ./cmd/notification-service

run-gateway:
	go run ./cmd/gateway

run-bridge:
	go run ./cmd/bridge-service

# Infrastructure only (databases, redis, nats, rabbitmq)
docker-up:
	docker compose up -d postgres-user postgres-product postgres-order postgres-payment postgres-notification redis nats rabbitmq

docker-down:
	docker compose down -v

# Build Docker images for all services
docker-build:
	@for svc in $(SERVICES); do \
		echo "Building Docker image for $$svc..."; \
		docker build -f deployments/docker/$$svc.Dockerfile -t go-grpc-http/$$svc:latest .; \
	done

# Start everything (infrastructure + services + frontend) in Docker
docker-all:
	docker compose up -d --build
