---
sidebar_position: 2
title: Docker Deployment
description: Deploy Rampart with Docker — single container, Docker Compose, environment variables, health checks, and production tips.
---

# Docker Deployment

Rampart ships as a single Docker image that includes the Go server and embedded admin UI. This guide covers running Rampart with Docker, configuring it for production, and setting up health checks.

## Docker Run (Single Container)

If you already have PostgreSQL running, you can start Rampart with a single `docker run` command:

```bash
docker run -d \
  --name rampart \
  -p 8080:8080 \
  -e RAMPART_DB_URL="postgres://rampart:changeme@host.docker.internal:5432/rampart?sslmode=disable" \
  -e RAMPART_SIGNING_KEY_PATH="/data/rampart-signing-key.pem" \
  -v rampart-data:/data \
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
      RAMPART_DB_URL: "postgres://rampart:changeme@postgres:5432/rampart?sslmode=disable"
      RAMPART_LOG_LEVEL: "info"
      RAMPART_ISSUER: "http://localhost:8080"
      RAMPART_SIGNING_KEY_PATH: "/data/rampart-signing-key.pem"
    volumes:
      - rampart-data:/data
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 5s
    restart: unless-stopped
    read_only: true
    security_opt:
      - no-new-privileges:true
    tmpfs:
      - /tmp:noexec,nosuid,size=64m

  postgres:
    image: postgres:16.6-alpine
    environment:
      POSTGRES_DB: rampart
      POSTGRES_USER: rampart
      POSTGRES_PASSWORD: changeme
    ports:
      - "127.0.0.1:5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U rampart"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped

volumes:
  pgdata:
  rampart-data:
```

Start the stack:

```bash
docker compose up -d
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_DB_URL` | PostgreSQL connection string | (required) |
| `RAMPART_PORT` | HTTP listen port | `8080` |
| `RAMPART_ISSUER` | Base URL used in OIDC Discovery and token `iss` claim | `http://localhost:8080` |
| `RAMPART_SIGNING_KEY_PATH` | Path to RSA private key for JWT signing | Auto-generated |
| `RAMPART_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `RAMPART_LOG_FORMAT` | Log format: `json`, `text`, `pretty` | `pretty` |
| `RAMPART_ALLOWED_ORIGINS` | Comma-separated list of allowed CORS origins | — |
| `RAMPART_SESSION_TTL` | Session time-to-live | `24h` |
| `RAMPART_ACCESS_TOKEN_TTL` | Access token lifetime | `900s` |
| `RAMPART_REFRESH_TOKEN_TTL` | Refresh token lifetime | `7d` |
| `RAMPART_HSTS_ENABLED` | Enable HTTP Strict Transport Security header | `false` |
| `RAMPART_RATE_LIMIT_RPS` | Requests per second rate limit | `10` |
| `RAMPART_RATE_LIMIT_BURST` | Rate limit burst size | `20` |
| `RAMPART_SECURE_COOKIES` | Set Secure flag on cookies (enable behind TLS) | `false` |

See the [Configuration reference](./configuration.md) for the full list.

## Health Checks

Rampart exposes a health check endpoint:

```
GET /healthz
```

Response when healthy:

```json
{
  "status": "ok"
}
```

The health check verifies connectivity to PostgreSQL. Use this endpoint for Docker health checks, load balancer probes, and Kubernetes liveness/readiness probes. A readiness probe is also available at `GET /readyz`.

## Volume Mounts

### Signing Keys

Rampart uses RSA keys for JWT signing. If no key is provided, one is generated on first startup and stored at the configured path (`/data/rampart-signing-key.pem` by default in Docker). To persist keys across container restarts, mount a volume:

```bash
-v rampart-data:/data
```

In production, you should provide your own RSA private key and mount it read-only:

```bash
-v /path/to/signing.pem:/data/rampart-signing-key.pem:ro
```

### Database Data

Mount a volume for PostgreSQL data to ensure persistence:

```bash
-v pgdata:/var/lib/postgresql/data
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

Always set `RAMPART_ISSUER` to the public-facing URL of your Rampart instance. This value appears in JWT `iss` claims and OIDC Discovery responses:

```bash
RAMPART_ISSUER=https://auth.example.com
```

### Restrict CORS Origins

In production, specify your application domains explicitly:

```bash
RAMPART_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
```

### Resource Limits

Rampart is lightweight, but you should still set resource limits in production:

```yaml
deploy:
  resources:
    limits:
      memory: 256M
      cpus: "1.0"
    reservations:
      memory: 64M
      cpus: "0.25"
```

### Security Settings

Harden the Rampart container with a read-only filesystem and prevent privilege escalation:

```yaml
read_only: true
security_opt:
  - no-new-privileges:true
tmpfs:
  - /tmp:noexec,nosuid,size=64m
```

The `read_only` flag makes the container filesystem immutable. The `no-new-privileges` option prevents processes from gaining additional privileges via `setuid`/`setgid`. A `tmpfs` mount provides a writable scratch space for temporary files without persisting anything to disk.

### Database Connection Pooling

For high-traffic deployments, consider using PgBouncer between Rampart and PostgreSQL to manage connection pooling efficiently.

### Log Format

The default log format is `pretty` for local development. Use JSON logging in production for structured log aggregation:

```bash
RAMPART_LOG_FORMAT=json
RAMPART_LOG_LEVEL=info
```
