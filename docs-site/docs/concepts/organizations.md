---
sidebar_position: 3
title: Organizations
description: Multi-tenancy in Rampart — organization isolation, resource scoping, default organizations, and cross-organization scenarios.
---

# Organizations

Rampart is multi-tenant by default. Organizations are the top-level isolation boundary for all resources — users, roles, clients, sessions, settings, and audit events are scoped to an organization. This model supports SaaS platforms, enterprise departments, partner ecosystems, and any scenario that requires tenant separation within a single Rampart deployment.

## Multi-Tenancy Model

Every resource in Rampart belongs to an organization. There is no global scope for end users — all operations happen within an organization context.

```
Rampart Instance
  |
  +-- Organization: "acme-corp"
  |     +-- Users (scoped)
  |     +-- Roles (scoped)
  |     +-- OAuth Clients (scoped)
  |     +-- Sessions (scoped)
  |     +-- Settings (scoped)
  |     +-- Audit Events (scoped)
  |     +-- Login Theme (scoped)
  |     +-- Identity Providers (scoped)
  |
  +-- Organization: "globex-inc"
  |     +-- Users (scoped)
  |     +-- Roles (scoped)
  |     +-- ...
  |
  +-- Organization: "default"
        +-- Users (scoped)
        +-- ...
```

### Organization Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | UUID | Unique identifier |
| `slug` | string | URL-safe identifier (e.g., `acme-corp`), unique across the instance |
| `name` | string | Display name |
| `domain` | string | Optional domain for automatic org resolution (e.g., `acme.com`) |
| `login_theme` | string | Theme identifier for the login/consent UI |
| `mfa_policy` | enum | MFA enforcement level (`optional`, `encouraged`, `required`, `required_for_admins`) |
| `password_policy` | object | Password complexity rules |
| `session_policy` | object | Session limits, idle timeout, absolute timeout |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last modification time |
| `enabled` | boolean | Whether the organization is active |

### Organization Resolution

When a request arrives, Rampart determines the target organization through (in priority order):

1. **Explicit header** — `X-Rampart-Org: acme-corp`
2. **Subdomain** — `acme-corp.auth.example.com` maps to the `acme-corp` organization
3. **Domain mapping** — the request's `Host` header is matched against configured organization domains
4. **Query parameter** — `?org=acme-corp` (for authorization endpoints)
5. **Default** — falls back to the `default` organization

## Organization Isolation

Organizations provide hard isolation. Resources in one organization are invisible to users and clients in another organization.

### What Is Isolated

| Resource | Isolation Behavior |
|----------|-------------------|
| **Users** | A user belongs to exactly one organization. The same email can exist in different organizations as separate accounts. |
| **Roles** | Each organization has its own role definitions. Built-in roles exist in every org but can be customized independently. |
| **OAuth Clients** | Clients are registered per organization. A client in `acme-corp` cannot be used to authenticate users in `globex-inc`. |
| **Sessions** | Sessions are scoped to the organization. Revoking all sessions in one org does not affect another. |
| **Identity Providers** | Social login and OIDC federation configs are per organization. Acme might use Google login; Globex might use Azure AD. |
| **Login Theme** | Each organization can select its own login page theme. |
| **Password Policy** | Password complexity rules are configured per organization. |
| **MFA Policy** | MFA enforcement level is set per organization. |
| **Audit Events** | Events are tagged with the organization and can only be queried within that org's context. |
| **Settings** | Token lifetimes, session policies, and other configuration are per organization. |

### Database Isolation

Organization isolation is enforced at the database level through a mandatory `org_id` column on all tenant-scoped tables. Every query includes the organization filter — there is no code path that can accidentally return cross-org data.

```sql
-- Every query is scoped
SELECT * FROM users WHERE org_id = $1 AND email = $2;
SELECT * FROM roles WHERE org_id = $1;
SELECT * FROM oauth_clients WHERE org_id = $1 AND client_id = $2;
```

Row-level security policies provide a defense-in-depth layer, ensuring that even raw database access cannot bypass organization boundaries.

## Default Organization

Every Rampart instance has a `default` organization that is created automatically during initial setup.

**Characteristics of the default organization:**

- **Slug:** `default`
- **Cannot be deleted** — it serves as the fallback organization
- **First user is super admin** — the first user registered in the default organization is automatically assigned the `super_admin` role
- **Fallback for unresolved requests** — when no organization can be determined from the request, the default org is used

**When to use the default organization:**

- Single-tenant deployments that do not need multi-tenancy
- Development and testing environments
- The initial admin's home organization before creating tenant-specific orgs

For single-tenant use cases, you can use only the default organization and ignore multi-tenancy entirely. Rampart works identically in both modes.

## Managing Organizations

### Creating an Organization

Organizations are created through the Admin API or the admin console.

**API:**

```bash
curl -X POST https://auth.example.com/api/admin/organizations \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "slug": "acme-corp",
    "name": "Acme Corporation",
    "domain": "acme.com",
    "login_theme": "corporate-blue",
    "mfa_policy": "encouraged"
  }'
```

Only users with the `super_admin` role can create organizations.

### Updating an Organization

```bash
curl -X PUT https://auth.example.com/api/admin/organizations/acme-corp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp International",
    "mfa_policy": "required"
  }'
```

Organization admins (`org_admin`) can update settings for their own organization. Super admins can update any organization.

### Listing Organizations

```bash
curl https://auth.example.com/api/admin/organizations \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Getting a Specific Organization

```bash
curl https://auth.example.com/api/admin/organizations/acme-corp \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Disabling an Organization

Disabling an organization prevents all authentication for that org without deleting data:

```bash
curl -X PUT https://auth.example.com/api/admin/organizations/acme-corp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ "enabled": false }'
```

When an organization is disabled:

- All login attempts are rejected
- Existing sessions remain but new token refreshes fail
- The admin console shows the org as disabled
- Re-enabling restores normal operation immediately

### Deleting an Organization

```bash
curl -X DELETE https://auth.example.com/api/admin/organizations/acme-corp \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Deletion is a destructive operation restricted to `super_admin`. It permanently removes the organization and all associated data (users, roles, clients, sessions, events). The `default` organization cannot be deleted.

## Organization-Level Settings

Each organization can customize its behavior through structured settings:

```json
{
  "login_theme": "midnight",
  "password_policy": {
    "min_length": 12,
    "max_length": 128,
    "require_uppercase": true,
    "require_lowercase": true,
    "require_digit": true,
    "require_special": true,
    "reject_common": true,
    "history_count": 5
  },
  "session_policy": {
    "absolute_timeout": "720h",
    "idle_timeout": "168h",
    "max_per_user": 5,
    "on_limit_exceeded": "revoke_oldest"
  },
  "mfa_policy": "required_for_admins",
  "token_lifetimes": {
    "access_token_ttl": "1h",
    "refresh_token_ttl": "7d",
    "authorization_code_ttl": "10m"
  },
  "allowed_email_domains": ["acme.com", "acme.org"],
  "self_registration": true,
  "require_email_verification": true
}
```

Settings can be updated via the Admin API:

```bash
curl -X PUT https://auth.example.com/api/admin/organizations/acme-corp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "settings": {
      "login_theme": "midnight",
      "self_registration": false
    }
  }'
```

## Cross-Organization Scenarios

While organizations are isolated by default, certain administrative scenarios require cross-org visibility.

### Super Admin Access

Users with the `super_admin` role (in the `default` organization) have cross-org privileges:

- List all organizations
- Create, update, disable, and delete organizations
- View users and audit events across all organizations
- Assign the `org_admin` role to users in any organization
- Access system-wide health and metrics

Super admins operate at the instance level, above the organization boundary.

### User Migration Between Organizations

Users can be migrated between organizations through the Admin API:

```bash
curl -X POST https://auth.example.com/api/admin/users/{user_id}/migrate \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_org": "globex-inc",
    "copy_roles": false
  }'
```

Migration:

- Moves the user record to the target organization
- Revokes all existing sessions and tokens
- Optionally copies role assignments (if matching roles exist in the target org)
- Emits `user.migrated` audit events in both the source and target organizations
- Does not transfer OAuth client authorizations or consent grants

### Shared Identity Providers

In some deployments, multiple organizations may share the same external identity provider (e.g., a company-wide Azure AD tenant). Rampart supports this by allowing identity provider configurations to reference a shared provider definition, while each organization maintains its own user mapping and claim configuration.

## Organization Limits and Quotas

To prevent resource exhaustion, Rampart supports per-organization limits:

| Limit | Default | Description |
|-------|---------|-------------|
| `max_users` | Unlimited | Maximum number of users |
| `max_clients` | 100 | Maximum number of OAuth clients |
| `max_sessions_per_user` | 10 | Maximum concurrent sessions per user |
| `max_roles` | 50 | Maximum number of custom roles |

Limits are configured per organization by a super admin. When a limit is reached, creation requests return `429 Too Many Requests` with a descriptive error message.
