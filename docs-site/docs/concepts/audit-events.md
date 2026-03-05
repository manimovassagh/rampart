---
sidebar_position: 6
title: Audit Events
description: Audit logging in Rampart — event types, event schema, querying via API and admin console, retention policies, and compliance support.
---

# Audit Events

Rampart records a comprehensive audit trail of all security-relevant actions. Every authentication attempt, role change, session operation, and administrative action is captured as an immutable event. The audit log is essential for security monitoring, incident response, compliance reporting, and operational debugging.

## Event Types

Rampart defines a structured set of event types organized by category. Each event type uses a `category.action` naming convention.

### Authentication Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `auth.login` | User successfully authenticated | info |
| `auth.login_failed` | Authentication attempt failed (wrong password, unknown user, locked account) | warning |
| `auth.logout` | User logged out | info |
| `auth.mfa_challenge` | MFA challenge issued after primary authentication | info |
| `auth.mfa_success` | MFA verification succeeded | info |
| `auth.mfa_failed` | MFA verification failed | warning |
| `auth.password_reset_requested` | Password reset email was sent | info |
| `auth.password_reset_completed` | Password was reset via reset link | info |
| `auth.password_changed` | User changed their password (while authenticated) | info |
| `auth.account_locked` | Account locked due to repeated failed attempts | warning |
| `auth.account_unlocked` | Account unlocked by admin or automatic timeout | info |

### Token Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `token.issued` | Access/refresh tokens issued | info |
| `token.refreshed` | Tokens refreshed via refresh token | info |
| `token.revoked` | Token explicitly revoked | info |
| `token.replay_detected` | A previously used refresh token was presented (potential theft) | critical |
| `token.introspected` | Token introspection endpoint called | info |

### Session Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `session.created` | New session established | info |
| `session.revoked` | Single session revoked | info |
| `session.bulk_revoked` | Multiple sessions revoked in one operation | warning |
| `session.expired` | Session expired due to timeout | info |
| `session.suspicious` | Suspicious session activity detected (IP change, user agent change) | warning |
| `session.impossible_travel` | Geo-location inconsistency detected | critical |

### User Management Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `user.created` | New user registered or created by admin | info |
| `user.updated` | User profile updated | info |
| `user.deleted` | User account deleted | warning |
| `user.disabled` | User account disabled | warning |
| `user.enabled` | User account re-enabled | info |
| `user.migrated` | User moved to a different organization | warning |
| `user.mfa_enrolled` | User enrolled in MFA | info |
| `user.mfa_disabled` | User or admin disabled MFA | warning |

### Role Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `role.created` | Custom role created | info |
| `role.updated` | Role permissions modified | warning |
| `role.deleted` | Custom role deleted | warning |
| `role.assigned` | Role assigned to a user | info |
| `role.unassigned` | Role removed from a user | warning |

### Client Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `client.created` | OAuth client registered | info |
| `client.updated` | Client configuration modified | info |
| `client.deleted` | OAuth client deleted | warning |
| `client.secret_rotated` | Client secret was rotated | info |

### Organization Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `org.created` | New organization created | info |
| `org.updated` | Organization settings modified | info |
| `org.disabled` | Organization disabled | critical |
| `org.enabled` | Organization re-enabled | info |
| `org.deleted` | Organization permanently deleted | critical |

### Identity Provider Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `idp.created` | Identity provider configured | info |
| `idp.updated` | Identity provider settings modified | info |
| `idp.deleted` | Identity provider removed | warning |
| `idp.login` | User authenticated via external identity provider | info |
| `idp.link` | External identity linked to existing account | info |

### System Events

| Event Type | Description | Severity |
|------------|-------------|----------|
| `system.config_changed` | Instance-level configuration modified | warning |
| `system.key_rotated` | Signing key rotated | info |
| `system.startup` | Rampart instance started | info |
| `system.shutdown` | Rampart instance shut down gracefully | info |

## Event Schema

Every audit event follows a consistent JSON schema. Events are immutable once written — they cannot be modified or deleted through the API.

### Schema Definition

```json
{
  "event_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "event_type": "auth.login",
  "severity": "info",
  "timestamp": "2026-03-05T14:22:31.847Z",
  "org_id": "acme-corp",
  "actor": {
    "type": "user",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "jane@acme.com",
    "ip_address": "203.0.113.42",
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)...",
    "geo": "San Francisco, US"
  },
  "target": {
    "type": "session",
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  },
  "details": {
    "method": "password",
    "mfa_used": true,
    "client_id": "web-app"
  },
  "request_id": "req-abc123"
}
```

### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event_id` | UUID | yes | Unique identifier for the event |
| `event_type` | string | yes | The event type (e.g., `auth.login`) |
| `severity` | enum | yes | `info`, `warning`, or `critical` |
| `timestamp` | ISO 8601 | yes | When the event occurred (UTC, millisecond precision) |
| `org_id` | string | yes | The organization context (or `system` for global events) |
| `actor` | object | yes | Who performed the action |
| `actor.type` | enum | yes | `user`, `admin`, `client`, or `system` |
| `actor.id` | string | yes | Actor's unique identifier |
| `actor.email` | string | no | Actor's email (when applicable) |
| `actor.ip_address` | string | no | Client IP address |
| `actor.user_agent` | string | no | Client User-Agent header |
| `actor.geo` | string | no | Approximate location from IP |
| `target` | object | no | The resource affected by the action |
| `target.type` | enum | no | `user`, `role`, `client`, `session`, `org`, `idp` |
| `target.id` | string | no | Target's unique identifier |
| `details` | object | no | Event-specific metadata (varies by event type) |
| `request_id` | string | no | Correlation ID for request tracing |

### Actor Types

| Actor Type | Description |
|------------|-------------|
| `user` | An authenticated end user performing a self-service action |
| `admin` | An admin user performing an administrative action |
| `client` | An OAuth client acting autonomously (e.g., client credentials flow) |
| `system` | The Rampart system itself (e.g., automatic session expiration, scheduled tasks) |

### Security Notes on Event Data

- **Sensitive data is never logged.** Passwords, tokens, client secrets, and MFA codes are excluded from events.
- **IP addresses are logged.** This is necessary for security monitoring but may have privacy implications. See [Retention and Compliance](#retention-and-compliance) for data handling.
- **Events are append-only.** There is no API to modify or delete individual events. Retention policies handle cleanup.

## Querying Events

### Via the Admin API

**List events for the current organization:**

```bash
curl "https://auth.example.com/api/admin/audit-events" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Filter by event type:**

```bash
curl "https://auth.example.com/api/admin/audit-events?type=auth.login_failed" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Filter by date range:**

```bash
curl "https://auth.example.com/api/admin/audit-events?since=2026-03-01T00:00:00Z&until=2026-03-05T23:59:59Z" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Filter by actor:**

```bash
curl "https://auth.example.com/api/admin/audit-events?actor_id=550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Filter by severity:**

```bash
curl "https://auth.example.com/api/admin/audit-events?severity=critical" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Filter by IP address:**

```bash
curl "https://auth.example.com/api/admin/audit-events?ip=203.0.113.42" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Combined filters with pagination:**

```bash
curl "https://auth.example.com/api/admin/audit-events?type=auth.login_failed&severity=warning&since=2026-03-01&limit=50&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Response format:**

```json
{
  "events": [
    {
      "event_id": "...",
      "event_type": "auth.login_failed",
      "severity": "warning",
      "timestamp": "2026-03-05T14:22:31.847Z",
      "actor": { "type": "user", "ip_address": "203.0.113.42", "email": "jane@acme.com" },
      "details": { "reason": "invalid_password" }
    }
  ],
  "total": 142,
  "limit": 50,
  "offset": 0
}
```

**Cross-organization query (super_admin only):**

```bash
curl "https://auth.example.com/api/admin/audit-events?scope=global&type=org.disabled" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN"
```

### Via the Admin Console

The admin console provides a visual audit log browser with:

- **Real-time event stream** — new events appear automatically at the top of the list
- **Full-text search** — search across event types, actor emails, IP addresses, and details
- **Faceted filtering** — filter by event category, severity, date range, and actor
- **Event detail view** — click any event to see the full JSON payload
- **Export** — download filtered events as CSV or JSON for external analysis
- **Visual timeline** — timeline view showing event density and patterns over time

### Via the CLI

```bash
# List recent events
rampart-cli audit list --limit 20

# Filter by type
rampart-cli audit list --type auth.login_failed --since 2026-03-01

# Follow events in real-time
rampart-cli audit tail --severity warning,critical

# Export events
rampart-cli audit export --since 2026-03-01 --format json > events.json
```

## Security Monitoring Patterns

### Detect Brute-Force Attacks

Look for repeated `auth.login_failed` events from the same IP:

```bash
curl "https://auth.example.com/api/admin/audit-events?type=auth.login_failed&ip=203.0.113.42&limit=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Detect Account Takeover

Look for `auth.password_changed` events followed by session revocations from a new IP:

```bash
curl "https://auth.example.com/api/admin/audit-events?type=auth.password_changed&actor_id={user_id}" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Detect Unauthorized Privilege Escalation

Monitor for `role.assigned` events where admin roles are being granted:

```bash
curl "https://auth.example.com/api/admin/audit-events?type=role.assigned&since=2026-03-01" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Detect Token Theft

Monitor for `token.replay_detected` events, which indicate a refresh token was used after it had already been consumed:

```bash
curl "https://auth.example.com/api/admin/audit-events?type=token.replay_detected&severity=critical" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Retention and Compliance

### Retention Policies

Audit events are stored in PostgreSQL. Retention is configurable per organization, with a global default.

| Setting | Default | Description |
|---------|---------|-------------|
| `retention_days` | 90 | Number of days to retain events before automatic deletion |
| `retention_min_days` | 30 | Minimum retention period (cannot be set lower) |
| `retention_max_days` | 3650 (10 years) | Maximum retention period |

**Configuration:**

```yaml
audit:
  retention:
    default_days: 90
    min_days: 30
    cleanup_schedule: "0 2 * * *"  # Run cleanup daily at 2 AM UTC
    batch_size: 10000              # Delete in batches to avoid long locks
```

**Per-organization override:**

```bash
curl -X PUT https://auth.example.com/api/admin/organizations/acme-corp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "audit_policy": {
      "retention_days": 365
    }
  }'
```

### Cleanup Process

Rampart runs a background job on the configured schedule that:

1. Identifies events older than the retention period for each organization
2. Deletes events in configurable batches to avoid locking the database
3. Logs the cleanup results as a `system.audit_cleanup` event
4. Vacuums the events table to reclaim disk space

### Compliance Support

Rampart's audit system is designed to support common compliance frameworks:

#### SOC 2

- **CC6.1 / CC6.2** — All logical access events (login, logout, failed auth) are logged
- **CC6.3** — Role changes and permission modifications are tracked with actor attribution
- **CC7.2** — Security events are monitored and can trigger alerts via webhooks

#### GDPR

- **Article 30** — Audit logs provide records of processing activities
- **Article 5(1)(f)** — Events demonstrate security measures in place
- **Data minimization** — events do not contain passwords, tokens, or unnecessary personal data
- **Right to erasure** — user-identifying fields in audit events can be pseudonymized via the admin API when a user exercises their right to be forgotten (the event structure is preserved)

#### HIPAA

- **164.312(b)** — Audit controls record activity in systems containing ePHI
- **164.312(c)(1)** — Immutable audit events ensure integrity of the log
- **164.308(a)(5)(ii)(C)** — Login monitoring provides the basis for security awareness

### External Log Shipping

For organizations that use centralized logging systems, Rampart supports shipping audit events to external destinations:

| Destination | Method | Configuration |
|-------------|--------|---------------|
| **Webhook** | HTTP POST for each event or batched | URL, headers, retry policy |
| **Syslog** | RFC 5424 over TCP/TLS | Host, port, facility, TLS config |
| **Stdout/Stderr** | Structured JSON logs | Log level filter |

**Webhook configuration example:**

```yaml
audit:
  webhooks:
    - url: "https://siem.example.com/api/events"
      headers:
        Authorization: "Bearer ${SIEM_TOKEN}"
      filter:
        severity: ["warning", "critical"]
      retry:
        max_attempts: 3
        backoff: "exponential"
      batch:
        size: 100
        interval: "10s"
```

Events are delivered at least once. Duplicate detection should be handled by the receiver using the `event_id` field.

### Immutability

Audit events are immutable by design:

- There is no API endpoint to update or delete individual events
- Retention-based cleanup is the only deletion mechanism
- Database-level protections (triggers or row-level policies) prevent direct modification
- Any attempt to tamper with the audit table is itself logged as a `system.audit_tamper_detected` event

This immutability provides confidence that the audit trail accurately reflects what happened, which is essential for incident investigation and compliance audits.
