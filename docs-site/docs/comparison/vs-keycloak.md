---
sidebar_position: 1
title: Rampart vs Keycloak
description: A fair comparison between Rampart and Keycloak — resource usage, deployment, extensibility, and trade-offs.
---

# Rampart vs Keycloak

Keycloak is one of the most widely deployed open-source IAM servers, with over 10 years of production use. Rampart takes a different architectural approach — a single Go binary with PostgreSQL — optimizing for simplicity and resource efficiency.

Both are excellent choices depending on your requirements. This page offers a factual comparison to help you decide.

## Comparison Table

| Aspect | Rampart | Keycloak |
|--------|---------|----------|
| **Language** | Go | Java |
| **Memory usage** | ~30 MB idle | 512 MB+ idle (JVM heap) |
| **Startup time** | < 1 second | 10–30 seconds |
| **Binary size** | ~20 MB single binary | 200+ MB (with dependencies) |
| **Deployment** | Single binary, Docker, or systemd | Docker, Kubernetes, or application server |
| **Database** | PostgreSQL | PostgreSQL, MySQL, Oracle, MSSQL |
| **Admin UI** | React + Vite + Tailwind | React (PatternFly, v22+) |
| **Login UI** | React SPA with CSS variable themes | FreeMarker templates |
| **Theming** | CSS variables, instant switching | FreeMarker template overrides |
| **Configuration** | Environment variables + REST API | Admin console, CLI, REST API |
| **Extension model** | Plugin system (planned) | Java SPIs |
| **Protocol support** | OAuth 2.0, OIDC, SAML (planned) | OAuth 2.0, OIDC, SAML, LDAP, Kerberos |
| **Multi-tenancy** | Built-in (organizations) | Realms |
| **License** | AGPL-3.0 | Apache 2.0 |
| **Maturity** | Early stage | 10+ years, battle-tested |

## Resource Usage

Go's compiled nature and lightweight goroutine model result in lower memory and faster startup compared to JVM-based servers.

| Metric | Rampart | Keycloak |
|--------|---------|----------|
| Idle memory | ~30 MB | 512 MB – 1 GB |
| Cold start | < 1 second | 10–30 seconds |
| Docker image size | ~25 MB | ~400 MB |

This difference is most relevant for edge deployments, development environments, CI/CD pipelines, and multi-tenant hosting where resource efficiency matters.

## Deployment

### Rampart

```bash
# Option 1: Download and run
curl -L https://github.com/manimovassagh/rampart/releases/latest/download/rampart-linux-amd64 -o rampart
chmod +x rampart
./rampart serve

# Option 2: Docker
docker run -p 8080:8080 rampart/rampart:latest
```

Single binary with PostgreSQL as the only external dependency.

### Keycloak

Keycloak requires a Java runtime and a database. Production deployments typically involve JVM tuning, clustering configuration via Infinispan, and a reverse proxy. Kubernetes deployments can use the Keycloak Operator.

## Where Keycloak Excels

Keycloak has significant strengths:

- **Maturity.** 10+ years of production use across thousands of organizations. Edge cases and security issues have been discovered and addressed at scale.
- **Protocol breadth.** Native SAML 2.0, LDAP/AD integration, and Kerberos support. Rampart's SAML support is planned but not yet available.
- **Database flexibility.** Supports PostgreSQL, MySQL, Oracle, and MSSQL. Rampart currently supports PostgreSQL only.
- **Ecosystem.** Extensive documentation, community themes, third-party integrations, and commercial support from Red Hat.
- **Certification.** Keycloak is OIDC certified. Rampart aims for certification but has not yet achieved it.
- **Extension system.** Keycloak's SPI system is mature and powerful for organizations with Java expertise.

## Where Rampart Excels

- **Resource efficiency.** Significantly lower memory and faster startup.
- **Deployment simplicity.** Single binary, minimal configuration.
- **Theming.** CSS variable-based themes with instant switching, no server restart needed.
- **Modern stack.** Go backend, React admin UI, built-in observability with Prometheus metrics.

## Migration Considerations

Organizations considering a migration should evaluate:

1. **Feature gap analysis.** Verify that all protocols and features in use are supported by Rampart. SAML, LDAP, and Kerberos are not yet available.
2. **User migration.** Rampart will provide tooling to import users. Password hashes may need re-hashing depending on the configured algorithm.
3. **Client reconfiguration.** OAuth 2.0/OIDC clients will need to be re-registered. Standard protocol endpoints may differ in URL structure.
4. **Theme migration.** Rampart's CSS variable system is different from FreeMarker templates but typically simpler to set up.

## Summary

Rampart is a strong choice if you value lightweight deployment, low resource usage, and a modern developer experience. Keycloak is the proven choice if you need broad protocol support (SAML, LDAP, Kerberos), extensive ecosystem integrations, or enterprise backing from Red Hat.

Both projects serve the open-source IAM community well, and the right choice depends on your specific requirements.
