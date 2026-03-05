---
sidebar_position: 1
title: SDK Adapters Overview
description: Overview of all official Rampart SDK adapters for integrating authentication and authorization into your applications.
---

# SDK Adapters Overview

Rampart provides official SDK adapters as thin wrappers around its standard OAuth 2.0 and OpenID Connect endpoints. Each adapter handles OIDC discovery, token verification, session management, and user context — so you can protect your application with a few lines of code.

## Available Adapters

| Adapter | Language / Framework | Use Case |
|---------|---------------------|----------|
| [`@rampart/node`](./node.md) | Node.js / Express | Backend APIs, server-rendered apps |
| [`@rampart/react`](./react.md) | React (SPA) | Single-page applications with PKCE |
| [`@rampart/nextjs`](./nextjs.md) | Next.js (App Router) | Full-stack Next.js applications |
| [`rampart-go`](./go.md) | Go (net/http, chi, gin, fiber) | Go microservices and APIs |
| [`rampart-python`](./python.md) | Python (FastAPI, Flask) | Python APIs and web apps |
| [`rampart-spring-boot`](./spring-boot.md) | Java (Spring Boot) | Enterprise Java applications |

## Compatibility Matrix

| Adapter | Min Runtime | Rampart Server | OIDC Discovery | PKCE | Token Refresh | RBAC |
|---------|-------------|----------------|----------------|------|---------------|------|
| `@rampart/node` | Node 18+ | v0.1+ | Yes | N/A | Yes | Yes |
| `@rampart/react` | React 18+ | v0.1+ | Yes | Yes | Yes | Yes |
| `@rampart/nextjs` | Next.js 14+ | v0.1+ | Yes | Yes | Yes | Yes |
| `rampart-go` | Go 1.21+ | v0.1+ | Yes | N/A | Yes | Yes |
| `rampart-python` | Python 3.10+ | v0.1+ | Yes | N/A | Yes | Yes |
| `rampart-spring-boot` | Java 17+ / Spring Boot 3.x | v0.1+ | Yes | N/A | Yes | Yes |

## Common Configuration

All adapters share a common set of configuration values. These can be passed as constructor options or read from environment variables.

### Environment Variables

```bash
# Required
RAMPART_URL=https://auth.example.com        # Base URL of your Rampart server
RAMPART_CLIENT_ID=my-app                     # OAuth 2.0 client ID
RAMPART_CLIENT_SECRET=secret                 # OAuth 2.0 client secret (confidential clients only)

# Optional
RAMPART_REALM=default                        # Organization/realm (default: "default")
RAMPART_SCOPES=openid profile email          # Requested scopes
RAMPART_REDIRECT_URI=http://localhost:3000/callback  # OAuth callback URL
```

### OIDC Discovery

Every adapter automatically fetches the OpenID Connect discovery document from your Rampart server:

```
GET {RAMPART_URL}/.well-known/openid-configuration
```

This provides all endpoint URLs (authorization, token, userinfo, JWKS, etc.) so you never need to hardcode them.

### JWKS Verification

Token verification uses the JSON Web Key Set published at:

```
GET {RAMPART_URL}/.well-known/jwks.json
```

All adapters cache JWKS keys and refresh them automatically when key rotation occurs.

## When to Use Which Adapter

### Backend API (no browser)

Use **`@rampart/node`**, **`rampart-go`**, **`rampart-python`**, or **`rampart-spring-boot`** depending on your language. These adapters verify incoming bearer tokens from the `Authorization` header and extract user claims.

```
Client (mobile app, CLI, other service)
  → sends Bearer token in Authorization header
  → your API verifies token with Rampart JWKS
  → extracts user claims and enforces permissions
```

### Single-Page Application (SPA)

Use **`@rampart/react`** for a standalone React SPA. It implements the Authorization Code flow with PKCE — the recommended flow for public clients. The adapter manages the full lifecycle: redirect to login, handle callback, store tokens, refresh silently.

### Full-Stack Next.js

Use **`@rampart/nextjs`** when you need both server-side and client-side auth in a Next.js application. It provides middleware for protecting routes at the edge, server-side token verification in Server Components, and a client-side auth context for interactive pages.

### Choosing Between Confidential and Public Clients

| Client Type | Has a Backend? | Can Store Secrets? | Flow |
|-------------|---------------|--------------------|------|
| **Confidential** | Yes | Yes | Authorization Code |
| **Public** | No (SPA, mobile) | No | Authorization Code + PKCE |

- SPAs and mobile apps are **public clients** — they cannot securely store a client secret. Use PKCE.
- Backend services are **confidential clients** — they exchange a client secret for tokens.

## Token Format

Rampart issues standard JWTs. A decoded access token looks like:

```json
{
  "iss": "https://auth.example.com",
  "sub": "user_01H8MZXK9Q2YPT4N6JKWER3FGH",
  "aud": "my-app",
  "exp": 1709654400,
  "iat": 1709650800,
  "scope": "openid profile email",
  "org_id": "org_01H8MZXK9Q2YPT4N6JKWER3ABC",
  "roles": ["admin", "editor"],
  "email": "user@example.com",
  "name": "Jane Doe"
}
```

All adapters provide typed access to these claims.

## Error Handling

All adapters follow a consistent error model:

| Error | HTTP Status | Meaning |
|-------|-------------|---------|
| `TokenExpiredError` | 401 | Access token has expired — refresh or re-authenticate |
| `TokenInvalidError` | 401 | Token signature or claims are invalid |
| `InsufficientScopeError` | 403 | Token lacks required scopes |
| `InsufficientRoleError` | 403 | User lacks required roles |
| `DiscoveryError` | 500 | Could not fetch OIDC discovery document |

## Next Steps

Pick the adapter for your stack and follow the integration guide:

- [Node.js / Express](./node.md)
- [React SPA](./react.md)
- [Next.js](./nextjs.md)
- [Go](./go.md)
- [Python](./python.md)
- [Spring Boot](./spring-boot.md)
