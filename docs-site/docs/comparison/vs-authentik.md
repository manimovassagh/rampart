---
sidebar_position: 4
title: Rampart vs Authentik
description: Comparison between Rampart and Authentik — performance, deployment model, resource usage, and architecture differences.
---

# Rampart vs Authentik

Authentik is an open-source identity provider built in Python (Django) with a focus on flexibility and a polished admin interface. It has gained significant adoption as a self-hosted identity solution, particularly in the homelab and small business space. Rampart differs fundamentally in runtime performance, deployment model, and resource efficiency.

## Comparison Table

| Aspect | Rampart | Authentik |
|--------|---------|-----------|
| **Language** | Go | Python (Django) |
| **Runtime** | Single compiled binary | Python interpreter + Gunicorn/Uvicorn |
| **Memory usage** | ~30 MB idle | 500 MB – 1 GB+ |
| **Request throughput** | High (compiled, goroutines) | Lower (interpreted, GIL constraints) |
| **Deployment** | Single binary, Docker optional | Docker required (multiple containers) |
| **Required services** | PostgreSQL, Redis | PostgreSQL, Redis, worker process, server process |
| **Admin UI** | React + Vite + Tailwind | lit-element web components |
| **Login UI** | React SPA, CSS variable themes | Flow-based, customizable |
| **Configuration** | YAML + REST API | Admin UI + REST API, YAML (limited) |
| **Flow engine** | Standard OAuth 2.0/OIDC flows | Visual flow designer (stages, policies) |
| **Protocol support** | OAuth 2.0, OIDC, SAML (planned) | OAuth 2.0, OIDC, SAML, LDAP, SCIM, Proxy |
| **License** | AGPL-3.0 | MIT (recently changed from custom) |

## Performance

The most significant difference between Rampart and Authentik is runtime performance. Go and Python have fundamentally different execution characteristics for server workloads.

### Throughput

Go's compiled nature, lightweight goroutine concurrency model, and efficient memory management give Rampart a substantial performance advantage for authentication workloads:

| Metric | Rampart (Go) | Authentik (Python/Django) |
|--------|-------------|--------------------------|
| Concurrent connections | Thousands (goroutines: ~8 KB each) | Hundreds (OS threads or async workers) |
| Token issuance throughput | ~5,000–10,000 req/s | ~500–1,000 req/s |
| P99 latency (token endpoint) | < 5 ms | 20–50 ms |
| CPU efficiency | Compiled, no interpreter overhead | Interpreted, GIL limits true parallelism |

These numbers are estimated based on typical Go vs Python web framework benchmarks and have not been measured against production Rampart builds. The order-of-magnitude difference is consistent across independent benchmarks of Go vs Python web frameworks.

### Why This Matters

For small deployments (< 100 users), both perform adequately. The difference becomes significant at scale:

- **Enterprise deployments** handling thousands of authentication requests per minute benefit from Rampart's lower latency and higher throughput per node.
- **Multi-tenant hosting** where a single instance serves many organizations benefits from Rampart's lower per-request overhead.
- **Cost efficiency** — fewer and smaller nodes required for the same workload means lower infrastructure cost.

## Deployment Model

### Authentik

Authentik requires Docker Compose (or Kubernetes) with multiple containers:

```yaml
# Authentik docker-compose.yml (simplified)
services:
  server:          # Django web server
  worker:          # Celery background worker
  postgresql:      # Database
  redis:           # Cache and message broker
```

A minimum Authentik deployment runs 4 containers. The server and worker are separate Python processes that must be deployed and scaled together. The worker handles background tasks (email, LDAP sync, flow execution) and is required for core functionality.

### Rampart

```bash
# Option 1: Single binary
./rampart serve --config rampart.yaml

# Option 2: Docker (single container)
docker run -p 8080:8080 rampart/rampart:latest
```

Rampart runs as a single process. Background tasks run as goroutines within the same process. No separate worker, no container orchestration required for basic deployments.

| Deployment aspect | Rampart | Authentik |
|-------------------|---------|-----------|
| Minimum containers | 1 (+ PostgreSQL, Redis) | 4 (server, worker, PostgreSQL, Redis) |
| Background processing | In-process goroutines | Separate Celery worker (required) |
| Without Docker | Yes (native binary) | Not officially supported |
| Systemd service | Single unit file | Multiple unit files or Docker dependency |

## Resource Usage

| Resource | Rampart | Authentik |
|----------|---------|-----------|
| Idle memory | ~30 MB | 500 MB – 1 GB |
| Docker image size | ~25 MB | ~1 GB (with dependencies) |
| Startup time | < 1 second | 15–30 seconds |
| Disk space (runtime) | ~20 MB binary | ~500 MB (Python packages, static files) |

Authentik's resource usage comes from the Python runtime, Django framework, installed packages, and the requirement to run multiple processes. The Celery worker alone consumes 200–400 MB.

```
Idle Memory Comparison (approximate):

Rampart:    ███ 30 MB
Authentik:  ████████████████████████████████████████████████████████████████████ 800 MB
```

## Feature Comparison

### Authentik's Flow Engine

Authentik's most distinctive feature is its visual flow designer. Authentication flows are composed from stages (login, MFA, consent) and policies (conditional logic) through a drag-and-drop interface. This allows non-developer administrators to build complex authentication sequences.

Rampart implements standard OAuth 2.0/OIDC flows directly. Custom flow logic will be supported through the plugin system. This approach is less visually flexible but more predictable and easier to reason about for standard use cases.

### Protocol Support

Authentik has broader protocol support today:

| Protocol | Rampart | Authentik |
|----------|---------|-----------|
| OAuth 2.0 | Yes | Yes |
| OIDC | Yes | Yes |
| SAML 2.0 | Planned | Yes |
| LDAP (outbound) | Planned | Yes (LDAP provider) |
| SCIM 2.0 | Planned (enterprise) | Yes |
| Proxy authentication | No | Yes (forward auth, nginx/Traefik integration) |

Authentik's proxy provider and LDAP outbound provider are particularly useful for legacy application integration.

## Where Authentik Wins

- **Flow engine.** The visual flow designer allows complex, conditional authentication flows without code. Powerful for organizations with non-standard requirements.
- **Protocol breadth.** SAML, LDAP outbound, SCIM, and proxy authentication are available today.
- **Community and ecosystem.** Large, active community with extensive documentation and community-contributed blueprints.
- **Blueprints.** Declarative YAML-based configuration as code for reproducible deployments.
- **MIT license.** More permissive than AGPL-3.0.
- **Maturity.** Production-tested with a large user base and active development.

## When to Choose Rampart

- Performance matters — you need high throughput, low latency, or efficient resource usage.
- You want a single binary deployment without Docker as a hard requirement.
- You prefer lower operational complexity (one process vs four).
- Your authentication flows follow standard OAuth 2.0/OIDC patterns.
- You are running in resource-constrained environments (edge, small VMs, ARM devices).
- You want a modern React-based admin UI.
- Your team has Go expertise for extending and contributing.

## When to Choose Authentik

- You need the visual flow designer for complex, conditional authentication flows.
- You need SAML, LDAP, SCIM, or proxy authentication today.
- Docker-based deployment is standard in your environment.
- You want a large community with established documentation and support.
- MIT licensing is preferred over AGPL-3.0.
- Performance at the level of hundreds of requests per second (not thousands) is sufficient.
