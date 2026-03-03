# Admin API

The Admin API provides full management of Rampart resources. All endpoints require a bearer token with `rampart:admin` scope.

```
Base: /api/v1/admin
Auth: Bearer <admin_access_token>
```

---

## Organizations

Multi-tenancy support. Each organization is an isolated realm with its own users, clients, and configuration.

### List Organizations

```http
GET /api/v1/admin/organizations?limit=20&cursor=...
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "org_abc123",
      "name": "Acme Corp",
      "slug": "acme",
      "display_name": "Acme Corporation",
      "enabled": true,
      "created_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-01-15T10:00:00Z"
    }
  ],
  "pagination": {
    "total": 3,
    "limit": 20,
    "has_more": false,
    "next_cursor": null
  }
}
```

### Create Organization

```http
POST /api/v1/admin/organizations
Content-Type: application/json

{
  "name": "Acme Corp",
  "slug": "acme",
  "display_name": "Acme Corporation"
}
```

**Response** `201 Created`

### Get Organization

```http
GET /api/v1/admin/organizations/{org_id}
```

**Response** `200 OK`

### Update Organization

```http
PUT /api/v1/admin/organizations/{org_id}
Content-Type: application/json

{
  "display_name": "Acme Corp International",
  "enabled": true
}
```

**Response** `200 OK`

### Delete Organization

```http
DELETE /api/v1/admin/organizations/{org_id}
```

**Response** `204 No Content`

### Update Organization Settings

```http
PUT /api/v1/admin/organizations/{org_id}/settings
Content-Type: application/json

{
  "session_lifetime": 3600,
  "mfa_policy": "optional",
  "password_policy": {
    "min_length": 12,
    "require_uppercase": true,
    "require_number": true,
    "require_special": true
  },
  "allowed_redirect_domains": ["*.acme.com"]
}
```

**Response** `200 OK`

### Update Organization Branding

```http
PUT /api/v1/admin/organizations/{org_id}/branding
Content-Type: application/json

{
  "logo_url": "https://acme.com/logo.png",
  "primary_color": "#8B5CF6",
  "background_color": "#1e1e2e",
  "custom_css": ""
}
```

**Response** `200 OK`

---

## Users

### List Users

```http
GET /api/v1/admin/users?limit=20&cursor=...&search=jane&enabled=true
```

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | integer | Max results (default 20, max 100) |
| `cursor` | string | Pagination cursor |
| `search` | string | Search by name, email, or username |
| `enabled` | boolean | Filter by enabled status |
| `org_id` | string | Filter by organization |

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "usr_abc123",
      "username": "jane.doe",
      "email": "jane@example.com",
      "email_verified": true,
      "given_name": "Jane",
      "family_name": "Doe",
      "enabled": true,
      "mfa_enabled": false,
      "org_id": "org_abc123",
      "created_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-01-15T10:00:00Z",
      "last_login_at": "2026-03-01T14:30:00Z"
    }
  ],
  "pagination": { "total": 42, "limit": 20, "has_more": true, "next_cursor": "..." }
}
```

### Create User

```http
POST /api/v1/admin/users
Content-Type: application/json

{
  "username": "jane.doe",
  "email": "jane@example.com",
  "given_name": "Jane",
  "family_name": "Doe",
  "password": "initial-password-123",
  "email_verified": false,
  "enabled": true,
  "org_id": "org_abc123"
}
```

**Response** `201 Created`

### Get User

```http
GET /api/v1/admin/users/{user_id}
```

**Response** `200 OK`

### Update User

```http
PUT /api/v1/admin/users/{user_id}
Content-Type: application/json

{
  "given_name": "Jane",
  "family_name": "Smith",
  "enabled": true
}
```

**Response** `200 OK`

### Delete User

```http
DELETE /api/v1/admin/users/{user_id}
```

**Response** `204 No Content`

### Reset User Password

```http
POST /api/v1/admin/users/{user_id}/reset-password
Content-Type: application/json

{
  "password": "new-password-456",
  "temporary": true
}
```

When `temporary` is `true`, the user must change their password on next login.

**Response** `204 No Content`

### Disable User

```http
POST /api/v1/admin/users/{user_id}/disable
```

**Response** `204 No Content`

### Enable User

```http
POST /api/v1/admin/users/{user_id}/enable
```

**Response** `204 No Content`

### List User Sessions

```http
GET /api/v1/admin/users/{user_id}/sessions
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "ses_abc123",
      "ip_address": "203.0.113.42",
      "user_agent": "Mozilla/5.0...",
      "started_at": "2026-03-01T14:30:00Z",
      "last_active_at": "2026-03-03T09:15:00Z",
      "expires_at": "2026-03-03T14:30:00Z"
    }
  ]
}
```

### Revoke User Sessions

```http
DELETE /api/v1/admin/users/{user_id}/sessions
```

Revokes all sessions for the user.

**Response** `204 No Content`

### List User Role Assignments

```http
GET /api/v1/admin/users/{user_id}/roles
```

**Response** `200 OK`

### Assign Roles to User

```http
POST /api/v1/admin/users/{user_id}/roles
Content-Type: application/json

{
  "role_ids": ["role_admin", "role_editor"]
}
```

**Response** `204 No Content`

### Remove Roles from User

```http
DELETE /api/v1/admin/users/{user_id}/roles
Content-Type: application/json

{
  "role_ids": ["role_editor"]
}
```

**Response** `204 No Content`

---

## Clients (OAuth Applications)

### List Clients

```http
GET /api/v1/admin/clients?limit=20&cursor=...
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "cli_abc123",
      "client_id": "my-spa",
      "name": "My Single Page App",
      "type": "public",
      "grant_types": ["authorization_code", "refresh_token"],
      "redirect_uris": ["https://app.example.com/callback"],
      "allowed_scopes": ["openid", "profile", "email"],
      "enabled": true,
      "org_id": "org_abc123",
      "created_at": "2026-01-15T10:00:00Z"
    }
  ],
  "pagination": { "total": 5, "limit": 20, "has_more": false, "next_cursor": null }
}
```

### Create Client

```http
POST /api/v1/admin/clients
Content-Type: application/json

{
  "client_id": "my-spa",
  "name": "My Single Page App",
  "type": "public",
  "grant_types": ["authorization_code", "refresh_token"],
  "redirect_uris": ["https://app.example.com/callback"],
  "post_logout_redirect_uris": ["https://app.example.com"],
  "allowed_scopes": ["openid", "profile", "email"],
  "token_endpoint_auth_method": "none",
  "org_id": "org_abc123"
}
```

| Field | Values |
|-------|--------|
| `type` | `public` (SPA, mobile, CLI) or `confidential` (server-side) |
| `grant_types` | `authorization_code`, `client_credentials`, `refresh_token`, `urn:ietf:params:oauth:grant-type:device_code` |
| `token_endpoint_auth_method` | `none`, `client_secret_basic`, `client_secret_post`, `private_key_jwt` |

**Response** `201 Created` — includes `client_secret` for confidential clients (only returned once).

### Get Client

```http
GET /api/v1/admin/clients/{client_id}
```

### Update Client

```http
PUT /api/v1/admin/clients/{client_id}
Content-Type: application/json

{
  "name": "My Updated App",
  "redirect_uris": ["https://app.example.com/callback", "https://staging.example.com/callback"]
}
```

### Delete Client

```http
DELETE /api/v1/admin/clients/{client_id}
```

**Response** `204 No Content`

### Rotate Client Secret

```http
POST /api/v1/admin/clients/{client_id}/rotate-secret
```

Generates a new client secret. The old secret remains valid for a configurable grace period (default: 24 hours).

**Response** `200 OK`

```json
{
  "client_secret": "new-secret-value",
  "previous_secret_expires_at": "2026-03-04T10:00:00Z"
}
```

---

## Roles

### List Roles

```http
GET /api/v1/admin/roles?limit=20&cursor=...
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "role_admin",
      "name": "admin",
      "display_name": "Administrator",
      "description": "Full access to all resources",
      "org_id": "org_abc123",
      "created_at": "2026-01-15T10:00:00Z"
    }
  ],
  "pagination": { "total": 4, "limit": 20, "has_more": false, "next_cursor": null }
}
```

### Create Role

```http
POST /api/v1/admin/roles
Content-Type: application/json

{
  "name": "editor",
  "display_name": "Editor",
  "description": "Can edit content but not manage users",
  "org_id": "org_abc123"
}
```

**Response** `201 Created`

### Get Role

```http
GET /api/v1/admin/roles/{role_id}
```

### Update Role

```http
PUT /api/v1/admin/roles/{role_id}
Content-Type: application/json

{
  "display_name": "Content Editor",
  "description": "Can create and edit content"
}
```

### Delete Role

```http
DELETE /api/v1/admin/roles/{role_id}
```

**Response** `204 No Content`

### List Role Permissions

```http
GET /api/v1/admin/roles/{role_id}/permissions
```

### Assign Permissions to Role

```http
POST /api/v1/admin/roles/{role_id}/permissions
Content-Type: application/json

{
  "permission_ids": ["perm_users_read", "perm_content_write"]
}
```

**Response** `204 No Content`

### Remove Permissions from Role

```http
DELETE /api/v1/admin/roles/{role_id}/permissions
Content-Type: application/json

{
  "permission_ids": ["perm_content_write"]
}
```

**Response** `204 No Content`

### List Users with Role

```http
GET /api/v1/admin/roles/{role_id}/users
```

---

## Groups

### List Groups

```http
GET /api/v1/admin/groups?limit=20&cursor=...
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "grp_abc123",
      "name": "engineering",
      "display_name": "Engineering Team",
      "org_id": "org_abc123",
      "member_count": 15,
      "created_at": "2026-01-15T10:00:00Z"
    }
  ],
  "pagination": { "total": 3, "limit": 20, "has_more": false, "next_cursor": null }
}
```

### Create Group

```http
POST /api/v1/admin/groups
Content-Type: application/json

{
  "name": "engineering",
  "display_name": "Engineering Team",
  "org_id": "org_abc123"
}
```

**Response** `201 Created`

### Get Group

```http
GET /api/v1/admin/groups/{group_id}
```

### Update Group

```http
PUT /api/v1/admin/groups/{group_id}
```

### Delete Group

```http
DELETE /api/v1/admin/groups/{group_id}
```

**Response** `204 No Content`

### List Group Members

```http
GET /api/v1/admin/groups/{group_id}/members
```

### Add Members to Group

```http
POST /api/v1/admin/groups/{group_id}/members
Content-Type: application/json

{
  "user_ids": ["usr_abc123", "usr_def456"]
}
```

**Response** `204 No Content`

### Remove Members from Group

```http
DELETE /api/v1/admin/groups/{group_id}/members
Content-Type: application/json

{
  "user_ids": ["usr_abc123"]
}
```

**Response** `204 No Content`

### List Group Role Assignments

```http
GET /api/v1/admin/groups/{group_id}/roles
```

### Assign Roles to Group

```http
POST /api/v1/admin/groups/{group_id}/roles
Content-Type: application/json

{
  "role_ids": ["role_editor"]
}
```

**Response** `204 No Content`

---

## Identity Providers

Manage external identity providers for social login and enterprise SSO.

### List Identity Providers

```http
GET /api/v1/admin/identity-providers?org_id=org_abc123
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "idp_google",
      "type": "oidc",
      "name": "Google",
      "enabled": true,
      "config": {
        "issuer": "https://accounts.google.com",
        "client_id": "google-client-id",
        "scopes": ["openid", "profile", "email"]
      },
      "org_id": "org_abc123",
      "created_at": "2026-01-15T10:00:00Z"
    }
  ]
}
```

### Create Identity Provider

```http
POST /api/v1/admin/identity-providers
Content-Type: application/json

{
  "type": "oidc",
  "name": "Google",
  "enabled": true,
  "config": {
    "issuer": "https://accounts.google.com",
    "client_id": "google-client-id",
    "client_secret": "google-client-secret",
    "scopes": ["openid", "profile", "email"]
  },
  "attribute_mapping": {
    "email": "email",
    "given_name": "given_name",
    "family_name": "family_name",
    "picture": "picture"
  },
  "org_id": "org_abc123"
}
```

| Type | Description |
|------|-------------|
| `oidc` | Generic OpenID Connect provider |
| `saml` | SAML 2.0 identity provider |
| `google` | Google (pre-configured OIDC) |
| `github` | GitHub OAuth |
| `apple` | Apple Sign In |
| `microsoft` | Microsoft Entra ID |

**Response** `201 Created`

### Get Identity Provider

```http
GET /api/v1/admin/identity-providers/{idp_id}
```

### Update Identity Provider

```http
PUT /api/v1/admin/identity-providers/{idp_id}
```

### Delete Identity Provider

```http
DELETE /api/v1/admin/identity-providers/{idp_id}
```

**Response** `204 No Content`

---

## Sessions

### List Active Sessions

```http
GET /api/v1/admin/sessions?limit=20&cursor=...&user_id=usr_abc123
```

**Query Parameters**

| Parameter | Description |
|-----------|-------------|
| `user_id` | Filter by user |
| `client_id` | Filter by client |
| `ip_address` | Filter by IP |

**Response** `200 OK`

### Revoke Session

```http
DELETE /api/v1/admin/sessions/{session_id}
```

**Response** `204 No Content`

### Revoke All Sessions

```http
POST /api/v1/admin/sessions/revoke-all
Content-Type: application/json

{
  "user_id": "usr_abc123"
}
```

**Response** `204 No Content`

---

## Audit Events

Query the audit log for security and compliance purposes.

### List Events

```http
GET /api/v1/admin/events?limit=50&cursor=...
```

**Query Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | Event type (e.g., `user.login`, `user.login_failed`, `client.created`) |
| `user_id` | string | Filter by user |
| `ip_address` | string | Filter by IP |
| `from` | datetime | Start of time range (ISO 8601) |
| `to` | datetime | End of time range (ISO 8601) |
| `org_id` | string | Filter by organization |

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "evt_abc123",
      "type": "user.login",
      "user_id": "usr_abc123",
      "ip_address": "203.0.113.42",
      "user_agent": "Mozilla/5.0...",
      "details": {
        "client_id": "my-spa",
        "method": "password"
      },
      "org_id": "org_abc123",
      "created_at": "2026-03-03T10:00:00Z"
    }
  ],
  "pagination": { "total": 1250, "limit": 50, "has_more": true, "next_cursor": "..." }
}
```

### Event Types

| Category | Events |
|----------|--------|
| **Authentication** | `user.login`, `user.login_failed`, `user.logout`, `user.mfa_challenge`, `user.mfa_success`, `user.mfa_failed` |
| **User Management** | `user.created`, `user.updated`, `user.deleted`, `user.disabled`, `user.enabled`, `user.password_reset` |
| **Client Management** | `client.created`, `client.updated`, `client.deleted`, `client.secret_rotated` |
| **Role & Permission** | `role.created`, `role.updated`, `role.deleted`, `role.assigned`, `role.unassigned` |
| **Organization** | `org.created`, `org.updated`, `org.deleted` |
| **Token** | `token.issued`, `token.refreshed`, `token.revoked`, `token.introspected` |
| **Session** | `session.created`, `session.revoked` |
