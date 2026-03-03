# API Overview

Rampart exposes a RESTful API organized into three domains:

| Domain | Base Path | Purpose | Auth |
|--------|-----------|---------|------|
| **OAuth / OIDC** | `/oauth/*`, `/oidc/*`, `/.well-known/*` | Standard protocol endpoints | Per-spec |
| **Admin API** | `/api/v1/admin/*` | Manage users, clients, roles, orgs | Bearer token (admin) |
| **Account API** | `/api/v1/account/*` | Self-service for end users | Bearer token (user) |
| **System** | `/healthz`, `/readyz`, `/metrics` | Operational endpoints | None |

## Design Principles

1. **RFC-first** — OAuth 2.0 and OIDC endpoints follow their respective RFCs exactly. No custom extensions to standard flows.
2. **API-first** — Every feature is accessible via REST before any UI is built. The admin dashboard and login pages are API consumers, not special.
3. **JSON everywhere** — All request and response bodies use `application/json` unless a spec requires otherwise (e.g., `application/x-www-form-urlencoded` for token requests per RFC 6749).
4. **Consistent error format** — All errors follow a single structure.
5. **Pagination by default** — All list endpoints return paginated results.

## Versioning

- Admin and Account APIs are versioned: `/api/v1/...`
- OAuth/OIDC endpoints are **not versioned** — they follow fixed RFC paths.
- Breaking changes increment the version (`v1` → `v2`). Old versions are supported for at least 6 months after deprecation.
- Non-breaking additions (new fields, new endpoints) do not change the version.

## Authentication

### Bearer Token (Admin API)

```http
Authorization: Bearer <access_token>
```

Admin API requires a token with admin-level scopes (e.g., `rampart:admin`). Tokens are obtained via the OAuth 2.0 token endpoint.

### Bearer Token (Account API)

```http
Authorization: Bearer <access_token>
```

Account API requires a valid user access token. Users can only access their own data.

### OAuth/OIDC Endpoints

Authentication varies per endpoint as defined by the relevant RFCs:
- Token endpoint: client authentication (client_secret_basic, client_secret_post, or private_key_jwt)
- Authorize endpoint: user authentication via login session
- Introspect/Revoke: client authentication

## Error Format

All errors return a consistent JSON structure:

```json
{
  "error": "invalid_request",
  "error_description": "The 'redirect_uri' parameter is required.",
  "status": 400,
  "request_id": "req_abc123"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `error` | string | Machine-readable error code |
| `error_description` | string | Human-readable explanation |
| `status` | integer | HTTP status code |
| `request_id` | string | Unique ID for debugging/support |

For OAuth endpoints, `error` codes follow RFC 6749 Section 5.2 (e.g., `invalid_grant`, `unauthorized_client`).

For Admin/Account APIs, error codes use a `rampart_` prefix (e.g., `rampart_not_found`, `rampart_validation_error`).

## Pagination

All list endpoints support cursor-based pagination:

```http
GET /api/v1/admin/users?limit=20&cursor=eyJ0IjoiMjAyNi0wMS0wMVQwMDowMDowMFoifQ
```

Response includes pagination metadata:

```json
{
  "data": [...],
  "pagination": {
    "total": 150,
    "limit": 20,
    "has_more": true,
    "next_cursor": "eyJ0IjoiMjAyNi0wMS0wMVQwMDowMDowMFoifQ"
  }
}
```

## Rate Limiting

Rate limit headers are included in all responses:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 97
X-RateLimit-Reset: 1709500000
```

When rate limited, the server returns `429 Too Many Requests`.

## Content Types

| Endpoint | Request | Response |
|----------|---------|----------|
| OAuth token | `application/x-www-form-urlencoded` | `application/json` |
| OAuth authorize | query parameters | redirect |
| Admin API | `application/json` | `application/json` |
| Account API | `application/json` | `application/json` |
| JWKS | — | `application/json` |
| Discovery | — | `application/json` |

## Request IDs

Every response includes a `X-Request-Id` header. Include this when reporting issues or debugging.

```http
X-Request-Id: req_abc123def456
```
