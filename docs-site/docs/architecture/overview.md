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
                          |  - Tokens             |
                          +-----------------------+
```

## Core Components

| Component | Responsibility |
|-----------|---------------|
| **HTTP Server** | chi-based router serving all endpoints, static assets, and middleware |
| **OAuth 2.0 / OIDC Engine** | Authorization code + PKCE, client credentials, token refresh, device flow, JWKS |
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
│   └── rampart/
│       └── main.go              # Entry point, wires dependencies
├── internal/
│   ├── auth/                    # OAuth 2.0/OIDC engine
│   │   ├── authorize.go         # Authorization endpoint
│   │   ├── token.go             # Token endpoint
│   │   ├── jwks.go              # JWKS endpoint and key management
│   │   └── pkce.go              # PKCE verification
│   ├── admin/                   # Admin API handlers
│   │   ├── users.go
│   │   ├── organizations.go
│   │   ├── roles.go
│   │   └── clients.go
│   ├── account/                 # Self-service account API
│   ├── session/                 # Session management (PostgreSQL-backed)
│   ├── audit/                   # Audit event logging
│   ├── middleware/              # Auth, CORS, rate limit, security headers
│   ├── model/                   # Domain types (User, Org, Role, etc.)
│   ├── store/                   # PostgreSQL data access (pgx, no ORM)
│   ├── config/                  # YAML config loading and validation
│   └── crypto/                  # Password hashing, token signing utilities
├── internal/templates/           # Go templates for admin dashboard (htmx + Tailwind)
├── internal/templates/login/    # Go templates for login/consent UI (Tailwind)
├── migrations/                  # SQL migration files
├── configs/                     # Example YAML configuration
├── docs/                        # API specs, architecture docs
└── docs-site/                   # Docusaurus documentation site
```

### Key Conventions

- **`internal/`** prevents external packages from importing Rampart internals, keeping the public API surface minimal.
- **No ORM.** All SQL is written explicitly using `pgx`. Queries are predictable, debuggable, and performant.
- **No dependency injection framework.** Dependencies are wired manually in `main.go` using constructor functions.
- **One concern per package.** The `auth` package handles OAuth/OIDC flows; it does not manage users or sessions directly.

## Design Principles

### Single Binary Deployment

Rampart compiles to a single static binary with the admin and login UIs (built with htmx, Go templates, and Tailwind CSS) embedded via Go's `embed` package. No external file dependencies, no runtime installations, no containers required (though Docker is supported).

```bash
# Deploy Rampart
./rampart serve --config rampart.yaml
```

### Minimal Dependencies

Every dependency is a supply chain risk, especially for an IAM product. Rampart uses the Go standard library wherever possible and limits third-party packages to well-audited, actively maintained libraries:

- **chi** — HTTP router (lightweight, stdlib-compatible)
- **pgx** — PostgreSQL driver (pure Go, high performance)
- **golang-jwt** — JWT signing and verification

### API-First

Every feature is accessible via REST API before any UI is built. The admin dashboard and login UI are consumers of the same APIs available to external integrations. This ensures feature parity between UI and API access.

### Multi-Tenant by Default

Organizations (tenants) are a first-class concept. Users, roles, clients, and policies are scoped to organizations. A single Rampart instance can serve multiple independent tenants with full data isolation.

### Security as the Product

Rampart is not a web framework with auth bolted on. Identity and access management is the entire product. Every design decision prioritizes security: defense in depth, principle of least privilege, secure defaults, and full audit logging.

### Standards Compliance

Rampart implements OAuth 2.0 (RFC 6749), OIDC Core 1.0, PKCE (RFC 7636), and related specifications precisely. Compatibility with standard client libraries and relying parties is a hard requirement.
