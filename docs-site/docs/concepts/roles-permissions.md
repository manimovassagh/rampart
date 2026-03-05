---
sidebar_position: 4
title: Roles and Permissions
description: Role-Based Access Control (RBAC) in Rampart — built-in roles, custom roles, permissions, role assignment, and inheritance.
---

# Roles and Permissions

Rampart uses Role-Based Access Control (RBAC) to manage authorization. Every user is assigned one or more roles within their organization, and each role grants a set of permissions. Roles determine what a user can see and do in both the API and the admin console.

## RBAC Model

Rampart's authorization model follows a straightforward hierarchy:

```
User  --(has)-->  Role(s)  --(grants)-->  Permission(s)
```

- A **user** belongs to one organization and has one or more roles
- A **role** is a named collection of permissions, scoped to an organization
- A **permission** is a granular authorization to perform a specific action on a specific resource type

**Authorization evaluation:** When a user makes a request, Rampart collects all permissions from all of the user's assigned roles. If any role grants the required permission, access is allowed. Permissions are additive — there are no "deny" rules.

## Built-in Roles

Every organization is created with three built-in roles. These roles cannot be deleted but their permissions can be extended (not reduced) through custom permissions.

### super_admin

The highest privilege level, available only in the `default` organization.

| Aspect | Detail |
|--------|--------|
| **Scope** | Instance-wide (crosses organization boundaries) |
| **Assignment** | Automatically assigned to the first registered user; manually assigned thereafter |
| **Purpose** | Manage organizations, system configuration, and global operations |

**Permissions:**

- All permissions from `org_admin`
- `organizations:create`, `organizations:update`, `organizations:delete`, `organizations:list`
- `system:configure` — modify instance-level settings
- `system:metrics` — access system health and performance metrics
- `users:migrate` — move users between organizations
- `audit:read_global` — read audit events across all organizations

### org_admin

The administrator role for a specific organization.

| Aspect | Detail |
|--------|--------|
| **Scope** | Single organization |
| **Assignment** | Assigned by a `super_admin` or another `org_admin` |
| **Purpose** | Manage users, roles, clients, and settings within the organization |

**Permissions:**

- All permissions from `user`
- `users:create`, `users:read`, `users:update`, `users:delete`, `users:list`
- `roles:create`, `roles:read`, `roles:update`, `roles:delete`, `roles:assign`
- `clients:create`, `clients:read`, `clients:update`, `clients:delete`
- `sessions:read`, `sessions:revoke`, `sessions:revoke_all`
- `audit:read` — read audit events within the organization
- `org:update` — modify organization settings (theme, policies, identity providers)
- `idp:create`, `idp:update`, `idp:delete` — manage identity provider configurations

### user

The default role assigned to all newly registered users.

| Aspect | Detail |
|--------|--------|
| **Scope** | Own account only |
| **Assignment** | Automatic on registration |
| **Purpose** | Standard end-user access |

**Permissions:**

- `account:read` — view own profile
- `account:update` — update own profile (name, email, password)
- `account:mfa` — manage own MFA enrollment
- `account:sessions` — view and revoke own sessions
- `account:delete` — delete own account (if self-deletion is enabled for the org)

## Custom Roles

Organizations can create custom roles to model their specific authorization requirements.

### Creating a Custom Role

**Via API:**

```bash
curl -X POST https://auth.example.com/api/admin/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "support_agent",
    "display_name": "Support Agent",
    "description": "Can view and update users but cannot delete or manage roles",
    "permissions": [
      "users:read",
      "users:list",
      "users:update",
      "sessions:read",
      "sessions:revoke",
      "audit:read"
    ]
  }'
```

**Via admin console:** Navigate to the organization's Roles tab, click "Create Role", and select permissions from the checklist.

### Custom Role Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | UUID | Unique identifier |
| `name` | string | Machine-readable name (unique within the org, e.g., `support_agent`) |
| `display_name` | string | Human-readable name (e.g., "Support Agent") |
| `description` | string | Optional description |
| `permissions` | string[] | List of granted permission identifiers |
| `org_id` | UUID | The organization this role belongs to |
| `built_in` | boolean | Whether this is a built-in role (read-only) |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last modification time |

### Example Custom Roles

| Role | Use Case | Key Permissions |
|------|----------|-----------------|
| `readonly_admin` | Auditors who need visibility without write access | `users:read`, `users:list`, `audit:read`, `sessions:read`, `clients:read` |
| `support_agent` | Support staff who manage users | `users:read`, `users:list`, `users:update`, `sessions:read`, `sessions:revoke` |
| `client_manager` | Developers who manage OAuth clients | `clients:create`, `clients:read`, `clients:update`, `clients:delete` |
| `security_officer` | Security team reviewing auth events | `audit:read`, `sessions:read`, `sessions:revoke_all`, `users:read` |

## Permission Model

Permissions follow a `resource:action` naming convention. This makes them predictable and easy to reason about.

### Permission Catalog

#### User Permissions

| Permission | Description |
|------------|-------------|
| `users:create` | Create new users within the organization |
| `users:read` | View user profiles |
| `users:update` | Modify user profiles (name, email, status) |
| `users:delete` | Delete users |
| `users:list` | List and search users |
| `users:migrate` | Move users between organizations (super_admin only) |

#### Role Permissions

| Permission | Description |
|------------|-------------|
| `roles:create` | Create custom roles |
| `roles:read` | View role definitions |
| `roles:update` | Modify role permissions and properties |
| `roles:delete` | Delete custom roles |
| `roles:assign` | Assign or remove roles from users |

#### OAuth Client Permissions

| Permission | Description |
|------------|-------------|
| `clients:create` | Register new OAuth clients |
| `clients:read` | View client configurations |
| `clients:update` | Modify client settings (redirect URIs, scopes, etc.) |
| `clients:delete` | Delete OAuth clients |

#### Session Permissions

| Permission | Description |
|------------|-------------|
| `sessions:read` | View active sessions (all users in the org) |
| `sessions:revoke` | Revoke individual sessions |
| `sessions:revoke_all` | Revoke all sessions for a user or the entire org |

#### Audit Permissions

| Permission | Description |
|------------|-------------|
| `audit:read` | Read audit events within the organization |
| `audit:read_global` | Read audit events across all organizations (super_admin only) |

#### Organization Permissions

| Permission | Description |
|------------|-------------|
| `org:update` | Modify organization settings |
| `organizations:create` | Create new organizations (super_admin only) |
| `organizations:update` | Modify any organization (super_admin only) |
| `organizations:delete` | Delete organizations (super_admin only) |
| `organizations:list` | List all organizations (super_admin only) |

#### Account Permissions (Self-Service)

| Permission | Description |
|------------|-------------|
| `account:read` | View own profile |
| `account:update` | Update own profile |
| `account:mfa` | Manage own MFA settings |
| `account:sessions` | View and revoke own sessions |
| `account:delete` | Delete own account |

#### Identity Provider Permissions

| Permission | Description |
|------------|-------------|
| `idp:create` | Configure new identity providers |
| `idp:update` | Modify identity provider settings |
| `idp:delete` | Remove identity providers |

#### System Permissions

| Permission | Description |
|------------|-------------|
| `system:configure` | Modify instance-level settings (super_admin only) |
| `system:metrics` | Access health and performance metrics (super_admin only) |

## Role Assignment

### Assigning Roles to Users

**Via API:**

```bash
curl -X POST https://auth.example.com/api/admin/users/{user_id}/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "roles": ["support_agent", "client_manager"]
  }'
```

**Via admin console:** Open a user's profile, navigate to the Roles tab, and toggle roles on or off.

### Removing Roles

```bash
curl -X DELETE https://auth.example.com/api/admin/users/{user_id}/roles/support_agent \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Viewing a User's Roles

```bash
curl https://auth.example.com/api/admin/users/{user_id}/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Assignment Rules

- A user can have multiple roles simultaneously
- Permissions from all assigned roles are combined (union)
- The `user` role cannot be removed — every user always has at least the base permissions
- Assigning the `org_admin` role requires `roles:assign` permission (held by `org_admin` and `super_admin`)
- Assigning the `super_admin` role requires an existing `super_admin`
- Role changes take effect on the next token issuance — existing access tokens retain the old roles until they expire

### Role Changes and Active Tokens

When a user's roles change:

1. The change is persisted immediately in the database
2. **Existing access tokens are not invalidated** — they contain the old roles and remain valid until expiration
3. The next token refresh or login will issue tokens with the updated roles
4. For immediate effect, an admin can revoke the user's sessions (forces re-authentication)

This is a deliberate design choice — it avoids the latency of checking role assignments on every API request while keeping access tokens stateless and verifiable without a database lookup.

## Inheritance

Rampart uses a flat permission model — roles do not inherit from other roles. This keeps the authorization model simple, predictable, and auditable.

**Why no role inheritance:**

- Inherited permissions create hidden access paths that are hard to audit
- Flat roles make it obvious what a role can do by looking at its permission list
- Debugging authorization issues is straightforward — check the user's roles and their permissions

**Composing access:** Instead of inheritance, assign multiple roles to a user. For example, a user who is both a support agent and a security officer gets the combined permissions of both roles:

```
User "jane@acme.com"
  +-- Role: support_agent
  |     +-- users:read, users:list, users:update
  |     +-- sessions:read, sessions:revoke
  |
  +-- Role: security_officer
        +-- audit:read, sessions:read
        +-- sessions:revoke_all, users:read

  Effective permissions (union):
    users:read, users:list, users:update,
    sessions:read, sessions:revoke, sessions:revoke_all,
    audit:read
```

## Roles in Tokens

User roles are included in JWT access tokens as the `roles` claim:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "roles": ["user", "support_agent"],
  "org": "acme-corp",
  ...
}
```

Resource servers can use the `roles` claim to make authorization decisions without calling back to Rampart. This makes authorization fast and decoupled from the identity provider.

For fine-grained authorization, resource servers can map Rampart roles to their own internal permissions. Rampart's SDK adapters (Node.js, Go, Python) provide middleware helpers for role-based route protection:

```javascript
// Node.js adapter example
app.get('/admin/users', requireRole('org_admin'), (req, res) => {
  // Only org_admin and super_admin can access this route
});
```

## Security Considerations

- **Principle of least privilege** — assign the minimum roles necessary for each user's function
- **Regular role audits** — review role assignments periodically; remove stale or excessive access
- **Separation of duties** — avoid giving the same user both `org_admin` and `security_officer` roles in sensitive environments
- **MFA for privileged roles** — enforce MFA for `super_admin` and `org_admin` through the organization's MFA policy
- **Audit role changes** — all role assignment and removal operations are recorded in the audit log with the actor, target, and timestamp
