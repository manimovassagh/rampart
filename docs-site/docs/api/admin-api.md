---
sidebar_position: 4
title: Admin API
description: Rampart Admin API reference -- user management, organizations, roles, OAuth clients, sessions, and audit logs with full CRUD examples.
---

# Admin API

The Admin API provides full management capabilities for Rampart. All endpoints require a Bearer token with the `admin` role or appropriate custom role permissions.

**Base path:** `/api/v1/admin/`

**Authentication:** All requests must include an `Authorization: Bearer <token>` header. Obtain a token using the [client credentials grant](./authentication.md#client-credentials-grant) with admin scopes.

**Content-Type:** All request and response bodies use `application/json`.

---

## Users

### POST /api/v1/admin/users

Create a new user account.

**Request:**

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "jane.doe",
    "email": "jane@example.com",
    "first_name": "Jane",
    "last_name": "Doe",
    "password": "SecureP@ssw0rd!",
    "enabled": true,
    "email_verified": false,
    "organization_id": "org_default",
    "roles": ["user"],
    "attributes": {
      "department": "Engineering",
      "employee_id": "EMP-1234"
    }
  }'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Unique username (3--128 characters, alphanumeric, dots, hyphens, underscores) |
| `email` | string | Yes | Valid email address, unique within the organization |
| `first_name` | string | Yes | User's first name |
| `last_name` | string | Yes | User's last name |
| `password` | string | Yes | Must meet password policy (min 8 chars, uppercase, lowercase, digit, special char) |
| `enabled` | boolean | No | Whether the user can authenticate (default: `true`) |
| `email_verified` | boolean | No | Whether the email is pre-verified (default: `false`) |
| `organization_id` | string | No | Organization to create the user in (default: `org_default`) |
| `roles` | array | No | Role names to assign (default: `["user"]`) |
| `attributes` | object | No | Custom key-value attributes for the user |

**Response (201 Created):**

```json
{
  "id": "usr_550e8400-e29b-41d4-a716-446655440000",
  "username": "jane.doe",
  "email": "jane@example.com",
  "first_name": "Jane",
  "last_name": "Doe",
  "enabled": true,
  "email_verified": false,
  "organization_id": "org_default",
  "roles": ["user"],
  "attributes": {
    "department": "Engineering",
    "employee_id": "EMP-1234"
  },
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T10:00:00Z",
  "last_login": null
}
```

**Error responses:**

- `409 Conflict` -- Username or email already exists in this organization
- `422 Validation Error` -- Invalid fields (weak password, invalid email, etc.)

### GET /api/v1/admin/users

List all users with pagination and filtering.

**Request:**

```bash
curl -X GET "https://your-rampart-instance/api/v1/admin/users?limit=20&search=jane&enabled=true&organization_id=org_default" \
  -H "Authorization: Bearer <token>"
```

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | integer | Items per page (1--100, default: 20) |
| `cursor` | string | Pagination cursor from a previous response |
| `sort` | string | Sort field: `created_at`, `username`, `email`, `last_login` (default: `created_at`) |
| `order` | string | Sort direction: `asc` or `desc` (default: `desc`) |
| `search` | string | Case-insensitive search across username, email, first name, last name |
| `enabled` | boolean | Filter by enabled/disabled status |
| `organization_id` | string | Filter by organization |
| `role` | string | Filter by role name |
| `email_verified` | boolean | Filter by email verification status |

**Response (200 OK):**

```json
{
  "data": [
    {
      "id": "usr_550e8400-e29b-41d4-a716-446655440000",
      "username": "jane.doe",
      "email": "jane@example.com",
      "first_name": "Jane",
      "last_name": "Doe",
      "enabled": true,
      "email_verified": true,
      "organization_id": "org_default",
      "roles": ["user"],
      "attributes": {},
      "created_at": "2026-03-05T10:00:00Z",
      "updated_at": "2026-03-05T10:00:00Z",
      "last_login": "2026-03-05T14:30:00Z"
    }
  ],
  "pagination": {
    "total": 142,
    "limit": 20,
    "has_more": true,
    "next_cursor": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wNVQxMDowMDowMFoiLCJpZCI6InVzcl81NTBlODQwMCJ9"
  }
}
```

### GET /api/v1/admin/users/\{user_id\}

Retrieve a single user by ID.

**Request:**

```bash
curl -X GET https://your-rampart-instance/api/v1/admin/users/usr_550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <token>"
```

**Response (200 OK):**

```json
{
  "id": "usr_550e8400-e29b-41d4-a716-446655440000",
  "username": "jane.doe",
  "email": "jane@example.com",
  "first_name": "Jane",
  "last_name": "Doe",
  "enabled": true,
  "email_verified": true,
  "organization_id": "org_default",
  "roles": ["user"],
  "attributes": {
    "department": "Engineering",
    "employee_id": "EMP-1234"
  },
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T12:00:00Z",
  "last_login": "2026-03-05T14:30:00Z"
}
```

### PUT /api/v1/admin/users/\{user_id\}

Update an existing user. Only provided fields are updated; omitted fields remain unchanged.

**Request:**

```bash
curl -X PUT https://your-rampart-instance/api/v1/admin/users/usr_550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Jane",
    "last_name": "Smith",
    "enabled": false,
    "roles": ["user", "editor"],
    "attributes": {
      "department": "Product",
      "employee_id": "EMP-1234"
    }
  }'
```

**Response (200 OK):** Returns the updated user object.

### PUT /api/v1/admin/users/\{user_id\}/password

Reset a user's password administratively.

**Request:**

```bash
curl -X PUT https://your-rampart-instance/api/v1/admin/users/usr_550e8400-e29b-41d4-a716-446655440000/password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "password": "NewSecureP@ssw0rd!",
    "temporary": true
  }'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `password` | string | Yes | The new password |
| `temporary` | boolean | No | If `true`, user must change password on next login (default: `false`) |

**Response (204 No Content)**

All active sessions for the user are revoked when a password is reset.

### DELETE /api/v1/admin/users/\{user_id\}

Delete a user and all associated data (sessions, tokens, audit events referencing this user).

**Request:**

```bash
curl -X DELETE https://your-rampart-instance/api/v1/admin/users/usr_550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <token>"
```

**Response (204 No Content)**

---

## Organizations

Organizations provide multi-tenant isolation. Each organization has its own users, roles, clients, and configuration.

### POST /api/v1/admin/organizations

Create a new organization.

**Request:**

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/organizations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "slug": "acme-corp",
    "display_name": "Acme Corp",
    "settings": {
      "theme": "corporate-blue",
      "session_ttl": "12h",
      "mfa_required": false,
      "allowed_domains": ["acme.com", "acme.co.uk"],
      "password_policy": {
        "min_length": 10,
        "require_uppercase": true,
        "require_lowercase": true,
        "require_digit": true,
        "require_special": true,
        "max_age_days": 90
      }
    }
  }'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Organization display name |
| `slug` | string | Yes | URL-safe identifier (lowercase, hyphens, 3--64 chars) |
| `display_name` | string | No | Shown on login pages (defaults to `name`) |
| `settings` | object | No | Organization-specific configuration |
| `settings.theme` | string | No | Login page theme name |
| `settings.session_ttl` | string | No | Session duration (e.g., `12h`, `7d`) |
| `settings.mfa_required` | boolean | No | Require MFA for all users |
| `settings.allowed_domains` | array | No | Restrict user email domains |
| `settings.password_policy` | object | No | Custom password policy |

**Response (201 Created):**

```json
{
  "id": "org_770e8400-e29b-41d4-a716-446655440002",
  "name": "Acme Corporation",
  "slug": "acme-corp",
  "display_name": "Acme Corp",
  "settings": {
    "theme": "corporate-blue",
    "session_ttl": "12h",
    "mfa_required": false,
    "allowed_domains": ["acme.com", "acme.co.uk"],
    "password_policy": {
      "min_length": 10,
      "require_uppercase": true,
      "require_lowercase": true,
      "require_digit": true,
      "require_special": true,
      "max_age_days": 90
    }
  },
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T10:00:00Z"
}
```

### GET /api/v1/admin/organizations

List all organizations.

**Request:**

```bash
curl -X GET "https://your-rampart-instance/api/v1/admin/organizations?limit=20&search=acme" \
  -H "Authorization: Bearer <token>"
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "id": "org_770e8400-e29b-41d4-a716-446655440002",
      "name": "Acme Corporation",
      "slug": "acme-corp",
      "display_name": "Acme Corp",
      "settings": { "theme": "corporate-blue" },
      "user_count": 47,
      "created_at": "2026-03-05T10:00:00Z",
      "updated_at": "2026-03-05T10:00:00Z"
    }
  ],
  "pagination": {
    "total": 3,
    "limit": 20,
    "has_more": false
  }
}
```

### GET /api/v1/admin/organizations/\{org_id\}

Retrieve a single organization by ID or slug.

```bash
curl -X GET https://your-rampart-instance/api/v1/admin/organizations/acme-corp \
  -H "Authorization: Bearer <token>"
```

### PUT /api/v1/admin/organizations/\{org_id\}

Update an organization. Only provided fields are updated.

```bash
curl -X PUT https://your-rampart-instance/api/v1/admin/organizations/acme-corp \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "Acme Inc.",
    "settings": {
      "theme": "midnight",
      "mfa_required": true
    }
  }'
```

**Response (200 OK):** Returns the updated organization object.

### DELETE /api/v1/admin/organizations/\{org_id\}

Delete an organization and all its associated data (users, roles, clients, sessions, events).

```bash
curl -X DELETE https://your-rampart-instance/api/v1/admin/organizations/acme-corp \
  -H "Authorization: Bearer <token>"
```

**Response (204 No Content)**

The `default` organization cannot be deleted.

---

## Roles

Roles define permissions within an organization. Rampart includes built-in roles (`admin`, `user`) that cannot be modified or deleted.

### POST /api/v1/admin/roles

Create a custom role.

**Request:**

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/roles \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "editor",
    "description": "Can view and edit content, manage own profile",
    "organization_id": "org_default",
    "permissions": [
      "users:read",
      "users:update",
      "content:read",
      "content:write",
      "content:delete",
      "profile:read",
      "profile:update"
    ]
  }'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Role name (unique within organization) |
| `description` | string | No | Human-readable description |
| `organization_id` | string | No | Organization scope (default: `org_default`) |
| `permissions` | array | Yes | List of permission strings |

**Response (201 Created):**

```json
{
  "id": "role_880e8400-e29b-41d4-a716-446655440003",
  "name": "editor",
  "description": "Can view and edit content, manage own profile",
  "organization_id": "org_default",
  "built_in": false,
  "permissions": [
    "users:read",
    "users:update",
    "content:read",
    "content:write",
    "content:delete",
    "profile:read",
    "profile:update"
  ],
  "user_count": 0,
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T10:00:00Z"
}
```

### Available Permissions

| Permission | Description |
|------------|-------------|
| `*` | Full access (admin only) |
| `users:read` | List and view users |
| `users:create` | Create new users |
| `users:update` | Update user attributes and roles |
| `users:delete` | Delete users |
| `roles:read` | List and view roles |
| `roles:create` | Create custom roles |
| `roles:update` | Modify role permissions |
| `roles:delete` | Delete custom roles |
| `organizations:read` | View organization settings |
| `organizations:create` | Create new organizations |
| `organizations:update` | Modify organization settings |
| `organizations:delete` | Delete organizations |
| `clients:read` | List and view OAuth clients |
| `clients:create` | Register new OAuth clients |
| `clients:update` | Modify client settings |
| `clients:delete` | Delete OAuth clients |
| `sessions:read` | List active sessions |
| `sessions:revoke` | Revoke user sessions |
| `events:read` | View audit log events |
| `profile:read` | Read own profile |
| `profile:update` | Update own profile |

### GET /api/v1/admin/roles

List all roles.

```bash
curl -X GET "https://your-rampart-instance/api/v1/admin/roles?organization_id=org_default" \
  -H "Authorization: Bearer <token>"
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "id": "role_001",
      "name": "admin",
      "description": "Full administrative access",
      "organization_id": "org_default",
      "built_in": true,
      "permissions": ["*"],
      "user_count": 2,
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    },
    {
      "id": "role_002",
      "name": "user",
      "description": "Standard user access",
      "organization_id": "org_default",
      "built_in": true,
      "permissions": ["profile:read", "profile:update"],
      "user_count": 140,
      "created_at": "2026-03-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    }
  ],
  "pagination": {
    "total": 2,
    "limit": 20,
    "has_more": false
  }
}
```

### GET /api/v1/admin/roles/\{role_id\}

Retrieve a single role by ID.

### PUT /api/v1/admin/roles/\{role_id\}

Update a custom role. Built-in roles cannot be updated.

```bash
curl -X PUT https://your-rampart-instance/api/v1/admin/roles/role_880e8400-e29b-41d4-a716-446655440003 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Content editor with user viewing capabilities",
    "permissions": [
      "users:read",
      "content:read",
      "content:write",
      "content:delete",
      "profile:read",
      "profile:update"
    ]
  }'
```

**Response (200 OK):** Returns the updated role object.

### DELETE /api/v1/admin/roles/\{role_id\}

Delete a custom role. Built-in roles cannot be deleted. Users who had this role will have it removed.

```bash
curl -X DELETE https://your-rampart-instance/api/v1/admin/roles/role_880e8400-e29b-41d4-a716-446655440003 \
  -H "Authorization: Bearer <token>"
```

**Response (204 No Content)**

---

## OAuth Clients

Manage OAuth 2.0 client applications registered with Rampart.

### POST /api/v1/admin/clients

Register a new OAuth client.

**Request:**

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/clients \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "my-backend-service",
    "name": "My Backend Service",
    "description": "Internal microservice for order processing",
    "type": "confidential",
    "organization_id": "org_default",
    "redirect_uris": [
      "https://api.example.com/callback"
    ],
    "web_origins": [
      "https://api.example.com"
    ],
    "grant_types": [
      "authorization_code",
      "client_credentials",
      "refresh_token"
    ],
    "scopes": ["openid", "profile", "email", "orders:read", "orders:write"],
    "token_endpoint_auth_method": "client_secret_basic",
    "access_token_ttl": 3600,
    "refresh_token_ttl": 2592000,
    "capabilities": ["token_introspection"]
  }'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `client_id` | string | Yes | Unique client identifier (3--128 chars) |
| `name` | string | Yes | Human-readable client name |
| `description` | string | No | Client description |
| `type` | string | Yes | `confidential` or `public` |
| `organization_id` | string | No | Organization scope (default: `org_default`) |
| `redirect_uris` | array | Conditional | Required for authorization code grant |
| `web_origins` | array | No | Allowed CORS origins |
| `grant_types` | array | Yes | Allowed grant types |
| `scopes` | array | Yes | Allowed scopes |
| `token_endpoint_auth_method` | string | No | `client_secret_basic`, `client_secret_post`, or `none` |
| `access_token_ttl` | integer | No | Access token lifetime in seconds (default: 3600) |
| `refresh_token_ttl` | integer | No | Refresh token lifetime in seconds (default: 2592000) |
| `capabilities` | array | No | Special capabilities (e.g., `token_introspection`) |

**Response (201 Created):**

```json
{
  "client_id": "my-backend-service",
  "client_secret": "rmp_cs_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "name": "My Backend Service",
  "description": "Internal microservice for order processing",
  "type": "confidential",
  "organization_id": "org_default",
  "redirect_uris": ["https://api.example.com/callback"],
  "web_origins": ["https://api.example.com"],
  "grant_types": ["authorization_code", "client_credentials", "refresh_token"],
  "scopes": ["openid", "profile", "email", "orders:read", "orders:write"],
  "token_endpoint_auth_method": "client_secret_basic",
  "access_token_ttl": 3600,
  "refresh_token_ttl": 2592000,
  "capabilities": ["token_introspection"],
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T10:00:00Z"
}
```

:::caution
The `client_secret` is returned only once during creation. Store it securely. If lost, you must regenerate it via `POST /api/v1/admin/clients/{client_id}/secret`.
:::

### GET /api/v1/admin/clients

List all registered clients.

```bash
curl -X GET "https://your-rampart-instance/api/v1/admin/clients?type=confidential&organization_id=org_default" \
  -H "Authorization: Bearer <token>"
```

**Response (200 OK):**

```json
{
  "data": [
    {
      "client_id": "my-backend-service",
      "name": "My Backend Service",
      "type": "confidential",
      "organization_id": "org_default",
      "grant_types": ["authorization_code", "client_credentials", "refresh_token"],
      "scopes": ["openid", "profile", "email", "orders:read", "orders:write"],
      "created_at": "2026-03-05T10:00:00Z",
      "updated_at": "2026-03-05T10:00:00Z"
    }
  ],
  "pagination": {
    "total": 5,
    "limit": 20,
    "has_more": false
  }
}
```

Note that `client_secret` is never included in list or get responses.

### GET /api/v1/admin/clients/\{client_id\}

Retrieve a single client by ID.

### PUT /api/v1/admin/clients/\{client_id\}

Update client settings. The `client_id` and `type` cannot be changed.

```bash
curl -X PUT https://your-rampart-instance/api/v1/admin/clients/my-backend-service \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Order Processing Service",
    "redirect_uris": [
      "https://api.example.com/callback",
      "https://staging.example.com/callback"
    ],
    "scopes": ["openid", "profile", "email", "orders:read", "orders:write", "inventory:read"]
  }'
```

**Response (200 OK):** Returns the updated client object (without `client_secret`).

### POST /api/v1/admin/clients/\{client_id\}/secret

Regenerate the client secret. The old secret is immediately invalidated.

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/clients/my-backend-service/secret \
  -H "Authorization: Bearer <token>"
```

**Response (200 OK):**

```json
{
  "client_id": "my-backend-service",
  "client_secret": "rmp_cs_new_secret_value_here"
}
```

### DELETE /api/v1/admin/clients/\{client_id\}

Delete a client. Existing tokens issued to this client remain valid until they expire.

```bash
curl -X DELETE https://your-rampart-instance/api/v1/admin/clients/my-backend-service \
  -H "Authorization: Bearer <token>"
```

**Response (204 No Content)**

---

## Sessions

Manage active user sessions across the Rampart instance.

### GET /api/v1/admin/sessions

List active sessions with optional filtering.

**Request:**

```bash
curl -X GET "https://your-rampart-instance/api/v1/admin/sessions?user_id=usr_550e8400&limit=20" \
  -H "Authorization: Bearer <token>"
```

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_id` | string | Filter by user ID |
| `client_id` | string | Filter by client ID |
| `organization_id` | string | Filter by organization |
| `limit` | integer | Items per page (default: 20) |
| `cursor` | string | Pagination cursor |

**Response (200 OK):**

```json
{
  "data": [
    {
      "id": "sess_990e8400-e29b-41d4-a716-446655440004",
      "user_id": "usr_550e8400-e29b-41d4-a716-446655440000",
      "username": "jane.doe",
      "client_id": "my-web-app",
      "organization_id": "org_default",
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
      "location": "San Francisco, CA, US",
      "started_at": "2026-03-05T08:00:00Z",
      "last_active_at": "2026-03-05T14:30:00Z",
      "expires_at": "2026-03-06T08:00:00Z"
    }
  ],
  "pagination": {
    "total": 23,
    "limit": 20,
    "has_more": true,
    "next_cursor": "eyJzdGFydGVkX2F0IjoiMjAyNi0wMy0wNVQwODowMDowMFoifQ=="
  }
}
```

### DELETE /api/v1/admin/sessions/\{session_id\}

Revoke a specific session. The session's tokens are immediately invalidated.

```bash
curl -X DELETE https://your-rampart-instance/api/v1/admin/sessions/sess_990e8400-e29b-41d4-a716-446655440004 \
  -H "Authorization: Bearer <token>"
```

**Response (204 No Content)**

### DELETE /api/v1/admin/users/\{user_id\}/sessions

Revoke all sessions for a specific user. All of the user's tokens are immediately invalidated.

```bash
curl -X DELETE https://your-rampart-instance/api/v1/admin/users/usr_550e8400-e29b-41d4-a716-446655440000/sessions \
  -H "Authorization: Bearer <token>"
```

**Response (200 OK):**

```json
{
  "revoked_count": 3,
  "message": "All sessions for user usr_550e8400-e29b-41d4-a716-446655440000 have been revoked."
}
```

### DELETE /api/v1/admin/sessions

Bulk revoke sessions by filter criteria. At least one filter parameter is required to prevent accidental mass revocation.

```bash
# Revoke all sessions for a specific organization
curl -X DELETE "https://your-rampart-instance/api/v1/admin/sessions?organization_id=org_acme" \
  -H "Authorization: Bearer <token>"

# Revoke all sessions for a specific client
curl -X DELETE "https://your-rampart-instance/api/v1/admin/sessions?client_id=compromised-app" \
  -H "Authorization: Bearer <token>"
```

**Response (200 OK):**

```json
{
  "revoked_count": 47,
  "message": "47 sessions have been revoked."
}
```

---

## Audit Events

Query the audit log for security-relevant events. Audit events are immutable and cannot be modified or deleted via the API.

### GET /api/v1/admin/events

List audit events with filtering.

**Request:**

```bash
curl -X GET "https://your-rampart-instance/api/v1/admin/events?type=user.login_failed&from=2026-03-01T00:00:00Z&to=2026-03-05T23:59:59Z&limit=50" \
  -H "Authorization: Bearer <token>"
```

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | Filter by event type (see table below) |
| `actor_id` | string | Filter by the user/client who triggered the event |
| `target_id` | string | Filter by the affected resource ID |
| `ip_address` | string | Filter by IP address |
| `organization_id` | string | Filter by organization |
| `from` | ISO 8601 | Start of date range (inclusive) |
| `to` | ISO 8601 | End of date range (inclusive) |
| `limit` | integer | Items per page (default: 20, max: 100) |
| `cursor` | string | Pagination cursor |
| `order` | string | `asc` or `desc` (default: `desc`) |

**Response (200 OK):**

```json
{
  "data": [
    {
      "id": "evt_aab8400-e29b-41d4-a716-446655440005",
      "type": "user.login_failed",
      "actor_id": null,
      "actor_email": null,
      "target_id": "usr_550e8400-e29b-41d4-a716-446655440000",
      "target_type": "user",
      "organization_id": "org_default",
      "ip_address": "203.0.113.42",
      "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
      "timestamp": "2026-03-05T14:30:00Z",
      "metadata": {
        "username_attempted": "jane.doe",
        "failure_reason": "invalid_password",
        "attempt_count": 3
      }
    },
    {
      "id": "evt_bbc8400-e29b-41d4-a716-446655440006",
      "type": "user.login",
      "actor_id": "usr_550e8400-e29b-41d4-a716-446655440000",
      "actor_email": "jane@example.com",
      "target_id": "usr_550e8400-e29b-41d4-a716-446655440000",
      "target_type": "user",
      "organization_id": "org_default",
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
      "timestamp": "2026-03-05T14:35:00Z",
      "metadata": {
        "client_id": "my-web-app",
        "grant_type": "authorization_code",
        "session_id": "sess_990e8400"
      }
    }
  ],
  "pagination": {
    "total": 1847,
    "limit": 50,
    "has_more": true,
    "next_cursor": "eyJ0aW1lc3RhbXAiOiIyMDI2LTAzLTA1VDE0OjM1OjAwWiJ9"
  }
}
```

### GET /api/v1/admin/events/\{event_id\}

Retrieve a single audit event by ID.

```bash
curl -X GET https://your-rampart-instance/api/v1/admin/events/evt_aab8400-e29b-41d4-a716-446655440005 \
  -H "Authorization: Bearer <token>"
```

### Event Types

| Event Type | Description |
|------------|-------------|
| `user.created` | New user account created |
| `user.updated` | User attributes updated |
| `user.deleted` | User account deleted |
| `user.enabled` | User account enabled |
| `user.disabled` | User account disabled |
| `user.password_changed` | User changed their password |
| `user.password_reset` | Admin reset a user's password |
| `user.login` | Successful user login |
| `user.login_failed` | Failed login attempt |
| `user.logout` | User logged out |
| `user.mfa_enabled` | MFA was enabled for a user |
| `user.mfa_disabled` | MFA was disabled for a user |
| `token.issued` | Token issued via any grant type |
| `token.refreshed` | Token refreshed using refresh token |
| `token.revoked` | Token explicitly revoked |
| `token.introspected` | Token introspection performed |
| `session.created` | New session started |
| `session.revoked` | Session explicitly revoked |
| `session.expired` | Session expired naturally |
| `client.created` | OAuth client registered |
| `client.updated` | OAuth client settings changed |
| `client.deleted` | OAuth client deleted |
| `client.secret_regenerated` | Client secret was regenerated |
| `role.created` | Custom role created |
| `role.updated` | Role permissions changed |
| `role.deleted` | Custom role deleted |
| `organization.created` | Organization created |
| `organization.updated` | Organization settings changed |
| `organization.deleted` | Organization deleted |
| `admin.settings_changed` | Server-level settings modified |

---

## Common Admin API Patterns

### Disabling a User and Revoking All Sessions

```bash
# Step 1: Disable the user
curl -X PUT https://your-rampart-instance/api/v1/admin/users/usr_550e8400 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Step 2: Revoke all their sessions
curl -X DELETE https://your-rampart-instance/api/v1/admin/users/usr_550e8400/sessions \
  -H "Authorization: Bearer <token>"
```

### Investigating Suspicious Activity

```bash
# Find failed login attempts from a specific IP
curl -X GET "https://your-rampart-instance/api/v1/admin/events?type=user.login_failed&ip_address=203.0.113.42&from=2026-03-04T00:00:00Z&limit=100" \
  -H "Authorization: Bearer <token>"
```

### Setting Up a New Organization

```bash
# 1. Create the organization
curl -X POST https://your-rampart-instance/api/v1/admin/organizations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Acme Corp", "slug": "acme-corp", "settings": {"theme": "corporate-blue"}}'

# 2. Create a custom role
curl -X POST https://your-rampart-instance/api/v1/admin/roles \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "org-admin", "organization_id": "org_acme-corp", "permissions": ["users:read", "users:create", "users:update", "sessions:read", "sessions:revoke"]}'

# 3. Create the first admin user
curl -X POST https://your-rampart-instance/api/v1/admin/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"username": "acme-admin", "email": "admin@acme.com", "first_name": "Admin", "last_name": "User", "password": "SecureP@ss!", "organization_id": "org_acme-corp", "roles": ["org-admin"]}'

# 4. Register an OAuth client for the organization
curl -X POST https://your-rampart-instance/api/v1/admin/clients \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"client_id": "acme-web-app", "name": "Acme Web App", "type": "public", "organization_id": "org_acme-corp", "redirect_uris": ["https://app.acme.com/callback"], "grant_types": ["authorization_code", "refresh_token"], "scopes": ["openid", "profile", "email"]}'
```
