FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /product-service ./cmd/product-service

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /product-service /product-service
EXPOSE 50052
ENTRYPOINT ["/product-service"]
