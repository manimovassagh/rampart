---
sidebar_position: 5
title: Sessions
description: PostgreSQL-backed session management in Rampart — session lifecycle, listing, revocation, security protections, and session policies.
---

# Sessions

Rampart uses PostgreSQL-backed server-side sessions to track authenticated users across requests. Sessions provide visibility into active logins, enable instant revocation, and support security policies like concurrent session limits and idle timeouts.

## PostgreSQL-Backed Sessions

Every successful authentication creates a session in PostgreSQL. Sessions are the server-side record of an authenticated user's login — they are separate from (but linked to) OAuth tokens.

**Why PostgreSQL (no Redis):**

- Single infrastructure dependency — no additional services to operate
- ACID-compliant session operations with full durability
- Consistent read-after-write guarantees
- Efficient indexing for session lookups and listing
- Simplified deployment and operational model

### Session Data Structure

Each session is stored with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string (UUID) | Unique session identifier |
| `user_id` | string (UUID) | The authenticated user |
| `org_id` | string (UUID) | The organization context |
| `client_id` | string | The OAuth client that initiated the session |
| `ip_address` | string | Client IP at login time |
| `user_agent` | string | Client's User-Agent string |
| `device_info` | string | Parsed device description (e.g., "Chrome 120 on macOS") |
| `geo` | string | Approximate location derived from IP (e.g., "San Francisco, US") |
| `created_at` | timestamp | When the session was created |
| `last_active` | timestamp | Last time the session was used for a token operation |
| `expires_at` | timestamp | Absolute session expiration |
| `mfa_verified` | boolean | Whether MFA was completed for this session |
| `refresh_token_id` | string | ID of the current refresh token linked to this session |

## Session Lifecycle

### Creation

A session is created when:

1. A user successfully authenticates via login (password, social login, or MFA)
2. An OAuth authorization code is exchanged for tokens
3. A device code flow completes successfully

**What happens during creation:**

1. Rampart generates a UUID for the session
2. Client metadata is extracted (IP, user agent, geo lookup)
3. The session record is written to PostgreSQL
4. The session is linked to the issued refresh token
5. A `session.created` audit event is emitted

### Activity Tracking

The `last_active` timestamp is updated whenever:

- A refresh token linked to this session is used
- The user accesses a session-aware endpoint

This enables idle timeout enforcement and provides accurate "last seen" information in the admin console.

### Expiration

Sessions expire through three mechanisms:

| Mechanism | Trigger | Description |
|-----------|---------|-------------|
| **Absolute timeout** | `expires_at` reached | The session ends regardless of activity (default: 30 days) |
| **Idle timeout** | No activity for the configured duration | The session ends if `last_active` is older than the idle threshold (default: 7 days) |
| **Background cleanup** | Periodic background job | Expired sessions are purged from PostgreSQL |

When a session expires:

1. The associated refresh token becomes invalid
2. Token refresh attempts return an error
3. The user must re-authenticate
4. A background job periodically cleans up expired session records

### Destruction

Sessions are explicitly destroyed when:

- The user logs out (`POST /oauth/logout`)
- An admin revokes the session
- A refresh token replay is detected (the entire session is terminated)
- The user's account is disabled or deleted
- The organization is disabled

## Session Listing

### User's Own Sessions

Users can view their active sessions through the Account API:

```bash
curl https://auth.example.com/api/account/sessions \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**

```json
{
  "sessions": [
    {
      "session_id": "a1b2c3d4-...",
      "ip_address": "203.0.113.42",
      "device_info": "Chrome 120 on macOS",
      "geo": "San Francisco, US",
      "created_at": "2026-03-05T08:30:00Z",
      "last_active": "2026-03-05T14:22:00Z",
      "current": true
    },
    {
      "session_id": "e5f6g7h8-...",
      "ip_address": "198.51.100.17",
      "device_info": "Safari on iPhone",
      "geo": "New York, US",
      "created_at": "2026-03-04T19:15:00Z",
      "last_active": "2026-03-05T09:00:00Z",
      "current": false
    }
  ],
  "total": 2
}
```

The `current` field indicates which session corresponds to the token making the request.

### Admin Session Listing

Admins can list sessions for any user in their organization:

```bash
# Sessions for a specific user
curl https://auth.example.com/api/admin/users/{user_id}/sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# All active sessions in the organization
curl https://auth.example.com/api/admin/sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Filtered by IP or date range
curl "https://auth.example.com/api/admin/sessions?ip=203.0.113.42&since=2026-03-01" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

The admin console provides a visual session browser with search, filtering, and bulk actions.

## Session Revocation

### Revoking a Single Session

**User revoking their own session:**

```bash
curl -X DELETE https://auth.example.com/api/account/sessions/{session_id} \
  -H "Authorization: Bearer $TOKEN"
```

**Admin revoking a user's session:**

```bash
curl -X DELETE https://auth.example.com/api/admin/sessions/{session_id} \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**What happens when a session is revoked:**

1. The session record is deleted from PostgreSQL
2. The linked refresh token is invalidated
4. Any subsequent token refresh attempt using that refresh token returns `invalid_grant`
5. A `session.revoked` audit event is emitted with the actor (user or admin)
6. The access token remains valid until it expires (short-lived by design)

### Revoking All Sessions for a User

**User logging out everywhere:**

```bash
curl -X DELETE https://auth.example.com/api/account/sessions \
  -H "Authorization: Bearer $TOKEN"
```

**Admin revoking all sessions for a user:**

```bash
curl -X DELETE https://auth.example.com/api/admin/users/{user_id}/sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

This revokes every active session for the user, invalidates all refresh tokens, and forces re-authentication on all devices.

### Bulk Revocation (Organization-Wide)

Super admins and org admins can revoke all sessions across the organization:

```bash
curl -X DELETE https://auth.example.com/api/admin/sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Use cases for bulk revocation:

- Responding to a security incident
- Forcing re-authentication after a policy change (e.g., enabling mandatory MFA)
- Rotating signing keys and forcing new token issuance

A `session.bulk_revoked` audit event is emitted with the scope and count of revoked sessions.

## Session Security

### Session Fixation Prevention

Rampart prevents session fixation attacks by generating a new session ID on every authentication event. There is no way for an attacker to pre-set a session ID.

- Session IDs are cryptographically random UUIDs (128 bits of entropy)
- Session IDs are never exposed in URLs or query parameters
- Session IDs are transmitted only in secure, HttpOnly cookies or as opaque token references

### Session Rotation

When a refresh token is exchanged for new tokens, Rampart optionally rotates the session:

1. A new session ID is generated
2. The session data is migrated to the new key
3. The old session key is deleted
4. The new session is linked to the new refresh token

Session rotation can be configured per organization:

```yaml
session:
  rotation:
    enabled: true
    on_refresh: true     # Rotate on every token refresh
    on_privilege: true   # Rotate after privilege escalation (e.g., MFA verification)
```

### Concurrent Session Limits

Organizations can limit the number of concurrent sessions per user:

| Policy | Behavior |
|--------|----------|
| `unlimited` | No limit on concurrent sessions (default) |
| `max_sessions: N` | When the limit is reached, the oldest session is revoked |
| `single_session` | Only one active session allowed; new login revokes the previous session |

**Configuration:**

```yaml
session:
  max_per_user: 10
  on_limit_exceeded: revoke_oldest  # or "reject_new"
```

When `on_limit_exceeded` is set to `reject_new`, the login attempt is rejected with an error instructing the user to end an existing session first.

### Suspicious Session Detection

Rampart flags sessions that exhibit unusual patterns:

| Signal | Detection | Action |
|--------|-----------|--------|
| **IP change** | The IP address used for a token refresh differs significantly from the session's original IP | Emit `session.suspicious` event; optionally require re-authentication |
| **Impossible travel** | Geo-location of the new IP is inconsistent with the time elapsed since the last activity | Emit `session.impossible_travel` event; optionally revoke the session |
| **User agent change** | The User-Agent header changes mid-session | Emit `session.suspicious` event |

These signals are logged as audit events and can trigger webhook notifications. The response (log only, require MFA, or revoke) is configurable per organization.

## Session Policies

Session behavior is configurable per organization through the Admin API or the admin console.

| Policy | Default | Description |
|--------|---------|-------------|
| `absolute_timeout` | 30 days | Maximum session lifetime regardless of activity |
| `idle_timeout` | 7 days | Session expires after this period of inactivity |
| `max_per_user` | 10 | Maximum concurrent sessions per user |
| `on_limit_exceeded` | `revoke_oldest` | What to do when session limit is reached |
| `rotation_on_refresh` | `true` | Rotate session ID on token refresh |
| `rotation_on_privilege` | `true` | Rotate session ID after MFA verification |
| `require_mfa_for_new_device` | `false` | Require MFA when a new device is detected |
| `bind_to_ip` | `false` | Invalidate the session if the client IP changes |

**API configuration:**

```bash
curl -X PUT https://auth.example.com/api/admin/organizations/acme-corp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "session_policy": {
      "absolute_timeout": "720h",
      "idle_timeout": "168h",
      "max_per_user": 5,
      "on_limit_exceeded": "revoke_oldest",
      "bind_to_ip": false
    }
  }'
```

## High Availability

In production deployments with multiple Rampart instances, all instances share the same PostgreSQL database. This ensures:

- Sessions created on one instance are immediately visible to all others
- Session revocation takes effect across all instances instantly
- No sticky sessions or session affinity is required at the load balancer level

Rampart uses PostgreSQL-based leader election with graceful shutdown for clustering. No Redis or external cache is required.
