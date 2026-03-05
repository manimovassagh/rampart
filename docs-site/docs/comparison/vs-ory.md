---
sidebar_position: 2
title: Rampart vs Ory
description: Comparison between Rampart and the Ory ecosystem (Hydra, Kratos, Keto, Oathkeeper) — architecture, deployment, and developer experience.
---

# Rampart vs Ory

Ory provides a suite of open-source identity and access management services: Hydra (OAuth 2.0/OIDC), Kratos (identity management), Keto (authorization), and Oathkeeper (access proxy). Each is a standalone microservice designed to be composed together. Rampart takes the opposite approach: a single, batteries-included binary.

## Comparison Table

| Aspect | Rampart | Ory (Hydra + Kratos + Keto) |
|--------|---------|----------------------------|
| **Architecture** | Single binary, all-in-one | Microservices (3–4 separate services) |
| **Language** | Go | Go |
| **Admin UI** | Included (React SPA) | Not included (build your own or use Ory Cloud) |
| **Login UI** | Included (React SPA, themeable) | Not included (you must build it) |
| **Deployment** | 1 binary + PostgreSQL + Redis | 3–4 binaries + PostgreSQL + migrations per service |
| **Configuration** | Single YAML file | Separate config per service |
| **User management** | Built-in admin API + UI | Kratos handles identity, no admin UI |
| **OAuth 2.0 / OIDC** | Built-in | Hydra (separate service) |
| **Authorization** | Built-in RBAC | Keto (separate service, Zanzibar-based) |
| **Database** | PostgreSQL | PostgreSQL, MySQL, CockroachDB |
| **Self-hosted** | Full feature set, free | Full feature set, free |
| **Cloud offering** | Planned | Ory Network (managed cloud) |
| **License** | AGPL-3.0 | Apache 2.0 |

## Architecture Approach

### Ory: Headless Microservices

Ory's philosophy is to provide identity infrastructure as composable building blocks. Each service handles one concern:

- **Hydra** — OAuth 2.0 and OIDC provider. Does not manage users; delegates authentication to your application.
- **Kratos** — Identity and user management. Handles registration, login, password recovery, MFA. Does not provide a UI.
- **Keto** — Fine-grained authorization based on Google's Zanzibar paper. Relationship-based access control.
- **Oathkeeper** — Reverse proxy that enforces authentication and authorization at the network layer.

A production Ory deployment requires running and configuring multiple services, building your own login/registration UI, and wiring everything together.

### Rampart: Batteries Included

Rampart bundles OAuth 2.0/OIDC, user management, RBAC, session management, audit logging, an admin dashboard, and a login UI into a single binary. Deploy one service, configure one YAML file, and everything works together.

```
Ory production deployment:

  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐
  │  Hydra   │  │  Kratos  │  │   Keto   │  │ Oathkeeper │
  │ (OAuth)  │  │ (Users)  │  │  (Authz) │  │  (Proxy)   │
  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬──────┘
       │              │             │               │
       ├──────────────┼─────────────┼───────────────┘
       │              │             │
  ┌────┴─────┐  ┌─────┴────┐  ┌────┴─────┐
  │ Hydra DB │  │Kratos DB │  │ Keto DB  │
  └──────────┘  └──────────┘  └──────────┘
  + Your custom login UI
  + Your custom admin UI
  + Your custom consent UI


Rampart production deployment:

  ┌─────────────────────────────┐
  │         Rampart             │
  │  (single binary, all-in-one)│
  └──────────────┬──────────────┘
                 │
       ┌─────────┴─────────┐
       │                   │
  ┌────┴─────┐      ┌─────┴────┐
  │PostgreSQL│      │  Redis   │
  └──────────┘      └──────────┘
  Login UI and admin UI included.
```

## Developer Experience

### Building a Login Page

**With Ory Kratos:**

Kratos provides identity APIs but no UI. You must:

1. Build a login page in your frontend framework.
2. Integrate with Kratos's self-service flow APIs (initialize flow, fetch UI nodes, submit form).
3. Handle error states, CSRF tokens, and redirects yourself.
4. Build registration, password recovery, settings, and verification pages the same way.

Ory provides reference implementations, but they are starting points, not production-ready UIs.

**With Rampart:**

Rampart includes a production-ready login UI with built-in themes. Configure your OAuth client, and users are redirected to the Rampart login page. Customize the appearance by selecting a theme or overriding CSS variables. No frontend development required for standard flows.

For custom login experiences, Rampart's APIs are available for headless integration, providing the same flexibility as Ory when needed.

### Operational Overhead

| Task | Rampart | Ory |
|------|---------|-----|
| Initial deployment | 1 binary, 1 config file | 3–4 services, 3–4 config files, 3–4 databases |
| Database migrations | Single migration set | Separate migrations per service |
| Version upgrades | Upgrade 1 binary | Coordinate upgrades across 3–4 services |
| Monitoring | 1 service to monitor | 3–4 services, inter-service health checks |
| Debugging | Single log stream | Correlate logs across services |
| Scaling | Scale 1 service | Scale each service independently |

## Where Ory Wins

Ory has genuine advantages for certain use cases:

- **Flexibility.** The headless approach gives complete control over the user experience. If you need a deeply customized login flow with non-standard UI, Ory's approach may suit better.
- **Zanzibar-based authorization.** Keto implements Google's Zanzibar model for fine-grained, relationship-based access control. This is more expressive than Rampart's RBAC model for complex authorization scenarios (e.g., "user X can edit document Y because they are a member of team Z which owns folder W").
- **Independent scaling.** In very large deployments, the ability to scale OAuth, identity, and authorization independently can be valuable.
- **Apache 2.0 license.** More permissive than Rampart's AGPL-3.0 for embedding in proprietary products.
- **Maturity.** Hydra is OIDC-certified and has been in production use since 2015.
- **Ory Network.** Managed cloud offering available for teams that want Ory's architecture without operational overhead.

## When to Choose Rampart

Rampart is the better fit when:

- You want a working IAM system quickly without building custom UIs.
- You prefer a single service to deploy, monitor, and upgrade.
- Your authorization model fits RBAC (users, roles, permissions).
- You want an admin dashboard included out of the box.
- Your team does not want to integrate and maintain multiple microservices for identity.
- You want login page theming without frontend development.

## When to Choose Ory

Ory is the better fit when:

- You need complete control over every UI interaction in the authentication flow.
- Your authorization model requires Zanzibar-style relationship-based access control.
- You are already running a microservices architecture and prefer identity as a composable service.
- You need OIDC certification today.
- Apache 2.0 licensing is a hard requirement.
