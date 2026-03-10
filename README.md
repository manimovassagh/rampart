<p align="center">
  <img src="https://raw.githubusercontent.com/manimovassagh/rampart/main/docs-site/static/img/logo.svg" alt="Rampart" width="80" />
</p>

<h1 align="center">Rampart</h1>

<p align="center">
  <strong>Open-source Identity & Access Management Server</strong><br />
  A single-binary IAM platform for modern applications
</p>

<p align="center">
  <a href="https://github.com/manimovassagh/rampart/actions/workflows/ci.yml"><img src="https://github.com/manimovassagh/rampart/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI" /></a>
  <a href="https://github.com/manimovassagh/rampart/actions/workflows/security.yml"><img src="https://github.com/manimovassagh/rampart/actions/workflows/security.yml/badge.svg?branch=main" alt="Security" /></a>
  <a href="https://github.com/manimovassagh/rampart/releases/latest"><img src="https://img.shields.io/github/v/release/manimovassagh/rampart?color=blue" alt="Release" /></a>
  <a href="https://github.com/manimovassagh/rampart/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-AGPL_v3-blue.svg" alt="License" /></a>
  <a href="https://manimovassagh.github.io/rampart/"><img src="https://img.shields.io/badge/docs-GitHub_Pages-blue?logo=github" alt="Documentation" /></a>
</p>

<br />

<p align="center">
  <img src="docs/images/hero-banner.svg" alt="Rampart — Open-source Identity & Access Management" width="100%" />
</p>

<br />

<p align="center">
  <a href="#why-rampart">Why Rampart</a> ·
  <a href="#features">Features</a> ·
  <a href="#admin-console">Admin Console</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="#quick-start">Quick Start</a> ·
  <a href="#how-it-works">How It Works</a> ·
  <a href="#configuration">Configuration</a> ·
  <a href="#documentation">Documentation</a> ·
  <a href="#contributing">Contributing</a>
</p>

---

## Why Rampart

Rampart ships as a **single Go binary** that gives you a complete identity platform out of the box. Lightweight, easy to deploy, and designed for both simplicity and production-grade security.

Deploy it on Docker, bare metal, or Kubernetes — and get OAuth 2.0, OpenID Connect, MFA, RBAC, and a built-in admin console in under a minute.

---

## Features

<table>
<tr>
<td width="50%" valign="top">

### Authentication
**OAuth 2.0 & OpenID Connect** — Authorization Code + PKCE, Client Credentials, Device Flow, and full JWKS rotation.

### Enterprise SSO
**SAML 2.0 & SCIM 2.0** — Service Provider bridge for single sign-on, automated user and group provisioning.

### Multi-Factor Auth
**WebAuthn & TOTP** — Passkeys, hardware keys, and time-based OTP with backup codes.

### Social Login
**Google, GitHub, Apple** — One-click social sign-in with automatic account linking and email verification.

</td>
<td width="50%" valign="top">

### Access Control
**Roles & Permissions** — Fine-grained RBAC with group support and scope-based authorization.

### Security Hardening
**Defense in depth** — Refresh token rotation, CSRF protection, per-endpoint rate limiting, configurable password policies, HSTS, encryption at rest, and OAuth consent flow protection.

### Observability
**Metrics & Audit** — Prometheus `/metrics` endpoint, structured audit logging, compliance dashboards (SOC 2, GDPR, HIPAA), and HMAC-signed webhook delivery.

### High Availability
**Clustering** — PostgreSQL-based leader election with graceful shutdown. No Redis, no external dependencies.

</td>
</tr>
</table>

---

## Admin Console

Rampart includes a **built-in admin dashboard** with real-time SSE updates — server-side rendered with Go templates. Manage users, applications, roles, sessions, and audit logs from a single interface.

<p align="center">
  <img src="docs/screenshots/dashboard.png" alt="Rampart Admin Console" width="100%" />
</p>

---

## Architecture

Rampart sits between your application and your database, handling all identity concerns so you don't have to.

<p align="center">
  <img src="docs/images/architecture.svg" alt="Rampart Architecture" width="100%" />
</p>

---

## Quick Start

### Docker Compose (recommended)

```bash
git clone https://github.com/manimovassagh/rampart.git
cd rampart
docker compose up -d --build
```

The admin console is available at **`http://localhost:8080/admin/`**

### From Source

```bash
go build ./cmd/rampart
./rampart
```

---

## How It Works

<p align="center">
  <img src="docs/images/demo-flow.svg" alt="Authentication Flow" width="100%" />
</p>

**1. Register a user**
```bash
curl -X POST http://localhost:8080/register \
  -H 'Content-Type: application/json' \
  -d '{"email": "user@example.com", "password": "S3cure!Pass"}'
```

**2. Login and get tokens**
```bash
curl -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"email": "user@example.com", "password": "S3cure!Pass"}'
```

**3. Call your API with the token**
```bash
curl http://localhost:8080/me \
  -H 'Authorization: Bearer <access_token>'
```

**4. Refresh when expired**
```bash
curl -X POST http://localhost:8080/token \
  -H 'Content-Type: application/json' \
  -d '{"grant_type": "refresh_token", "refresh_token": "<refresh_token>"}'
```

---

## Configuration

| Variable | Description | Default |
|---|---|---|
| `RAMPART_DB_URL` | PostgreSQL connection string (required) | — |
| `RAMPART_PORT` | HTTP listen port | `8080` |
| `RAMPART_ISSUER` | OIDC issuer URL | `http://localhost:8080` |
| `RAMPART_SIGNING_KEY_PATH` | RSA signing key (auto-generated if missing) | `rampart-signing-key.pem` |
| `RAMPART_ACCESS_TOKEN_TTL` | Access token lifetime in seconds | `900` (15 min) |
| `RAMPART_REFRESH_TOKEN_TTL` | Refresh token lifetime in seconds | `604800` (7 days) |
| `RAMPART_ENCRYPTION_KEY` | Hex-encoded 32-byte key for secrets at rest | — |
| `RAMPART_HSTS_ENABLED` | Enable HSTS header | `false` |
| `RAMPART_SECURE_COOKIES` | Set Secure flag on cookies (requires HTTPS) | `false` |
| `RAMPART_RATE_LIMIT_LOGIN` | Login attempts per minute per IP | `10` |
| `RAMPART_RATE_LIMIT_REGISTER` | Registrations per minute per IP | `5` |
| `RAMPART_RATE_LIMIT_TOKEN` | Token requests per minute per IP | `10` |
| `RAMPART_LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `RAMPART_LOG_FORMAT` | Log format (`pretty`, `text`, `json`) | `pretty` |
| `RAMPART_ALLOWED_ORIGINS` | Comma-separated CORS origins | — |
| `RAMPART_SMTP_HOST` | SMTP server for transactional emails | — |
| `RAMPART_SMTP_PORT` | SMTP port | `587` |
| `RAMPART_SMTP_FROM` | From address for emails | — |
| `RAMPART_GOOGLE_CLIENT_ID` | Google OAuth client ID | — |
| `RAMPART_GITHUB_CLIENT_ID` | GitHub OAuth client ID | — |
| `RAMPART_APPLE_CLIENT_ID` | Apple Sign-In client ID | — |

See `.env.example` for the full set including SMTP credentials and social login secrets.

---

## Tech Stack

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/htmx-1.9-3366CC?style=for-the-badge&logo=htmx&logoColor=white" alt="htmx" />
  <img src="https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white" alt="Tailwind CSS" />
</p>

---

## Documentation

Full documentation is available at **[manimovassagh.github.io/rampart](https://manimovassagh.github.io/rampart/)**

<table>
<tr>
<td align="center" width="25%">
  <a href="https://manimovassagh.github.io/rampart/"><strong>📖 Getting Started</strong></a><br />
  <sub>Installation, configuration, and first steps</sub>
</td>
<td align="center" width="25%">
  <a href="docs/api/overview.md"><strong>🔌 API Reference</strong></a><br />
  <sub>REST endpoints, OAuth flows, and OIDC</sub>
</td>
<td align="center" width="25%">
  <a href="docs/architecture/system-context.md"><strong>🏗️ Architecture</strong></a><br />
  <sub>System design, data model, and deployment</sub>
</td>
<td align="center" width="25%">
  <a href="docs/sdk/integration-guide.md"><strong>🧩 Integration</strong></a><br />
  <sub>SDK guide, samples, and best practices</sub>
</td>
</tr>
</table>

<p align="center">
  <a href="https://manimovassagh.github.io/rampart/">
    <img src="https://img.shields.io/badge/Read_the_Docs-%E2%86%92-blue?style=for-the-badge&logo=github" alt="Documentation" />
  </a>
</p>

---

## Sample Apps

The [`samples/`](samples/) directory contains working integration examples:

| Sample | Description | Port |
|--------|-------------|------|
| [express-backend](samples/express-backend/) | Express API with JWT verification via `@rampart/node` | 3001 |
| [web-frontend](samples/web-frontend/) | Browser app with login/logout UI via `@rampart/web` | 3000 |
| [react-app](samples/react-app/) | React SPA with routing, auth, and RBAC via `@rampart/react` | 3002 |

---

## Development

```bash
go test ./...          # Run all tests
golangci-lint run      # Lint
make check             # Full quality check (lint + vet + test + security)
```

## CI/CD

Automated pipelines run on every push: **CI** · **Tests** · **Lint** · **Security scanning** (gosec, govulncheck) · **Docker build** · **Release** · **GitHub Pages**

---

## Contributing

Contributions are welcome! Whether it's bug reports, feature requests, or pull requests — we'd love your help.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

---

<p align="center">
  <sub>Licensed under the <a href="LICENSE">GNU Affero General Public License v3.0</a></sub><br />
  <sub>Built with ❤️ by <a href="https://github.com/manimovassagh">Mani Movassagh</a></sub>
</p>
