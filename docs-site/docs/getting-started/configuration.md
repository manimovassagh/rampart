---
sidebar_position: 3
title: Configuration
description: Complete configuration reference for Rampart — environment variables, YAML config, database settings, signing keys, CORS, and logging.
---

# Configuration

Rampart supports configuration through environment variables and YAML configuration files. Environment variables take precedence over YAML values, allowing you to use YAML for defaults and override specific settings per environment.

## Configuration File

By default, Rampart looks for a configuration file at `./rampart.yaml`. You can specify a different path with the `RAMPART_CONFIG` environment variable:

```bash
RAMPART_CONFIG=/etc/rampart/config.yaml ./rampart
```

### Example YAML Configuration

```yaml
server:
  port: 8080
  issuer: https://auth.example.com

database:
  url: postgres://rampart:secret@localhost:5432/rampart?sslmode=require
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

signing:
  key_path: /data/keys/signing.pem
  algorithm: RS256

tokens:
  access_token_ttl: 1h
  refresh_token_ttl: 168h  # 7 days
  id_token_ttl: 1h

sessions:
  ttl: 24h

cors:
  allowed_origins:
    - https://app.example.com
    - https://admin.example.com
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  allowed_headers:
    - Authorization
    - Content-Type
  max_age: 3600

logging:
  level: info
  format: json
```

## Environment Variables Reference

All environment variables are prefixed with `RAMPART_`.

### Server

| Variable | YAML Path | Description | Default |
|----------|-----------|-------------|---------|
| `RAMPART_PORT` | `server.port` | HTTP listen port | `8080` |
| `RAMPART_ISSUER` | `server.issuer` | Public base URL for OIDC Discovery and JWT `iss` | `http://localhost:8080` |
| `RAMPART_CONFIG` | — | Path to YAML configuration file | `./rampart.yaml` |

### Database

| Variable | YAML Path | Description | Default |
|----------|-----------|-------------|---------|
| `RAMPART_DB_URL` | `database.url` | PostgreSQL connection string | (required) |
| `RAMPART_DB_MAX_OPEN_CONNS` | `database.max_open_conns` | Maximum open database connections | `25` |
| `RAMPART_DB_MAX_IDLE_CONNS` | `database.max_idle_conns` | Maximum idle database connections | `5` |
| `RAMPART_DB_CONN_MAX_LIFETIME` | `database.conn_max_lifetime` | Maximum connection lifetime | `5m` |

The database connection string follows the standard PostgreSQL URI format:

```
postgres://user:password@host:port/database?sslmode=require
```

Supported `sslmode` values: `disable`, `require`, `verify-ca`, `verify-full`. Use `verify-full` in production.

### Signing Keys

| Variable | YAML Path | Description | Default |
|----------|-----------|-------------|---------|
| `RAMPART_SIGNING_KEY_PATH` | `signing.key_path` | Path to RSA private key (PEM format) | Auto-generated |
| `RAMPART_SIGNING_ALGORITHM` | `signing.algorithm` | JWT signing algorithm | `RS256` |

If no signing key is provided, Rampart generates a 2048-bit RSA key pair on first startup and persists it to the configured path. In production, you should generate and manage your own keys:

```bash
openssl genrsa -out signing.pem 2048
openssl rsa -in signing.pem -pubout -out signing-pub.pem
```

### Tokens

| Variable | YAML Path | Description | Default |
|----------|-----------|-------------|---------|
| `RAMPART_ACCESS_TOKEN_TTL` | `tokens.access_token_ttl` | Access token lifetime | `1h` |
| `RAMPART_REFRESH_TOKEN_TTL` | `tokens.refresh_token_ttl` | Refresh token lifetime | `168h` (7 days) |
| `RAMPART_ID_TOKEN_TTL` | `tokens.id_token_ttl` | OIDC ID token lifetime | `1h` |

### Sessions

| Variable | YAML Path | Description | Default |
|----------|-----------|-------------|---------|
| `RAMPART_SESSION_TTL` | `sessions.ttl` | Session time-to-live | `24h` |

### CORS

| Variable | YAML Path | Description | Default |
|----------|-----------|-------------|---------|
| `RAMPART_ALLOWED_ORIGINS` | `cors.allowed_origins` | Comma-separated allowed origins | `*` |
| `RAMPART_CORS_METHODS` | `cors.allowed_methods` | Comma-separated allowed HTTP methods | `GET,POST,PUT,DELETE,OPTIONS` |
| `RAMPART_CORS_HEADERS` | `cors.allowed_headers` | Comma-separated allowed headers | `Authorization,Content-Type` |
| `RAMPART_CORS_MAX_AGE` | `cors.max_age` | Preflight cache duration in seconds | `3600` |

### Logging

| Variable | YAML Path | Description | Default |
|----------|-----------|-------------|---------|
| `RAMPART_LOG_LEVEL` | `logging.level` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `RAMPART_LOG_FORMAT` | `logging.format` | Output format: `json`, `text` | `json` |

## Precedence Order

Configuration values are resolved in the following order (highest priority first):

1. **Environment variables** — always win
2. **YAML configuration file** — used as defaults
3. **Built-in defaults** — used when neither is set

## Validating Configuration

Start Rampart with the `--validate` flag to check your configuration without starting the server:

```bash
./rampart --validate
```

This verifies database connectivity, signing key validity, and configuration syntax.
