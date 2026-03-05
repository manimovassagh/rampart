---
name: rampart-docker-quickstart
description: Spin up a Rampart IAM server with Docker Compose. Creates docker-compose.yml with Rampart, PostgreSQL, and Redis, ready to use as your auth provider. Use when you need a local Rampart instance for development.
argument-hint: [project-name]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Spin Up Rampart with Docker Compose

Get a fully working Rampart IAM server running locally in seconds.

## What This Skill Does

1. Creates a `docker-compose.yml` with Rampart, PostgreSQL, and Redis
2. Configures networking and health checks
3. Creates a default admin user
4. Exposes the admin console and all OIDC endpoints
5. Registers an OAuth client for your app

## Step-by-Step

### 1. Create docker-compose.yml

Create `docker-compose.yml` (or `rampart/docker-compose.yml` to keep it separate):

```yaml
version: "3.9"

services:
  rampart:
    image: ghcr.io/manimovassagh/rampart:latest
    # Or build from source:
    # build: https://github.com/manimovassagh/rampart.git
    ports:
      - "8080:8080"
    environment:
      RAMPART_DB_URL: postgres://rampart:rampart@postgres:5432/rampart?sslmode=disable
      RAMPART_REDIS_URL: redis://redis:6379/0
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/healthz"]
      interval: 5s
      timeout: 3s
      retries: 10

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: rampart
      POSTGRES_PASSWORD: rampart
      POSTGRES_DB: rampart
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "rampart"]
      interval: 5s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  pgdata:
```

### 2. Start the stack

```bash
docker compose up -d
```

Wait for Rampart to be healthy:

```bash
docker compose exec rampart wget -qO- http://localhost:8080/healthz
# Should return: {"status":"alive"}
```

### 3. Create an admin user

```bash
curl -s http://localhost:8080/register -H 'Content-Type: application/json' \
  -d '{"username":"admin","email":"admin@example.com","password":"AdminP@ss123","given_name":"Admin","family_name":"User"}'
```

### 4. Access the admin console

Open http://localhost:8080/admin/login and sign in with `admin@example.com` / `AdminP@ss123`.

### 5. Register your app as an OAuth client

In the admin console: **Clients > Create Client**

| Field | Value |
|-------|-------|
| Name | $ARGUMENTS or "My App" |
| Client ID | `my-app` |
| Type | Public (for SPAs) or Confidential (for server apps) |
| Redirect URI | `http://localhost:3000/callback` (adjust to your app's port) |

### 6. OIDC Discovery

Your app can discover all endpoints at:

```
http://localhost:8080/.well-known/openid-configuration
```

Key endpoints:
- **Authorization**: `http://localhost:8080/oauth/authorize`
- **Token**: `http://localhost:8080/oauth/token`
- **UserInfo**: `http://localhost:8080/me`
- **JWKS**: `http://localhost:8080/.well-known/jwks.json`

### 7. Add to .gitignore

```
# Rampart local data
pgdata/
```

### 8. Environment variables for your app

Add to your app's `.env`:

```bash
RAMPART_ISSUER=http://localhost:8080
RAMPART_CLIENT_ID=my-app
```

## Quick Test

```bash
# Register a test user
curl -s localhost:8080/register -H 'Content-Type: application/json' \
  -d '{"username":"testuser","email":"test@example.com","password":"Test1234!","given_name":"Test","family_name":"User"}'

# Login and get tokens
curl -s localhost:8080/login -H 'Content-Type: application/json' \
  -d '{"identifier":"test@example.com","password":"Test1234!"}'

# Use the access_token from above
curl -s localhost:8080/me -H 'Authorization: Bearer <access_token>'
```

## Next Steps

After Rampart is running, use one of the stack-specific skills to integrate auth into your app:

- `/rampart-node-setup` — Express/Node.js backend
- `/rampart-react-setup` — React SPA
- `/rampart-nextjs-setup` — Next.js full-stack
- `/rampart-python-setup` — FastAPI or Flask
- `/rampart-go-setup` — Go backend
- `/rampart-spring-setup` — Spring Boot (Java/Kotlin)

## Checklist

- [ ] `docker-compose.yml` created
- [ ] Stack running (`docker compose up -d`)
- [ ] Admin user created
- [ ] Admin console accessible at http://localhost:8080/admin
- [ ] OAuth client registered for your app
- [ ] App `.env` configured with `RAMPART_ISSUER`
