# --- Stage 1: Builder ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependencies first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build both binaries statically
# CGO_ENABLED=0 ensures we don't need libc, making the final image tiny
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/consumer ./cmd/consumer/main.go

# --- Stage 2: Runtime ---
FROM alpine:latest

WORKDIR /root/

# Copy binaries from builder
COPY --from=builder /bin/api /usr/local/bin/api
COPY --from=builder /bin/consumer /usr/local/bin/consumer

# Copy migrations (optional, if you run migrations from the container)
COPY --from=builder /app/migrations ./migrations

# Expose API port
EXPOSE 8080

# Default command (can be overridden)
CMD ["api"]