---
sidebar_position: 1
title: API Overview
description: Design principles, authentication, error handling, pagination, and rate limiting for the Rampart REST API.
---

# API Overview

Rampart exposes a RESTful JSON API for all identity and access management operations. Every feature available in the admin console or CLI is backed by this API, making it the single source of truth for integrations, automation, and custom tooling.

## Design Principles

Rampart's API follows these core design principles:

- **REST** -- Resources are identified by URLs, manipulated with standard HTTP methods (GET, POST, PUT, DELETE), and represented as JSON.
- **Versioned** -- The Admin API is versioned via the URL path (`/api/v1/`). Breaking changes only happen in new major versions.
- **Consistent** -- All endpoints follow the same conventions for errors, pagination, filtering, and response envelopes.
- **Specification-compliant** -- OAuth 2.0 and OpenID Connect endpoints follow their respective RFCs exactly (RFC 6749, RFC 7009, RFC 7517, RFC 7636, RFC 7662).
- **API-first** -- Every feature is accessible via the API before it appears in the admin console or CLI.

## Base URL

All API endpoints are relative to your Rampart instance base URL:

```
https://your-rampart-instance
```

The API is organized into three endpoint groups:

| Group | Base Path | Purpose |
|-------|-----------|---------|
| **OAuth / OIDC** | `/oauth/` | Token issuance, authorization, revocation, introspection, UserInfo |
| **Discovery** | `/.well-known/` | OpenID Connect discovery and JWKS |
| **Admin API** | `/api/v1/admin/` | User, organization, role, client, session, and audit management |

### URL Convention

- All Admin API paths use plural nouns: `/api/v1/admin/users`, `/api/v1/admin/organizations`.
- Individual resources are accessed by ID: `/api/v1/admin/users/{user_id}`.
- Sub-resources are nested: `/api/v1/admin/users/{user_id}/sessions`.
- Query parameters use `snake_case`.
- No trailing slashes. Requests with trailing slashes receive a 301 redirect.

## API Versioning

The Admin API is versioned via the URL path (`/api/v1/`). OAuth and OIDC endpoints follow their respective RFCs and are not versioned -- their paths are stable by specification.

When a new API version is introduced, the previous version will continue to be supported for at least 12 months with deprecation notices in response headers:

```
Deprecation: true
Sunset: Sat, 01 Mar 2028 00:00:00 GMT
Link: <https://your-rampart-instance/api/v2/admin/users>; rel="successor-version"
```

## Authentication

### Admin API

All `/api/v1/admin/` endpoints require a valid Bearer token in the `Authorization` header. The token must belong to a user or service account with the appropriate admin role.

```bash
curl -X GET https://your-rampart-instance/api/v1/admin/users \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Content-Type: application/json"
```

To obtain an admin token, authenticate using the OAuth token endpoint with a client that has admin scopes:

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=admin-cli" \
  -d "client_secret=your-client-secret" \
  -d "scope=openid admin"
```

**Response:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid admin"
}
```

Use the returned `access_token` as the Bearer token for all subsequent Admin API calls.

### OAuth Endpoints

OAuth endpoints use standard OAuth 2.0 authentication mechanisms:

- **Confidential clients** authenticate via HTTP Basic (`Authorization: Basic base64(client_id:client_secret)`) or by including `client_id` and `client_secret` in the request body.
- **Public clients** (SPAs, mobile apps) use PKCE and do not send a client secret. Only the `client_id` is required.
- **Token introspection and revocation** require client authentication.

### Client Authentication Methods

| Method | Header / Parameter | Use Case |
|--------|-------------------|----------|
| `client_secret_basic` | `Authorization: Basic base64(id:secret)` | Server-side applications (recommended) |
| `client_secret_post` | `client_id` + `client_secret` in body | When Basic auth is not practical |
| `none` | `client_id` only | Public clients (SPAs, mobile apps) with PKCE |

### API Keys (Future)

Long-lived API keys for service-to-service integrations are planned for a future release.

## Content Type

All request and response bodies use JSON (`application/json`), except for the OAuth token endpoint which accepts `application/x-www-form-urlencoded` as required by RFC 6749.

```bash
# Admin API -- JSON body
curl -X POST https://your-rampart-instance/api/v1/admin/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jane.doe",
    "email": "jane@example.com",
    "first_name": "Jane",
    "last_name": "Doe",
    "enabled": true
  }'
```

```bash
# OAuth token endpoint -- form-urlencoded body
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=my-service" \
  -d "client_secret=secret123"
```

## Error Format

All errors follow a consistent JSON structure:

```json
{
  "error": "not_found",
  "error_description": "User with ID '550e8400-e29b-41d4-a716-446655440000' was not found.",
  "request_id": "req_abc123def456"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `error` | string | Machine-readable error code (e.g., `invalid_request`, `not_found`, `forbidden`) |
| `error_description` | string | Human-readable message with details about what went wrong |
| `request_id` | string | Unique identifier for the request, useful for support and debugging |

### OAuth Error Codes

OAuth endpoints return error codes defined by RFC 6749 and RFC 7009:

| Error Code | HTTP Status | Meaning |
|------------|-------------|---------|
| `invalid_request` | 400 | Missing or malformed parameter |
| `invalid_client` | 401 | Client authentication failed |
| `invalid_grant` | 400 | Authorization code or refresh token is invalid or expired |
| `unauthorized_client` | 400 | Client is not authorized for this grant type |
| `unsupported_grant_type` | 400 | The grant type is not supported |
| `invalid_scope` | 400 | Requested scope is invalid or exceeds granted scopes |
| `access_denied` | 403 | Resource owner denied the request |
| `server_error` | 500 | Unexpected internal error |

### Admin API Error Codes

| Error Code | HTTP Status | Meaning |
|------------|-------------|---------|
| `bad_request` | 400 | Invalid request body or parameters |
| `unauthorized` | 401 | Missing or invalid Bearer token |
| `forbidden` | 403 | Authenticated but lacks required permissions |
| `not_found` | 404 | Requested resource does not exist |
| `conflict` | 409 | Resource already exists (e.g., duplicate username or email) |
| `validation_error` | 422 | Request body failed validation |
| `rate_limited` | 429 | Too many requests |
| `internal_error` | 500 | Unexpected server error |

### Validation Errors

When a request body fails validation, the response includes field-level details:

```json
{
  "error": "validation_error",
  "error_description": "Request body failed validation.",
  "request_id": "req_abc123def456",
  "details": [
    {
      "field": "email",
      "message": "Must be a valid email address."
    },
    {
      "field": "username",
      "message": "Must be between 3 and 128 characters."
    },
    {
      "field": "password",
      "message": "Must contain at least one uppercase letter, one lowercase letter, one digit, and one special character."
    }
  ]
}
```

### HTTP Status Code Summary

| Status | Meaning |
|--------|---------|
| 200 | Success (GET, PUT, POST for token operations) |
| 201 | Resource created (POST) |
| 204 | Success with no content (DELETE) |
| 301 | Redirect (trailing slash removal) |
| 302 | OAuth redirect (authorize endpoint) |
| 400 | Bad request |
| 401 | Authentication required or failed |
| 403 | Forbidden (insufficient permissions) |
| 404 | Resource not found |
| 409 | Conflict (duplicate resource) |
| 422 | Validation error |
| 429 | Rate limited |
| 500 | Internal server error |

## Pagination

All list endpoints in the Admin API support cursor-based pagination. This provides consistent results even when records are added or removed between pages.

### Request Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 20 | Number of items per page (1--100) |
| `cursor` | string | -- | Opaque cursor from a previous response |
| `sort` | string | `created_at` | Field to sort by (varies per resource) |
| `order` | string | `desc` | Sort direction: `asc` or `desc` |

### Response Envelope

All list responses use a consistent envelope:

```json
{
  "data": [
    { "id": "usr_001", "username": "jane.doe", "email": "jane@example.com" },
    { "id": "usr_002", "username": "john.smith", "email": "john@example.com" }
  ],
  "pagination": {
    "total": 1284,
    "limit": 20,
    "has_more": true,
    "next_cursor": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wMVQxMjowMDowMFoiLCJpZCI6InVzcl8wMDIifQ=="
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `data` | array | Array of resource objects |
| `pagination.total` | integer | Total number of matching records |
| `pagination.limit` | integer | Number of items requested per page |
| `pagination.has_more` | boolean | Whether more pages exist |
| `pagination.next_cursor` | string | Cursor to pass as `cursor` parameter for the next page (absent if `has_more` is `false`) |

### Paginating Through Results

```bash
# First page
curl -X GET "https://your-rampart-instance/api/v1/admin/users?limit=20" \
  -H "Authorization: Bearer <token>"

# Next page (using next_cursor from previous response)
curl -X GET "https://your-rampart-instance/api/v1/admin/users?limit=20&cursor=eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wMVQxMjowMDowMFoiLCJpZCI6InVzcl8wMDIifQ==" \
  -H "Authorization: Bearer <token>"
```

Continue fetching pages until `has_more` is `false`.

## Filtering

List endpoints support field-level filtering via query parameters:

```bash
# Find enabled users in a specific organization
curl -X GET "https://your-rampart-instance/api/v1/admin/users?enabled=true&organization_id=org_abc123" \
  -H "Authorization: Bearer <token>"

# Search users by email domain
curl -X GET "https://your-rampart-instance/api/v1/admin/users?search=@example.com" \
  -H "Authorization: Bearer <token>"

# Filter audit events by type and date range
curl -X GET "https://your-rampart-instance/api/v1/admin/events?type=user.login&from=2026-03-01T00:00:00Z&to=2026-03-05T23:59:59Z" \
  -H "Authorization: Bearer <token>"
```

The `search` parameter performs a case-insensitive substring match against common text fields (username, email, first name, last name for users; name and slug for organizations).

### Common Filter Parameters

| Parameter | Type | Available On | Description |
|-----------|------|-------------|-------------|
| `search` | string | Users, Organizations, Clients | Free-text search across common fields |
| `enabled` | boolean | Users | Filter by enabled/disabled status |
| `organization_id` | string | Users, Clients, Sessions | Filter by organization |
| `role` | string | Users | Filter by role name |
| `type` | string | Events | Filter by event type |
| `from` | ISO 8601 | Events | Start of date range |
| `to` | ISO 8601 | Events | End of date range |
| `user_id` | string | Sessions, Events | Filter by user |
| `client_id` | string | Sessions | Filter by client |

## Rate Limiting

Rampart enforces rate limits to protect against abuse. Limits are applied per client IP for unauthenticated endpoints and per access token for authenticated endpoints.

### Default Limits

| Endpoint Group | Limit | Window |
|----------------|-------|--------|
| `POST /oauth/token` | 20 requests | per minute per IP |
| `POST /oauth/authorize` | 30 requests | per minute per IP |
| `POST /oauth/introspect` | 100 requests | per minute per client |
| Admin API (read) | 300 requests | per minute per token |
| Admin API (write) | 60 requests | per minute per token |

### Rate Limit Headers

Every response includes rate limit headers:

```
X-RateLimit-Limit: 300
X-RateLimit-Remaining: 287
X-RateLimit-Reset: 1709510460
```

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests allowed in the current window |
| `X-RateLimit-Remaining` | Requests remaining in the current window |
| `X-RateLimit-Reset` | Unix timestamp when the window resets |
| `Retry-After` | Seconds to wait before retrying (only present on 429 responses) |

When rate limited, the server returns HTTP 429:

```json
{
  "error": "rate_limited",
  "error_description": "Too many requests. Please retry after 12 seconds.",
  "request_id": "req_xyz789"
}
```

Rate limits are configurable in the Rampart server configuration file under the `rate_limiting` section.

## Request IDs

Every response includes a `X-Request-Id` header containing a unique identifier for the request. This ID also appears in server logs and error responses, making it straightforward to correlate client-side issues with server-side logs.

```
X-Request-Id: req_abc123def456
```

If you provide a `X-Request-Id` header in your request, Rampart will use your value instead of generating one. This is useful for tracing requests across multiple services.

## CORS

Cross-Origin Resource Sharing (CORS) is configured per client application. When registering a client in the admin console or via the Admin API, you specify allowed redirect URIs and web origins. Rampart uses these origins to set CORS headers on OAuth and discovery endpoints.

```
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 86400
```

Admin API endpoints do not support CORS by default -- they are intended for server-to-server or admin console use only. If you need to call Admin API endpoints from a browser, configure the `admin.cors_origins` setting.

## Health Check

Rampart exposes health endpoints for load balancers and monitoring:

```bash
curl -X GET https://your-rampart-instance/healthz
```

```json
{
  "status": "ok"
}
```

The health endpoint (`/healthz`) returns HTTP 200 when the server is running. A readiness probe is available at `/readyz` — it returns HTTP 200 only when the server is ready to accept traffic (database connectivity verified), and HTTP 503 when the database is unhealthy.

## Idempotency

Write operations (POST) support idempotency via the `Idempotency-Key` header. If you send the same request with the same idempotency key within a 24-hour window, Rampart returns the original response without performing the operation again.

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: usr-creation-abc123" \
  -d '{
    "username": "jane.doe",
    "email": "jane@example.com",
    "first_name": "Jane",
    "last_name": "Doe",
    "enabled": true,
    "password": "SecureP@ssw0rd!"
  }'
```

PUT and DELETE operations are naturally idempotent and do not require this header.

## Common curl Examples

### Authenticate and get an admin token

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=admin-cli" \
  -d "client_secret=your-secret" \
  -d "scope=openid admin"
```

### List users with pagination and filtering

```bash
curl -X GET "https://your-rampart-instance/api/v1/admin/users?limit=10&order=asc&search=jane" \
  -H "Authorization: Bearer <token>"
```

### Create a user

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Johnson",
    "enabled": true,
    "password": "SecureP@ssw0rd!"
  }'
```

### Update a user

```bash
curl -X PUT https://your-rampart-instance/api/v1/admin/users/usr_1234567890 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Alice",
    "last_name": "Smith",
    "enabled": false
  }'
```

### Delete a user

```bash
curl -X DELETE https://your-rampart-instance/api/v1/admin/users/usr_1234567890 \
  -H "Authorization: Bearer <token>"
```

### Introspect a token

```bash
curl -X POST https://your-rampart-instance/oauth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-service:secret123" \
  -d "token=eyJhbGciOiJSUzI1NiIs..."
```

### Fetch OIDC discovery document

```bash
curl -X GET https://your-rampart-instance/.well-known/openid-configuration
```

### Check server health

```bash
curl -X GET https://your-rampart-instance/health
```
