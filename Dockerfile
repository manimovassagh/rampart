# Build stage
FROM golang:1.26.1-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-trimpath \
    go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /rampart ./cmd/rampart

# Runtime stage — alpine (~8MB, supports shell for permission setup)
FROM alpine:3.21

# OCI image labels
LABEL org.opencontainers.image.title="Rampart" \
      org.opencontainers.image.description="Lightweight, modern identity and access management server" \
      org.opencontainers.image.source="https://github.com/manimovassagh/rampart" \
      org.opencontainers.image.vendor="Rampart" \
      org.opencontainers.image.licenses="AGPL-3.0"

# Install minimal runtime dependencies and clean cache in same layer
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 rampart \
    && mkdir -p /data \
    && chown rampart:rampart /data

COPY --from=builder /rampart /rampart
COPY --from=builder /app/migrations /migrations

# Security: limit Go crash output, read-only friendly
ENV GOTRACEBACK=single

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

USER rampart

ENTRYPOINT ["/rampart"]
