# Build stage
FROM golang:1.25.7-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /rampart ./cmd/rampart

# Runtime stage — distroless nonroot (~15MB)
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /rampart /rampart
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/rampart"]
