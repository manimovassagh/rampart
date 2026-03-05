---
sidebar_position: 1
title: Rampart vs Keycloak
description: Detailed comparison between Rampart and Keycloak — resource usage, deployment, extensibility, and migration considerations.
---

# Rampart vs Keycloak

Keycloak is the most widely deployed open-source IAM server. Built on Java and WildFly (now Quarkus), it has been the default choice for organizations needing self-hosted identity management since 2014. Rampart aims to provide the same comprehensive feature set with a fundamentally more efficient and maintainable architecture.

## Comparison Table

| Aspect | Rampart | Keycloak |
|--------|---------|----------|
| **Language** | Go | Java |
| **Memory usage** | ~30 MB idle | 512 MB+ idle (JVM heap) |
| **Startup time** | < 1 second | 10–30 seconds |
| **Binary size** | ~20 MB single binary | 200+ MB (with dependencies) |
| **Deployment** | Single binary, Docker, or systemd | Docker, Kubernetes, or application server |
| **Database** | PostgreSQL | PostgreSQL, MySQL, Oracle, MSSQL |
| **Admin UI** | React + Vite + Tailwind (modern SPA) | FreeMarker templates (legacy) → React (v22+, ongoing migration) |
| **Login UI** | React SPA, CSS variable themes | FreeMarker templates |
| **Theming** | CSS variables, instant switching | FreeMarker template overrides, server restart often required |
| **Configuration** | YAML + REST API (GitOps-friendly) | Admin console, CLI, partial REST API |
| **Extension model** | Plugin system (planned) | Java SPIs (requires Java knowledge, repackaging) |
| **Protocol support** | OAuth 2.0, OIDC, SAML (planned) | OAuth 2.0, OIDC, SAML, LDAP, Kerberos |
| **Multi-tenancy** | Built-in (organizations) | Realms |
| **License** | AGPL-3.0 | Apache 2.0 |
| **Maturity** | Early stage | 10+ years, battle-tested |

## Resource Usage

Keycloak's JVM-based architecture carries inherent overhead. Even a minimal Keycloak deployment requires significant memory allocation for the JVM heap, class loading, and garbage collection.

```
Idle Memory Comparison (approximate):

Rampart:    ████ 30 MB
Keycloak:   ████████████████████████████████████████████████████ 512 MB+
```

| Metric | Rampart | Keycloak |
|--------|---------|----------|
| Idle memory | ~30 MB | 512 MB – 1 GB |
| Per-request overhead | Minimal (goroutines: ~8 KB each) | Significant (thread pools, GC pressure) |
| Cold start | < 1 second | 10–30 seconds |
| Docker image size | ~25 MB | ~400 MB |

This difference matters for edge deployments, multi-tenant hosting, development environments, and CI/CD pipelines where fast startup and low overhead directly reduce cost and iteration time.

## Deployment Complexity

### Rampart

```bash
# Option 1: Download and run
curl -L https://github.com/manimovassagh/rampart/releases/latest/download/rampart-linux-amd64 -o rampart
chmod +x rampart
./rampart serve --config rampart.yaml

# Option 2: Docker
docker run -p 8080:8080 rampart/rampart:latest
```

Single binary. No runtime dependencies beyond PostgreSQL and Redis. No JVM, no application server, no classpath management.

### Keycloak

Keycloak requires a Java runtime, a database, and typically a reverse proxy. Production deployments involve configuring JVM heap sizes, GC tuning, clustering via Infinispan, and managing the Quarkus/WildFly runtime. Kubernetes deployments use the Keycloak Operator, which adds another layer of complexity.

## Admin UI

Keycloak's admin console was historically built on AngularJS and FreeMarker templates. The Keycloak team has been migrating to React (PatternFly) since version 19, but the migration is ongoing and the developer experience reflects the transition.

Rampart's admin UI is built from scratch with React, Vite, and Tailwind CSS. It is a modern single-page application with fast build times, hot module replacement during development, and a responsive design. The UI is embedded in the Go binary — no separate frontend deployment.

## Login Page Theming

Keycloak's theming system is one of its most frequently cited pain points. Customizing login pages requires:

1. Creating FreeMarker template overrides
2. Understanding Keycloak's template variable model
3. Packaging themes into a JAR or Docker image layer
4. Restarting the server to apply changes

Rampart uses CSS variable-based themes. Admins select from built-in themes or create custom themes through the admin dashboard. Theme changes take effect immediately without server restarts. The login UI is a React SPA that reads theme configuration from the server and applies CSS variables at runtime.

## Extensibility

| Extension Type | Rampart | Keycloak |
|---------------|---------|----------|
| Custom auth flows | Plugin system (planned) | Java SPIs (AuthenticatorFactory) |
| Custom user storage | Plugin system (planned) | Java SPIs (UserStorageProvider) |
| Event listeners | Webhooks + plugins (planned) | Java SPIs (EventListenerProvider) |
| Custom endpoints | REST API extension (planned) | Java SPIs (RealmResourceProvider) |

Keycloak's SPI (Service Provider Interface) system is powerful but requires Java development skills, compiling against Keycloak's internal APIs, and repackaging the server. API changes between Keycloak versions frequently break extensions.

Rampart's plugin system (planned) will support Go plugins, webhooks for event-driven integrations, and potentially WASM-based extensions for language-agnostic customization.

## Where Keycloak Wins

Keycloak has meaningful advantages that should be acknowledged:

- **Maturity.** 10+ years of production use across thousands of organizations. Edge cases, bugs, and security issues have been discovered and fixed at scale.
- **Protocol breadth.** Native SAML 2.0, LDAP/AD integration, Kerberos support. Rampart's SAML and LDAP support is planned but not yet available.
- **Database flexibility.** Keycloak supports PostgreSQL, MySQL, Oracle, and MSSQL. Rampart currently supports PostgreSQL only.
- **Ecosystem.** Extensive documentation, community themes, third-party integrations, commercial support from Red Hat.
- **Certification.** Keycloak is OIDC certified. Rampart aims for certification but has not yet achieved it.

## Migration Considerations

Organizations considering a migration from Keycloak to Rampart should evaluate:

1. **Feature gap analysis.** Verify that all protocols and features in use are supported by Rampart. SAML, LDAP, and Kerberos are not yet available.
2. **User migration.** Rampart will provide tooling to import users from Keycloak exports. Password hashes may need re-hashing depending on the algorithm Keycloak was configured to use.
3. **Client reconfiguration.** OAuth 2.0/OIDC clients will need to be re-registered in Rampart. The standard protocol endpoints may differ in URL structure.
4. **Theme migration.** FreeMarker themes cannot be reused. Rampart's CSS variable system requires different customization, but the migration is typically simpler.
5. **Extension migration.** Java SPIs must be reimplemented using Rampart's extension mechanisms.

A detailed migration guide will be published when Rampart reaches feature parity for core OAuth 2.0/OIDC flows.

## Summary

Rampart is the right choice if you want a lightweight, modern IAM server that is easy to deploy, easy to theme, and efficient with resources. Keycloak is the safer choice today if you need battle-tested maturity, SAML/LDAP support, or Red Hat enterprise backing.

As Rampart matures, the feature gap narrows while the architectural advantages remain permanent.
