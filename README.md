<p align="center">
  <img src="https://raw.githubusercontent.com/manimovassagh/rampart/main/docs-site/static/img/logo.svg" alt="Rampart" width="120" />
</p>

<h1 align="center">Rampart</h1>

<p align="center">
  <strong>Open-source Identity & Access Management server</strong><br />
  OAuth 2.0 &bull; OpenID Connect &bull; SAML 2.0 &bull; SCIM 2.0
</p>

<p align="center">
  <a href="https://github.com/manimovassagh/rampart/actions/workflows/ci.yml"><img src="https://github.com/manimovassagh/rampart/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI" /></a>
  <a href="https://github.com/manimovassagh/rampart/actions/workflows/security.yml"><img src="https://github.com/manimovassagh/rampart/actions/workflows/security.yml/badge.svg?branch=main" alt="Security" /></a>
  <a href="https://github.com/manimovassagh/rampart/releases/latest"><img src="https://img.shields.io/github/v/release/manimovassagh/rampart?color=blue" alt="Release" /></a>
  <a href="https://github.com/manimovassagh/rampart/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-AGPL_v3-blue.svg" alt="License" /></a>
  <a href="https://manimovassagh.github.io/rampart/"><img src="https://img.shields.io/badge/docs-GitHub_Pages-blue?logo=github" alt="Documentation" /></a>
</p>

---

Rampart is a **single-binary** IAM server written in Go. Deploy it anywhere — Docker, bare metal, or Kubernetes — and get a complete identity platform out of the box.

## Highlights

| | Feature | Details |
|---|---|---|
| **Auth** | OAuth 2.0 / OIDC | Authorization Code + PKCE, Client Credentials, Device Flow, JWKS |
| **Enterprise** | SAML 2.0 & SCIM 2.0 | SP bridge for SSO, user & group provisioning |
| **MFA** | WebAuthn & TOTP | Passkeys, time-based OTP with backup codes |
| **Social** | Google, GitHub, Apple | One-click social login with automatic account linking |
| **RBAC** | Roles & Permissions | Fine-grained role-based access control with group support |
| **Compliance** | SOC 2, GDPR, HIPAA | Built-in compliance dashboards and audit trail export |
| **Ops** | Webhooks & Plugins | HMAC-signed event delivery, custom plugin extensions |
| **HA** | Clustering | PostgreSQL-based leader election for high availability |
| **Observability** | Metrics & Audit | Prometheus `/metrics` endpoint, full audit logging |
| **Admin** | Built-in Console | Server-side rendered with htmx + Tailwind CSS |

## Quick Start

```bash
# Docker Compose (recommended)
git clone https://github.com/manimovassagh/rampart.git
cd rampart
docker compose up -d --build
```

This starts PostgreSQL, Redis, and Rampart. The admin console is available at `http://localhost:8080/admin/`.

**From source:**

```bash
go build ./cmd/rampart
./rampart
```

## Configuration

| Variable | Description | Example |
|---|---|---|
| `RAMPART_DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@localhost:5432/rampart` |
| `RAMPART_ISSUER` | OIDC issuer URL | `http://localhost:8080` |
| `RAMPART_ENCRYPTION_KEY` | Key for encrypting secrets at rest | 32-byte hex string |
| `RAMPART_PORT` | HTTP listen port | `8080` |

See `docker-compose.yml` and `.env.example` for the full set of environment variables.

## Development

```bash
go test ./...          # Run all tests
golangci-lint run      # Lint
make check             # Full quality check (lint + vet + test + security)
```

## Tech Stack

**Go 1.26** &bull; **PostgreSQL** &bull; **Redis** (optional) &bull; **htmx + Tailwind CSS** &bull; **Docker**

## Documentation

Full documentation is available at **[manimovassagh.github.io/rampart](https://manimovassagh.github.io/rampart/)** — including API reference, architecture guides, deployment instructions, and tutorials.

[![Documentation Site](https://img.shields.io/badge/Read_the_Docs-%E2%86%92-blue?style=for-the-badge&logo=github)](https://manimovassagh.github.io/rampart/)

## CI/CD

Automated pipelines run on every push: **CI** &bull; **Tests** &bull; **Lint** &bull; **Security scanning** (gosec, govulncheck) &bull; **Docker build** &bull; **Release** &bull; **GitHub Pages**

## License

Licensed under the [GNU Affero General Public License v3.0](LICENSE).
