# Rampart Spring Boot Sample Backend

A minimal Spring Boot backend that demonstrates Rampart IAM integration using Spring Security OAuth2 Resource Server. This is a **drop-in replacement** for the Express sample backend — same port (3001), same endpoints, same JSON responses.

## Prerequisites

- Java 17+
- Maven 3.8+
- A running Rampart server (default: `http://localhost:8080`)

## Quick Start

```bash
# From this directory
./mvnw spring-boot:run

# Or with a custom Rampart issuer
RAMPART_ISSUER=https://auth.example.com ./mvnw spring-boot:run
```

If you don't have the Maven wrapper, use your system Maven:

```bash
mvn spring-boot:run
```

## Configuration

| Property / Env Var | Default | Description |
|---|---|---|
| `rampart.issuer` / `RAMPART_ISSUER` | `http://localhost:8080` | Rampart server URL |
| `server.port` / `PORT` | `3001` | Server listen port |

## Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/health` | Public | Health check, returns `{"status":"ok","issuer":"..."}` |
| GET | `/api/profile` | JWT required | User profile from JWT claims |
| GET | `/api/claims` | JWT required | All raw JWT claims |
| GET | `/api/editor/dashboard` | `editor` role | Editor dashboard (mock data) |
| GET | `/api/manager/reports` | `manager` role | Manager reports (mock data) |

## How It Works

This sample uses Spring Security OAuth2 Resource Server directly, following the same approach as the `rampart-spring-boot-starter`:

1. **JWT Validation** — Tokens are verified against the JWKS endpoint at `{issuer}/.well-known/jwks.json`
2. **Role Mapping** — The `roles` claim from Rampart JWTs is mapped to Spring Security `ROLE_*` authorities
3. **RBAC** — Endpoints use `.hasRole("editor")` and `.hasRole("manager")` for access control

Once the `rampart-spring-boot-starter` is published to Maven Central, you can replace the manual security config with the starter dependency for even simpler setup.

## Switching from Express

This backend is API-compatible with the Express sample. To switch:

1. Stop the Express backend
2. Start this Spring Boot backend
3. The React frontend will work without any changes (same port, same endpoints, same JSON format)
