---
sidebar_position: 1
title: Quickstart
description: Get Rampart running in under five minutes with Docker Compose, create your first user, and access the admin console.
---

# Quickstart

This guide walks you through getting Rampart running locally with Docker Compose, creating your first user, and accessing the admin console. The entire process takes under five minutes.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (version 20.10 or later)
- [Docker Compose](https://docs.docker.com/compose/install/) (version 2.0 or later)
- [curl](https://curl.se/) (for API calls)
- [Git](https://git-scm.com/) (to clone the repository)

## Step 1: Clone the Repository

```bash
git clone https://github.com/manimovassagh/rampart.git
cd rampart
```

## Step 2: Start Rampart with Docker Compose

```bash
docker compose up -d
```

This starts two containers:

- **rampart** — the IAM server on port `8080`
- **postgres** — PostgreSQL database on port `5432`

Wait a few seconds for all services to be healthy:

```bash
docker compose ps
```

You should see both services in a `running` (healthy) state.

## Step 3: Verify Rampart Is Running

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{
  "status": "ok"
}
```

## Step 4: Create Your First User

Register a new user via the API:

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "SecureP@ssw0rd!",
    "given_name": "Admin",
    "family_name": "User"
  }'
```

Expected response:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "admin",
  "email": "admin@example.com",
  "given_name": "Admin",
  "family_name": "User",
  "created_at": "2026-03-05T10:00:00Z"
}
```

## Step 5: Log In and Get a Token

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "admin@example.com",
    "password": "SecureP@ssw0rd!"
  }'
```

The `identifier` field accepts either an email address or a username.

Expected response:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

Save the `access_token` for subsequent requests:

```bash
export TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Step 6: Access the Admin Console

Open your browser and navigate to:

```
http://localhost:8080/admin
```

Log in with the credentials you created in Step 4. The admin console provides a web UI for managing users, organizations, roles, sessions, clients, and audit events.

## Step 7: Use the CLI (Optional)

If you have built the CLI tool (`go build ./cmd/rampart-cli`), you can authenticate directly from the terminal:

```bash
rampart-cli login --issuer http://localhost:8080 --email admin@example.com --password 'SecureP@ssw0rd!'
```

Check your identity:

```bash
rampart-cli whoami
```

See the [CLI documentation](./cli.md) for the full command reference.

## Step 8: Explore the API

Fetch your own profile using the token from Step 5:

```bash
curl http://localhost:8080/me \
  -H "Authorization: Bearer $TOKEN"
```

List all users (requires admin role):

```bash
curl http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer $TOKEN"
```

## Stopping Rampart

```bash
docker compose down
```

To also remove the database volume (this deletes all data):

```bash
docker compose down -v
```

## Next Steps

- [Docker deployment guide](./docker.md) — production Docker configuration
- [Configuration reference](./configuration.md) — environment variables and YAML config
- [Social login](./configuration#social-login-providers) — configure Google, GitHub, and Apple sign-in
- [CLI tool](./cli.md) — full CLI command reference
- [API overview](../api/overview.md) — REST API documentation
