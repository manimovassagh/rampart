# Rampart

**A Go-based Identity & Access Management server with OAuth 2.0, OpenID Connect, SAML 2.0, and SCIM 2.0 support.**

[![CI](https://github.com/manimovassagh/rampart/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/manimovassagh/rampart/actions)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://github.com/manimovassagh/rampart/blob/main/LICENSE)

Rampart is a single-binary IAM server written in Go. It provides a complete identity platform with a built-in admin console, social login, multi-factor authentication, webhook delivery, a plugin system, and high-availability clustering.

**Current version: v2.0.0**

---

## Features

**Authentication & Authorization**
- OAuth 2.0 with Authorization Code + PKCE, Client Credentials, and Device Flow
- OpenID Connect discovery, ID tokens, and JWKS endpoint
- SAML 2.0 Service Provider bridge for enterprise SSO
- SCIM 2.0 user and group provisioning

**Multi-Factor Authentication**
- WebAuthn / Passkey support
- TOTP (time-based one-time passwords) with backup codes

**User & Identity Management**
- Email verification and forgot-password flows
- Social login (Google, GitHub, Apple)
- Role-based access control (RBAC) with fine-grained permissions

**Operations & Extensibility**
- Webhook event delivery with HMAC signing
- Plugin system for custom extensions
- HA clustering with PostgreSQL leader election
- Compliance dashboards (SOC 2, GDPR, HIPAA)
- Prometheus metrics endpoint (`/metrics`)
- Admin console (server-side rendered with htmx + Go templates)

---

## Quick Start

**Docker Compose (recommended):**

```bash
git clone https://github.com/manimovassagh/rampart.git
cd rampart
docker compose up -d --build
```

This starts PostgreSQL, Redis, and the Rampart server.

**From source:**

```bash
git clone https://github.com/manimovassagh/rampart.git
cd rampart
go build ./cmd/rampart
./rampart
```

Or use the Makefile:

```bash
make build
./bin/rampart
```

---

## Configuration

Rampart is configured via environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `RAMPART_DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@localhost:5432/rampart` |
| `RAMPART_ISSUER` | OIDC issuer URL | `http://localhost:8080` |
| `RAMPART_ENCRYPTION_KEY` | Key for encrypting secrets at rest | 32-byte hex string |
| `RAMPART_PORT` | HTTP listen port | `8080` |

See `docker-compose.yml` and `.env.example` for the full set of configurable variables.

---

## Project Structure

```
cmd/
  rampart/          Main server entry point
  rampart-cli/      CLI tool
internal/
  handler/          HTTP handlers and admin UI (SSR)
  middleware/       HTTP middleware (auth, logging, etc.)
  model/            Domain models
  token/            JWT token issuance and validation
  oauth/            OAuth 2.0 and OIDC flows
  session/          Session management
  store/            Database access layer (interface-based)
  plugin/           Plugin system
  cluster/          HA clustering and leader election
  webhook/          Webhook event delivery
  audit/            Audit logging
  social/           Social login providers
  crypto/           Encryption utilities
  signing/          Signing key management
  mfa/              Multi-factor authentication (TOTP, WebAuthn)
  email/            Email sending (verification, password reset)
  metrics/          Prometheus metrics
  config/           Configuration loading
  database/         Database connection and migrations
  auth/             Authentication logic
  logging/          Structured logging
  server/           HTTP server setup
  apierror/         API error types
  cli/              CLI command definitions
migrations/         SQL migration files
examples/           React + Node.js demo applications
.github/workflows/  CI/CD pipelines
```

---

## Build, Test, Lint

```bash
# Build
go build ./cmd/rampart

# Run all tests
go test ./...

# Lint
golangci-lint run

# Full quality check (lint + vet + test + security)
make check
```

---

## CI/CD

GitHub Actions workflows in `.github/workflows/`:

- `ci.yml` -- Full CI pipeline
- `test.yml` -- Unit and integration tests
- `lint.yml` -- golangci-lint
- `security.yml` -- Security scanning (gosec, govulncheck)
- `docker.yml` -- Docker image build and push
- `release.yml` -- Tagged release builds
- `pages.yml` -- GitHub Pages deployment

---

## Tech Stack

- **Language:** Go 1.26
- **Database:** PostgreSQL (required)
- **Cache/Sessions:** Redis (optional)
- **Admin UI:** htmx + Tailwind CSS + Go templates
- **Container:** Docker / Docker Compose

---

## License

Licensed under the [GNU Affero General Public License v3.0](LICENSE).
