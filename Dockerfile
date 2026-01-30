FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all services
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main-api cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main-worker cmd/worker/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main-consumer cmd/consumer/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main-payment cmd/payment/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main-ticket cmd/ticket/main.go

FROM alpine:3.18

WORKDIR /app

# Copy all binaries
COPY --from=builder /app/main-api .
COPY --from=builder /app/main-worker .
COPY --from=builder /app/main-consumer .
COPY --from=builder /app/main-payment .
COPY --from=builder /app/main-ticket .

# Copy config and migrations
# Repo keeps config.example.yaml tracked; config.yaml is expected to be local-only.
COPY --from=builder /app/config.example.yaml ./config.yaml
COPY --from=builder /app/migrations ./migrations

# Expose API port
EXPOSE 8080

# Default command (overridden by docker-compose)
CMD ["./main-api"]
