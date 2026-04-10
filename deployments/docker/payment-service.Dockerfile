FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /payment-service ./cmd/payment-service

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /payment-service /payment-service
EXPOSE 50054
ENTRYPOINT ["/payment-service"]
