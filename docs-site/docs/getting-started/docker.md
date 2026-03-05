---
sidebar_position: 2
title: Docker Deployment
description: Deploy Rampart with Docker — single container, Docker Compose, environment variables, health checks, and production tips.
---

# Docker Deployment

Rampart ships as a single Docker image that includes the Go server and embedded admin UI. This guide covers running Rampart with Docker, configuring it for production, and setting up health checks.

## Docker Run (Single Container)

If you already have PostgreSQL and Redis running, you can start Rampart with a single `docker run` command:

```bash
docker run -d \
  --name rampart \
  -p 8080:8080 \
  -e RAMPART_DB_URL="postgres://rampart:secret@host.docker.internal:5432/rampart?sslmode=disable" \
  -e RAMPART_REDIS_URL="redis://host.docker.internal:6379/0" \
  -e RAMPART_SIGNING_KEY_PATH="/data/keys/signing.pem" \
  -v rampart-keys:/data/keys \
  ghcr.io/manimovassagh/rampart:latest
```

## Docker Compose (Recommended)

For a complete stack, use Docker Compose. Create a `docker-compose.yml`:

```yaml
services:
  rampart:
    image: ghcr.io/manimovassagh/rampart:latest
    ports:
      - "8080:8080"
    environment:
      RAMPART_DB_URL: "postgres://rampart:secret@postgres:5432/rampart?sslmode=disable"
      RAMPART_REDIS_URL: "redis://redis:6379/0"
      RAMPART_LOG_LEVEL: "info"
      RAMPART_ISSUER_URL: "http://localhost:8080"
    volumes:
      - rampart-keys:/data/keys
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 5s
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: rampart
      POSTGRES_USER: rampart
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U rampart"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

volumes:
  pgdata:
  redisdata:
  rampart-keys:
```

Start the stack:

```bash
docker compose up -d
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_DB_URL` | PostgreSQL connection string | (required) |
| `RAMPART_REDIS_URL` | Redis connection string | (required) |
| `RAMPART_PORT` | HTTP listen port | `8080` |
| `RAMPART_ISSUER_URL` | Base URL used in OIDC Discovery and token `iss` claim | `http://localhost:8080` |
| `RAMPART_SIGNING_KEY_PATH` | Path to RSA private key for JWT signing | Auto-generated |
| `RAMPART_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `RAMPART_LOG_FORMAT` | Log format: `json`, `text` | `json` |
| `RAMPART_CORS_ORIGINS` | Comma-separated list of allowed CORS origins | `*` |
| `RAMPART_SESSION_TTL` | Session time-to-live | `24h` |
| `RAMPART_ACCESS_TOKEN_TTL` | Access token lifetime | `1h` |
| `RAMPART_REFRESH_TOKEN_TTL` | Refresh token lifetime | `7d` |

See the [Configuration reference](./configuration.md) for the full list.

## Health Checks

Rampart exposes a health check endpoint:

```
GET /health
```

Response when healthy:

```json
{
  "status": "ok"
}
```

The health check verifies connectivity to both PostgreSQL and Redis. Use this endpoint for Docker health checks, load balancer probes, and Kubernetes liveness/readiness probes.

## Volume Mounts

### Signing Keys

Rampart uses RSA keys for JWT signing. If no key is provided, one is generated on first startup and stored at the configured path. To persist keys across container restarts, mount a volume:

```bash
-v rampart-keys:/data/keys
```

In production, you should provide your own RSA private key and mount it read-only:

```bash
-v /path/to/signing.pem:/data/keys/signing.pem:ro
```

### Database Data

Mount volumes for PostgreSQL and Redis data to ensure persistence:

```bash
-v pgdata:/var/lib/postgresql/data
-v redisdata:/data
```

## Production Tips

### Use a Reverse Proxy

In production, place Rampart behind a reverse proxy (Nginx, Caddy, or Traefik) that handles TLS termination:

```nginx
server {
    listen 443 ssl http2;
    server_name auth.example.com;

    ssl_certificate /etc/ssl/certs/auth.example.com.pem;
    ssl_certificate_key /etc/ssl/private/auth.example.com.key;

    location / {
        proxy_pass http://rampart:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Set the Issuer URL

Always set `RAMPART_ISSUER_URL` to the public-facing URL of your Rampart instance. This value appears in JWT `iss` claims and OIDC Discovery responses:

```bash
RAMPART_ISSUER_URL=https://auth.example.com
```

### Restrict CORS Origins

In production, do not use the default wildcard CORS origin. Specify your application domains:

```bash
RAMPART_CORS_ORIGINS=https://app.example.com,https://admin.example.com
```

### Resource Limits

Rampart is lightweight, but you should still set resource limits in production:

```yaml
deploy:
  resources:
    limits:
      memory: 128M
      cpus: "0.5"
    reservations:
      memory: 64M
      cpus: "0.25"
```

### Database Connection Pooling

For high-traffic deployments, consider using PgBouncer between Rampart and PostgreSQL to manage connection pooling efficiently.

### Log Format

Use JSON logging in production for structured log aggregation:

```bash
RAMPART_LOG_FORMAT=json
RAMPART_LOG_LEVEL=info
```
