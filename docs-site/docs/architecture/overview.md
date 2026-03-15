---
sidebar_position: 1
title: Architecture Overview
description: High-level architecture of the Rampart IAM server, including component design, request flow, package structure, and design principles.
---

# Architecture Overview

Rampart is a single-binary identity and access management (IAM) server built in Go. It bundles a PostgreSQL-backed data layer and embedded admin and login UIs into one deployable artifact. No Redis or external cache is required.

## High-Level Architecture

```
                          +-----------------------+
                          |     Load Balancer     |
                          |   (nginx / ALB / etc) |
                          +-----------+-----------+
                                      |
                                      v
                          +-----------------------+
                          |    Rampart Server      |
                          |    (single Go binary)  |
                          |                       |
                          |  +------ HTTP ------+ |
                          |  |                   | |
                          |  |  OAuth 2.0/OIDC   | |
                          |  |  Endpoints        | |
                          |  |  (/oauth, /.well-| |
                          |  |   known/openid-   | |
                          |  |   configuration)  | |
                          |  |                   | |
                          |  |  Admin REST API   | |
                          |  |  (/api/v1/admin)  | |
                          |  |                   | |
                          |  |  Account API      | |
                          |  |  (/api/v1/account)| |
                          |  |                   | |
                          |  |  Embedded UIs     | |
                          |  |  (Admin + Login)  | |
                          |  +-------------------+ |
                          +-----------+-----------+
                                      |
                                      |
                                      v
                          +-----------------------+
                          |      PostgreSQL       |
                          |                       |
                          |  - Users              |
                          |  - Organizations      |
                          |  - Roles              |
                          |  - OAuth Clients      |
                          |  - Sessions           |
                          |  - Audit Events       |
                          +-----------------------+
```

## Core Components

| Component | Responsibility |
|-----------|---------------|
| **HTTP Server** | chi-based router serving all endpoints, static assets, and middleware |
| **OAuth 2.0 / OIDC Engine** | Authorization code + PKCE, client credentials, token refresh, JWKS |
| **Admin API** | CRUD for users, organizations, roles, clients; RBAC-protected |
| **Account API** | Self-service profile, password change, MFA enrollment |
| **Session Manager** | PostgreSQL-backed session creation, validation, revocation |
| **Audit Logger** | Append-only event log for security-relevant actions |
| **Embedded Admin UI** | htmx + Go templates + Tailwind served from `/admin` |
| **Embedded Login UI** | Go templates + Tailwind served from `/login`, themeable per-tenant |

## Request Flow

A typical OAuth 2.0 authorization code flow through Rampart:

```
Client App                Rampart Server              PostgreSQL
    |                           |                          |                |
    |-- GET /oauth/authorize ->|                          |                |
    |                           |-- validate client ------>|                |
    |                           |<-- client record --------|                |
    |                           |                          |                |
    |<-- 302 to /login ---------|                          |                |
    |                           |                          |                |
    |-- POST /login ----------->|                          |                |
    |                           |-- verify credentials --->|                |
    |                           |<-- user record ----------|                |
    |                           |-- create session ------->|--------------->|
    |                           |-- store auth code ------>|                |
    |<-- 302 to callback -------|                          |                |
    |                           |                          |                |
    |-- POST /oauth/token ---->|                          |                |
    |                           |-- exchange auth code --->|                |
    |                           |-- issue JWT (signed) ----|                |
    |<-- access_token, id_token,|                          |                |
    |    refresh_token ---------|                          |                |
```

## Package Structure

Rampart follows standard Go project layout conventions:

```
rampart/
├── cmd/
│   ├── rampart/
│   │   └── main.go              # Server entry point, wires dependencies
│   └── rampart-cli/
│       └── main.go              # CLI tool entry point
├── internal/
│   ├── apierror/                # Structured API error types
│   ├── audit/                   # Audit event logging
│   ├── auth/                    # Password hashing (argon2id), credential utilities
│   ├── cli/                     # CLI command definitions
│   ├── cluster/                 # Leader election for HA background workers
│   ├── config/                  # Environment-variable-based configuration loading
│   ├── crypto/                  # Encryption at rest for secrets
│   ├── database/                # PostgreSQL connection, migrations, data access (pgx)
│   ├── email/                   # SMTP transactional email sender
│   ├── handler/                 # HTTP handlers (OAuth, admin, login, MFA, SAML, SCIM, etc.)
│   ├── logging/                 # Structured logging helpers (pretty, JSON, text)
│   ├── metrics/                 # Prometheus metrics endpoint
│   ├── mfa/                     # Multi-factor authentication (TOTP, WebAuthn)
│   ├── middleware/              # Auth, CORS, rate limiting, security headers
│   ├── model/                   # Domain types (User, Org, Role, etc.)
│   ├── oauth/                   # OAuth 2.0 / OIDC flow logic
│   ├── plugin/                  # Plugin registry for event hooks and claim enrichers
│   ├── server/                  # HTTP server setup and route registration
│   ├── session/                 # Session management (PostgreSQL-backed)
│   ├── signing/                 # RSA key pair loading, generation, and rotation
│   ├── social/                  # Social login providers (Google, GitHub, Apple)
│   ├── store/                   # Additional data access helpers
│   ├── token/                   # JWT token issuance and validation
│   └── webhook/                 # Webhook dispatch and delivery retry
├── migrations/                  # SQL migration files
├── adapters/                    # Official SDK adapters (Node, Go, Python, etc.)
├── cookbook/                     # Complete sample applications
├── docs/                        # API specs, architecture docs
└── docs-site/                   # Docusaurus documentation site
```

### Key Conventions

- **`internal/`** prevents external packages from importing Rampart internals, keeping the public API surface minimal.
- **No ORM.** All SQL is written explicitly using `pgx`. Queries are predictable, debuggable, and performant.
- **No dependency injection framework.** Dependencies are wired manually in `main.go` using constructor functions.

## Design Principles

### Single Binary Deployment

Rampart compiles to a single static binary with the admin and login UIs (built with htmx, Go templates, and Tailwind CSS) embedded via Go's `embed` package. No external file dependencies, no runtime installations, no containers required (though Docker is supported).

```bash
# Deploy Rampart (configured entirely via environment variables)
export RAMPART_DB_URL="postgres://user:pass@host:5432/rampart"
export RAMPART_ISSUER="https://auth.example.com"
./rampart
```

### Minimal Dependencies

Every dependency is a supply chain risk, especially for an IAM product. Rampart uses the Go standard library wherever possible and limits third-party packages to well-audited, actively maintained libraries:

- **chi** -- HTTP router (lightweight, stdlib-compatible)
- **pgx** -- PostgreSQL driver (pure Go, high performance)
- **golang-jwt** -- JWT signing and verification

### API-First

Every feature is accessible via REST API before any UI is built. The admin dashboard and login UI are consumers of the same APIs available to external integrations. This ensures feature parity between UI and API access.

### Multi-Tenant by Default

Organizations (tenants) are a first-class concept. Users, roles, clients, and policies are scoped to organizations. A single Rampart instance can serve multiple independent tenants with full data isolation.

### Security as the Product

Rampart is not a web framework with auth bolted on. Identity and access management is the entire product. Every design decision prioritizes security: defense in depth, principle of least privilege, secure defaults, and full audit logging.

### Standards Compliance

Rampart implements OAuth 2.0 (RFC 6749), OIDC Core 1.0, PKCE (RFC 7636), and related specifications precisely. Compatibility with standard client libraries and relying parties is a hard requirement.
