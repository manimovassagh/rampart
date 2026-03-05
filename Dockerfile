# Build stage
FROM golang:1.25.7-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /rampart ./cmd/rampart

# Runtime stage — alpine (~8MB, supports shell for permission setup)
FROM alpine:3.21

RUN adduser -D -u 10001 rampart && mkdir -p /data && chown rampart:rampart /data

COPY --from=builder /rampart /rampart
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

USER rampart

ENTRYPOINT ["/rampart"]
