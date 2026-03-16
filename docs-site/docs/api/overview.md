---
sidebar_position: 1
title: API Overview
description: Complete endpoint reference, design principles, authentication, error handling, pagination, and rate limiting for the Rampart REST API.
---

# API Overview

Rampart exposes a RESTful JSON API for all identity and access management operations. Every feature available in the admin console or CLI is backed by this API, making it the single source of truth for integrations, automation, and custom tooling.

## Complete Endpoint Reference

All endpoints exposed by Rampart, organized by category. Auth column indicates: **No** (public), **Yes** (Bearer token required), **Admin** (admin role required).

### Authentication

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/register` | No | User self-registration |
| POST | `/login` | No | Authenticate with username/email and password |
| POST | `/logout` | No | Invalidate a session by refresh token |
| POST | `/token/refresh` | No | Exchange a refresh token for a new access token |
| GET | `/me` | Yes | Get the authenticated user's profile |

### Password Reset

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/forgot-password` | No | Request a password reset email |
| POST | `/reset-password` | No | Reset password using a reset token |

### Email Verification

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/verify-email/send` | No | Send a verification email |
| GET | `/verify-email` | No | Verify email using a token (query param) |

### MFA -- TOTP

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/mfa/totp/enroll` | Yes | Start TOTP enrollment, returns QR code / secret |
| POST | `/mfa/totp/verify-setup` | Yes | Confirm TOTP setup with a verification code |
| POST | `/mfa/totp/disable` | Yes | Disable TOTP for the authenticated user |
| POST | `/mfa/totp/verify` | No | Verify a TOTP code during login (uses MFA token) |

### MFA -- WebAuthn / Passkeys

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/mfa/webauthn/register/begin` | Yes | Begin WebAuthn credential registration |
| POST | `/mfa/webauthn/register/complete` | Yes | Complete WebAuthn credential registration |
| GET | `/mfa/webauthn/credentials` | Yes | List WebAuthn credentials for the authenticated user |
| DELETE | `/mfa/webauthn/credentials/{id}` | Yes | Delete a WebAuthn credential |
| POST | `/mfa/webauthn/login/begin` | No | Begin WebAuthn login challenge (uses MFA token) |
| POST | `/mfa/webauthn/login/complete` | No | Complete WebAuthn login challenge |

### OAuth 2.0

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/oauth/authorize` | No | Authorization endpoint -- initiates user authentication |
| POST | `/oauth/authorize` | No | Authorization endpoint -- form submission handler |
| POST | `/oauth/consent` | No | User consent submission |
| POST | `/oauth/token` | Conditional | Token endpoint -- issues and refreshes tokens |
| POST | `/oauth/revoke` | Yes | Revocation endpoint -- invalidates tokens (RFC 7009) |

### Social Login

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/oauth/social/{provider}` | No | Initiate social login (redirect to provider) |
| GET | `/oauth/social/{provider}/callback` | No | Social login callback (GET) |
| POST | `/oauth/social/{provider}/callback` | No | Social login callback (POST, e.g. Apple Sign In) |

### SAML

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/saml/providers` | No | List configured SAML identity providers |
| GET | `/saml/{providerID}/metadata` | No | SP metadata XML for a SAML provider |
| GET | `/saml/{providerID}/login` | No | Initiate SAML SSO login (redirect to IdP) |
| POST | `/saml/{providerID}/acs` | No | Assertion Consumer Service (receives SAML response) |

### OIDC Discovery

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/.well-known/openid-configuration` | No | OpenID Connect discovery document |
| GET | `/.well-known/jwks.json` | No | JSON Web Key Set for token verification |

### SCIM 2.0 Provisioning

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/scim/v2/ServiceProviderConfig` | Admin | SCIM service provider configuration |
| GET | `/scim/v2/ResourceTypes` | Admin | SCIM resource type definitions |
| GET | `/scim/v2/Schemas` | Admin | SCIM schema definitions |
| GET | `/scim/v2/Users` | Admin | List SCIM users |
| POST | `/scim/v2/Users` | Admin | Create a SCIM user |
| GET | `/scim/v2/Users/{id}` | Admin | Get a SCIM user |
| PUT | `/scim/v2/Users/{id}` | Admin | Replace a SCIM user |
| PATCH | `/scim/v2/Users/{id}` | Admin | Partially update a SCIM user |
| DELETE | `/scim/v2/Users/{id}` | Admin | Delete a SCIM user |
| GET | `/scim/v2/Groups` | Admin | List SCIM groups |
| POST | `/scim/v2/Groups` | Admin | Create a SCIM group |
| GET | `/scim/v2/Groups/{id}` | Admin | Get a SCIM group |
| PUT | `/scim/v2/Groups/{id}` | Admin | Replace a SCIM group |
| PATCH | `/scim/v2/Groups/{id}` | Admin | Partially update a SCIM group |
| DELETE | `/scim/v2/Groups/{id}` | Admin | Delete a SCIM group |

### Admin API -- Users

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/stats` | Admin | Dashboard statistics (user count, session count, etc.) |
| GET | `/api/v1/admin/users` | Admin | List users with pagination and filtering |
| POST | `/api/v1/admin/users` | Admin | Create a new user |
| GET | `/api/v1/admin/users/{id}` | Admin | Get a user by ID |
| PUT | `/api/v1/admin/users/{id}` | Admin | Update a user |
| DELETE | `/api/v1/admin/users/{id}` | Admin | Delete a user |
| POST | `/api/v1/admin/users/{id}/reset-password` | Admin | Admin password reset for a user |
| GET | `/api/v1/admin/users/{id}/sessions` | Admin | List sessions for a user |
| DELETE | `/api/v1/admin/users/{id}/sessions` | Admin | Revoke all sessions for a user |

### Admin API -- Organizations

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/organizations` | Admin | List organizations |
| POST | `/api/v1/admin/organizations` | Admin | Create an organization |
| GET | `/api/v1/admin/organizations/{id}` | Admin | Get an organization |
| PUT | `/api/v1/admin/organizations/{id}` | Admin | Update an organization |
| DELETE | `/api/v1/admin/organizations/{id}` | Admin | Delete an organization |
| GET | `/api/v1/admin/organizations/{id}/settings` | Admin | Get organization settings |
| PUT | `/api/v1/admin/organizations/{id}/settings` | Admin | Update organization settings |
| GET | `/api/v1/admin/organizations/{id}/export` | Admin | Export organization configuration |
| POST | `/api/v1/admin/organizations/{id}/import` | Admin | Import organization configuration |

### Admin API -- Compliance

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/compliance/soc2` | Admin | Generate SOC 2 compliance report |
| GET | `/api/v1/compliance/gdpr` | Admin | Generate GDPR compliance report |
| GET | `/api/v1/compliance/hipaa` | Admin | Generate HIPAA compliance report |
| GET | `/api/v1/compliance/audit-export` | Admin | Export audit trail |

### Health & Monitoring

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/healthz` | No | Liveness probe -- returns 200 if server is running |
| GET | `/readyz` | No | Readiness probe -- returns 200 if database is healthy, 503 otherwise |
| GET | `/metrics` | Token | Prometheus metrics (requires `RAMPART_METRICS_TOKEN`) |

---

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

The API is organized into these endpoint groups:

| Group | Base Path | Purpose |
|-------|-----------|---------|
| **Authentication** | `/` | Registration, login, logout, token refresh, password reset, email verification |
| **User Profile** | `/me` | Authenticated user's own profile |
| **MFA** | `/mfa/` | TOTP and WebAuthn enrollment, verification, and management |
| **OAuth / OIDC** | `/oauth/` | Token issuance, authorization, consent, revocation, social login |
| **SAML** | `/saml/` | SAML SP metadata, login initiation, and assertion consumer |
| **Discovery** | `/.well-known/` | OpenID Connect discovery and JWKS |
| **SCIM** | `/scim/v2/` | SCIM 2.0 user and group provisioning |
| **Admin API** | `/api/v1/admin/` | User, organization, session, and stats management |
| **Compliance** | `/api/v1/compliance/` | SOC 2, GDPR, HIPAA reports and audit export |
| **Health** | `/healthz`, `/readyz` | Health and readiness probes |

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

### Public Endpoints

The following endpoints do not require authentication: `/register`, `/login`, `/forgot-password`, `/reset-password`, `/verify-email/send`, `/verify-email`, `/mfa/totp/verify`, `/mfa/webauthn/login/*`, `/oauth/authorize`, `/oauth/social/*`, `/saml/*`, `/.well-known/*`, `/healthz`, `/readyz`.

### Bearer Token Endpoints

Endpoints like `/me`, `/mfa/totp/enroll`, `/mfa/totp/verify-setup`, `/mfa/totp/disable`, and `/mfa/webauthn/register/*` require a valid Bearer token in the `Authorization` header.

```bash
curl -X GET https://your-rampart-instance/me \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..."
```

### Admin API

All `/api/v1/admin/` endpoints require a valid Bearer token belonging to a user with the `admin` role.

```bash
curl -X GET https://your-rampart-instance/api/v1/admin/users \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Content-Type: application/json"
```

### OAuth Endpoints

OAuth endpoints use standard OAuth 2.0 authentication mechanisms:

- **Confidential clients** authenticate via HTTP Basic (`Authorization: Basic base64(client_id:client_secret)`) or by including `client_id` and `client_secret` in the request body.
- **Public clients** (SPAs, mobile apps) use PKCE and do not send a client secret. Only the `client_id` is required.
- **Token revocation** requires client authentication.

### Client Authentication Methods

| Method | Header / Parameter | Use Case |
|--------|-------------------|----------|
| `client_secret_basic` | `Authorization: Basic base64(id:secret)` | Server-side applications (recommended) |
| `client_secret_post` | `client_id` + `client_secret` in body | When Basic auth is not practical |
| `none` | `client_id` only | Public clients (SPAs, mobile apps) with PKCE |

## Content Type

All request and response bodies use JSON (`application/json`), except for the OAuth token endpoint which accepts `application/x-www-form-urlencoded` as required by RFC 6749.

```bash
# Authentication endpoints -- JSON body
curl -X POST https://your-rampart-instance/login \
  -H "Content-Type: application/json" \
  -d '{"identifier": "jane@example.com", "password": "SecureP@ssw0rd!"}'
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
| 204 | Success with no content (DELETE, logout) |
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
| `POST /login` | 20 requests | per minute per IP |
| `POST /register` | 20 requests | per minute per IP |
| `POST /forgot-password` | 10 requests | per minute per IP |
| `POST /token/refresh` | 20 requests | per minute per IP |
| `POST /mfa/totp/verify` | 20 requests | per minute per IP |
| `POST /oauth/token` | 20 requests | per minute per IP |
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

Cross-Origin Resource Sharing (CORS) is configured via the `RAMPART_ALLOWED_ORIGINS` environment variable. Rampart uses these origins to set CORS headers on all endpoints.

```
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Accept, Authorization, Content-Type, X-Request-Id, X-Org-Context
Access-Control-Max-Age: 300
```

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

The health endpoint (`/healthz`) returns HTTP 200 when the server is running. A readiness probe is available at `/readyz` -- it returns HTTP 200 only when the server is ready to accept traffic (database connectivity verified), and HTTP 503 when the database is unhealthy.

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

### Register a new user

```bash
curl -X POST https://your-rampart-instance/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jane.doe",
    "email": "jane@example.com",
    "password": "SecureP@ssw0rd!",
    "given_name": "Jane",
    "family_name": "Doe"
  }'
```

### Log in

```bash
curl -X POST https://your-rampart-instance/login \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "jane@example.com",
    "password": "SecureP@ssw0rd!"
  }'
```

### Get current user profile

```bash
curl -X GET https://your-rampart-instance/me \
  -H "Authorization: Bearer <access_token>"
```

### Refresh an access token

```bash
curl -X POST https://your-rampart-instance/token/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

### Log out

```bash
curl -X POST https://your-rampart-instance/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

### Check server health

```bash
curl -X GET https://your-rampart-instance/healthz
```

### Fetch OIDC discovery document

```bash
curl -X GET https://your-rampart-instance/.well-known/openid-configuration
```
