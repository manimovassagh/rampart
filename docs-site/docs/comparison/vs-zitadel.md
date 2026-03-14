---
sidebar_position: 3
title: Rampart vs Zitadel
description: Comparison between Rampart and Zitadel — database requirements, extensibility, resource usage, and architecture trade-offs.
---

# Rampart vs Zitadel

Zitadel is a Go-based identity management platform that provides OAuth 2.0/OIDC, user management, and a built-in admin console. Like Rampart, it compiles to a single binary. The key differences lie in database requirements, extensibility philosophy, and architectural decisions.

## Comparison Table

| Aspect | Rampart | Zitadel |
|--------|---------|---------|
| **Language** | Go | Go |
| **Database** | PostgreSQL | CockroachDB (primary) or PostgreSQL (added later) |
| **Cache/Sessions** | PostgreSQL | In-process (event-sourced projections) |
| **Architecture** | Traditional CRUD + event audit log | Full event sourcing (CQRS) |
| **Admin UI** | htmx + Go templates + Tailwind | Angular (custom framework) |
| **Login UI** | htmx + Go templates, CSS variable themes | Server-rendered Go templates |
| **Deployment** | Single binary + PostgreSQL | Single binary + CockroachDB (or PostgreSQL) |
| **Extensibility** | Plugin system (planned), webhooks | Actions (JavaScript/TypeScript runtime) |
| **Multi-tenancy** | Organizations | Organizations + instances |
| **Resource usage** | ~30 MB idle | ~100–200 MB idle |
| **License** | AGPL-3.0 | Apache 2.0 |
| **OIDC certified** | Planned | Yes |

## Database Requirements

### Zitadel and CockroachDB

Zitadel was originally built exclusively for CockroachDB, a distributed SQL database. While PostgreSQL support was added later, CockroachDB remains the recommended and most thoroughly tested option.

CockroachDB brings benefits for Zitadel's event-sourcing architecture (strong consistency across distributed nodes, serializable isolation), but it introduces significant operational considerations:

- **Operational complexity.** CockroachDB is a distributed database that requires understanding of node management, replication, range splits, and distributed query planning.
- **Resource overhead.** CockroachDB recommends a minimum of 3 nodes for production, each with 4+ CPU cores and 8+ GB RAM.
- **Ecosystem familiarity.** Most teams have PostgreSQL experience. Fewer have CockroachDB expertise.
- **Managed offerings.** While CockroachDB Cloud exists, PostgreSQL has more managed options (AWS RDS, Azure Database, Google Cloud SQL, Supabase, Neon) at lower cost.

### Rampart and PostgreSQL

Rampart uses PostgreSQL exclusively. PostgreSQL is the most widely deployed open-source relational database, with decades of production hardening, extensive tooling, and broad managed service availability.

- Teams almost certainly already run PostgreSQL or have access to a managed instance.
- No new database technology to learn, deploy, or maintain.
- Works with standard PostgreSQL tooling: pg_dump, pg_restore, pgAdmin, any PostgreSQL-compatible managed service.
- Single-node PostgreSQL is sufficient for most deployments. High availability is achieved with standard PostgreSQL replication.

## Architecture

### Zitadel: Full Event Sourcing

Zitadel uses event sourcing with CQRS (Command Query Responsibility Segregation) as its core architecture. Every state change is stored as an event. Read models (projections) are derived from the event stream.

This provides strong auditability and the ability to rebuild state from events, but it adds complexity:

- Projection lag can cause eventual consistency in reads after writes.
- Event schema evolution requires careful management.
- Debugging requires understanding the event stream, not just the current state.
- Storage grows with every event, not just current state.

### Rampart: CRUD + Audit Log

Rampart uses a traditional CRUD model with an append-only audit event log. Current state is stored directly in normalized tables. Security-relevant changes are recorded as audit events.

This provides the auditability benefits of event sourcing for security purposes while keeping the data model simple, predictable, and easy to query.

## Extensibility

| Extension Type | Rampart | Zitadel |
|---------------|---------|---------|
| Custom logic on events | Webhooks + plugins (planned) | Actions (JS/TS runtime, limited API) |
| Custom auth flows | Plugin system (planned) | Actions on pre/post authentication |
| External integrations | Webhooks, REST API | Actions, REST API |
| Language support | Go (plugins), any (webhooks) | JavaScript/TypeScript only (Actions) |

Zitadel's Actions system runs JavaScript/TypeScript code within the Zitadel process using a sandboxed runtime. This allows custom logic on authentication events (pre/post login, pre/post registration) without external service calls. However, the JavaScript runtime has limited API access and cannot perform arbitrary operations.

Rampart's plugin system (planned) will support Go-native plugins for deep integration and webhooks for language-agnostic event-driven extensions.

## Resource Usage

Both projects are written in Go and compile to single binaries, but their runtime characteristics differ:

| Metric | Rampart | Zitadel |
|--------|---------|---------|
| Idle memory | ~30 MB | ~100–200 MB |
| Binary size | ~20 MB | ~80 MB |
| Startup time | < 1 second | 5–15 seconds (projection rebuild) |
| Database requirements | PostgreSQL (single node) | CockroachDB cluster or PostgreSQL |

Zitadel's higher memory usage comes from its event sourcing architecture: in-memory projections, event stream processing, and the embedded JavaScript runtime for Actions.

## Admin UI and Theming

### Zitadel

Zitadel's admin console is built with Angular using a custom component framework. The login UI uses server-rendered Go templates with CSS customization.

### Rampart

Rampart's admin dashboard and login UI are both built with htmx and Go server-side templates, styled with Tailwind CSS. The login UI supports per-tenant theming via CSS variables with instant preview in the admin dashboard. No server restart is required to change themes.

## Where Zitadel Wins

- **OIDC certification.** Zitadel is officially OIDC certified. Rampart is working toward certification.
- **Event sourcing benefits.** Full event history with the ability to rebuild state. Useful for compliance scenarios that require complete change history beyond audit logs.
- **Actions system.** Available today for custom logic, whereas Rampart's plugin system is still planned.
- **Multi-instance architecture.** Zitadel supports multiple isolated instances within a single deployment, useful for SaaS platforms providing identity as a feature.
- **Apache 2.0 license.** More permissive for embedding in proprietary products.
- **Maturity.** Production-tested at scale with paying customers.

## When to Choose Rampart

- Your infrastructure runs PostgreSQL and you do not want to introduce CockroachDB.
- You prefer a simpler, CRUD-based architecture that is easy to reason about and debug.
- Lower resource usage is important (edge deployments, cost-sensitive environments).
- You want a modern htmx-based admin UI and login experience with easy theming.
- Your team has Go expertise and wants to extend the IAM server in Go.
- Fast startup time matters (CI/CD pipelines, auto-scaling, development).

## When to Choose Zitadel

- You need OIDC certification today.
- Full event sourcing is a compliance or architectural requirement.
- You need the Actions system for custom authentication logic without deploying external services.
- You are already running CockroachDB or plan to.
- Apache 2.0 licensing is required.
- You need a multi-instance architecture for SaaS identity isolation.
