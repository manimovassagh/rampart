---
sidebar_position: 3
title: Configuration
description: Complete configuration reference for Rampart — environment variables, database settings, signing keys, CORS, rate limiting, SMTP, and social login.
---

# Configuration

Rampart is configured entirely through environment variables. All variables are prefixed with `RAMPART_` (except social login providers which use `RAMPART_GOOGLE_*`, `RAMPART_GITHUB_*`, and `RAMPART_APPLE_*`).

## Environment Variables Reference

### Server

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_PORT` | HTTP listen port | `8080` |
| `RAMPART_ISSUER` | Public base URL for OIDC Discovery and JWT `iss` claim | `http://localhost:8080` |

### Database

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_DB_URL` | PostgreSQL connection string | (required) |

The database connection string follows the standard PostgreSQL URI format:

```
postgres://user:password@host:port/database?sslmode=require
```

Supported `sslmode` values: `disable`, `require`, `verify-ca`, `verify-full`. Use `verify-full` in production.

### Signing Keys

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_SIGNING_KEY_PATH` | Path to RSA private key (PEM format) | `rampart-signing-key.pem` |

If the signing key file does not exist at the configured path, Rampart generates a 2048-bit RSA key pair on first startup and persists it there. In production, you should generate and manage your own keys:

```bash
openssl genrsa -out rampart-signing-key.pem 2048
openssl rsa -in rampart-signing-key.pem -pubout -out rampart-signing-key-pub.pem
```

### Tokens

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_ACCESS_TOKEN_TTL` | Access token lifetime in seconds | `900` (15 minutes) |
| `RAMPART_REFRESH_TOKEN_TTL` | Refresh token lifetime in seconds | `604800` (7 days) |

Both values are integers representing seconds.

### CORS

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_ALLOWED_ORIGINS` | Comma-separated allowed origins | (none) |

### Security

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_HSTS_ENABLED` | Enable HTTP Strict Transport Security header | Auto-enabled when `RAMPART_ISSUER` starts with `https://` |
| `RAMPART_SECURE_COOKIES` | Set the `Secure` flag on all cookies (requires HTTPS) | `false` |
| `RAMPART_ENCRYPTION_KEY` | Hex-encoded 32-byte key for encrypting secrets at rest | (none — secrets stored in plaintext) |
| `RAMPART_METRICS_TOKEN` | Bearer token required to access `/metrics` endpoint | (none — endpoint disabled) |
| `RAMPART_TRUSTED_PROXIES` | Comma-separated CIDR ranges or IPs whose `X-Forwarded-For` / `X-Real-IP` headers are trusted for rate limiting | (none — proxy headers ignored) |

### Rate Limiting

Per-IP rate limits in requests per minute for authentication endpoints.

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_RATE_LIMIT_LOGIN` | Login endpoint rate limit (req/min per IP) | `10` |
| `RAMPART_RATE_LIMIT_REGISTER` | Registration endpoint rate limit (req/min per IP) | `5` |
| `RAMPART_RATE_LIMIT_TOKEN` | Token endpoint rate limit (req/min per IP) | `10` |

### Logging

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `RAMPART_LOG_FORMAT` | Output format: `pretty`, `text`, `json` | `pretty` |

### SMTP (Email)

Required for transactional emails (password reset, email verification, etc.). If `RAMPART_SMTP_HOST` is not set, email features are disabled.

| Variable | Description | Default |
|----------|-------------|---------|
| `RAMPART_SMTP_HOST` | SMTP server hostname | (none) |
| `RAMPART_SMTP_PORT` | SMTP server port | `587` |
| `RAMPART_SMTP_USERNAME` | SMTP authentication username | (none) |
| `RAMPART_SMTP_PASSWORD` | SMTP authentication password | (none) |
| `RAMPART_SMTP_FROM` | Sender address for outgoing emails | (none) |

### Social Login Providers

Configure OAuth credentials to enable social login. Each provider is optional — only configure the ones you need.

#### Google

| Variable | Description |
|----------|-------------|
| `RAMPART_GOOGLE_CLIENT_ID` | Google OAuth 2.0 client ID |
| `RAMPART_GOOGLE_CLIENT_SECRET` | Google OAuth 2.0 client secret |

#### GitHub

| Variable | Description |
|----------|-------------|
| `RAMPART_GITHUB_CLIENT_ID` | GitHub OAuth App client ID |
| `RAMPART_GITHUB_CLIENT_SECRET` | GitHub OAuth App client secret |

#### Apple

| Variable | Description |
|----------|-------------|
| `RAMPART_APPLE_CLIENT_ID` | Apple Services ID |
| `RAMPART_APPLE_TEAM_ID` | Apple Developer Team ID |
| `RAMPART_APPLE_KEY_ID` | Apple private key ID |
| `RAMPART_APPLE_PRIVATE_KEY` | Apple private key contents (PEM format) |
