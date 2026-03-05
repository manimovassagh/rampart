---
sidebar_position: 3
title: Security Architecture
description: Security design, threat model, and hardening measures in the Rampart IAM server.
---

# Security Architecture

Rampart is an identity and access management product. Security is not a feature of Rampart — it is the product. Every design decision, default configuration, and code path is evaluated through a security lens.

## Security Principles

1. **Defense in depth.** No single control prevents all attacks. Multiple overlapping layers protect against failures in any one mechanism.
2. **Secure by default.** Safe defaults out of the box. Administrators must explicitly opt in to weaker configurations.
3. **Principle of least privilege.** Users, clients, and internal components have only the permissions they need.
4. **Fail closed.** When something goes wrong, deny access rather than grant it.
5. **Auditability.** Every security-relevant action is logged with sufficient context for forensic analysis.

## Password Security

### Hashing

All passwords are hashed using **bcrypt** before storage. Plaintext passwords never touch disk or logs.

| Parameter | Default | Configurable |
|-----------|---------|-------------|
| Algorithm | bcrypt | No (bcrypt only; argon2id planned) |
| Work factor (cost) | 12 | Yes (`security.password.bcrypt_cost`) |
| Min password length | 8 | Yes (`security.password.min_length`) |
| Max password length | 128 | Yes (`security.password.max_length`) |

### Password Policy

Configurable per organization:

- Minimum length (default: 8)
- Maximum length (default: 128, to prevent bcrypt DoS via long inputs)
- Complexity requirements (optional: uppercase, lowercase, digit, special character)
- Breach database check via k-anonymity (optional, uses the HaveIBeenPwned Passwords API without sending the full hash)

### Credential Storage Rules

- Passwords are hashed immediately on receipt, before any other processing.
- Raw passwords are never written to logs, error messages, or debug output.
- Password hashes are excluded from all API responses, including admin endpoints.
- Client secrets for OAuth clients follow the same hashing and storage rules.

## Token Security

### Access Tokens (JWT)

| Property | Value |
|----------|-------|
| Format | JWT (RFC 7519) |
| Signing algorithm | RS256 (RSA 2048-bit minimum) |
| Default lifetime | 1 hour |
| Configurable lifetime | Yes (`tokens.access_token_ttl`) |
| Storage | Not stored server-side (stateless validation via signature) |

Access tokens contain standard OIDC claims (`sub`, `iss`, `aud`, `exp`, `iat`, `scope`) and are signed with the server's private key. Relying parties validate tokens using the public key from the JWKS endpoint.

### Refresh Tokens

| Property | Value |
|----------|-------|
| Format | Opaque random string (256-bit entropy) |
| Default lifetime | 30 days |
| Storage | SHA-256 hash stored in PostgreSQL |
| Rotation | New refresh token issued on each use (rotation enabled by default) |
| Revocation | Immediate via database flag |

Refresh token rotation mitigates the impact of token theft. When a refresh token is used, it is invalidated and a new one is issued. If a previously-used refresh token is presented, Rampart revokes the entire token family as a potential compromise indicator.

### Key Management

- RSA key pairs are generated on first startup if not provided.
- Keys can be supplied via configuration for environments where key material must be externally managed.
- Key rotation is supported: multiple keys can be active in the JWKS endpoint. New tokens are signed with the current key; old keys remain available for verification until removed.
- The JWKS endpoint (`/.well-known/jwks.json`) serves public keys for token verification.

## Session Security

| Property | Value |
|----------|-------|
| Session ID | Cryptographically random, 256-bit |
| Storage | Redis (fast lookup) + PostgreSQL (audit record) |
| Transport | HTTP-only, Secure, SameSite=Lax cookie |
| Default TTL | 24 hours (configurable) |
| Idle timeout | 30 minutes of inactivity (configurable) |

### Session Protections

- **HTTP-only cookies** prevent JavaScript access, mitigating XSS-based session theft.
- **Secure flag** ensures cookies are only sent over HTTPS.
- **SameSite=Lax** provides baseline CSRF protection for top-level navigations.
- **Session binding** — sessions are associated with the originating IP and user agent. Significant changes trigger re-authentication.
- **Concurrent session limits** — configurable maximum active sessions per user (default: unlimited).
- **Administrative revocation** — admins can view and terminate any user's active sessions.

## Input Validation

All input is validated at system boundaries before any processing:

- **Request body size limits** — maximum payload size enforced at the HTTP layer (default: 1MB).
- **String length limits** — all string fields have explicit maximum lengths.
- **Type validation** — UUIDs, emails, URLs, and other structured types are parsed and validated, not treated as raw strings.
- **Redirect URI validation** — exact match against the registered redirect URIs for the client. No wildcard matching, no partial matching, no open redirectors.
- **Scope validation** — requested scopes are checked against the client's allowed scopes.
- **SQL injection prevention** — all queries use parameterized statements via pgx. No string concatenation in SQL.
- **JSON validation** — request bodies are deserialized into strongly-typed Go structs. Unknown fields are rejected.

## Rate Limiting

Rate limiting protects against brute-force attacks and abuse:

| Endpoint | Default Limit | Key |
|----------|--------------|-----|
| `POST /login` | 5 per minute | IP + username |
| `POST /oauth/token` | 20 per minute | Client ID |
| `POST /api/v1/account/register` | 3 per minute | IP |
| Admin API (all) | 100 per minute | Authenticated user |
| All other endpoints | 60 per minute | IP |

Rate limit state is stored in Redis using sliding window counters. Limits are configurable per endpoint and can be overridden per organization.

When rate limited, the server responds with:
- HTTP `429 Too Many Requests`
- `Retry-After` header indicating when the client can retry
- No information about which limit was hit (to avoid information leakage)

### Account Lockout

After a configurable number of failed login attempts (default: 10), the account is temporarily locked:

| Parameter | Default |
|-----------|---------|
| Failed attempts before lockout | 10 |
| Lockout duration | 15 minutes |
| Lockout scope | Per user, per IP |

Lockout events are logged in the audit trail and can trigger webhook notifications.

## CORS Policy

CORS is configured per OAuth client, not globally. Each client registration includes allowed origins.

| Header | Policy |
|--------|--------|
| `Access-Control-Allow-Origin` | Matches against client's registered origins (no wildcards in production) |
| `Access-Control-Allow-Methods` | Matches the endpoint's allowed methods |
| `Access-Control-Allow-Headers` | `Authorization`, `Content-Type`, `X-Request-ID` |
| `Access-Control-Allow-Credentials` | `true` (for cookie-based flows) |
| `Access-Control-Max-Age` | 3600 seconds |

Preflight (`OPTIONS`) requests are handled automatically. Origins that do not match any registered client are rejected.

## Security Headers

All responses include the following security headers:

| Header | Value | Purpose |
|--------|-------|---------|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Enforce HTTPS |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `Content-Security-Policy` | Restrictive policy (self only, no inline scripts) | Prevent XSS |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limit referrer leakage |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Disable unused browser features |
| `Cache-Control` | `no-store` (on auth endpoints) | Prevent caching of sensitive responses |

## Threat Model Overview

### Authentication Threats

| Threat | Mitigation |
|--------|-----------|
| Credential stuffing | Rate limiting, account lockout, breach database check |
| Password brute force | bcrypt (slow hashing), rate limiting, lockout |
| Phishing | Redirect URI exact matching, no open redirectors |
| Session hijacking | HTTP-only + Secure cookies, session binding, TLS enforcement |
| Session fixation | New session ID generated on authentication |

### Token Threats

| Threat | Mitigation |
|--------|-----------|
| Token theft | Short access token lifetime (15 min), refresh token rotation |
| Token replay | `jti` claim for unique token IDs, token blacklist in Redis |
| Token forgery | RSA signature verification, JWKS-published public keys |
| Refresh token theft | Rotation with family revocation on reuse detection |

### OAuth 2.0 Threats

| Threat | Mitigation |
|--------|-----------|
| Authorization code interception | PKCE required for all public clients, recommended for confidential |
| Open redirect | Exact redirect URI matching, no wildcards |
| Client impersonation | Client authentication at token endpoint |
| Scope escalation | Scopes validated against client registration |
| CSRF on authorization | State parameter validation |

### Infrastructure Threats

| Threat | Mitigation |
|--------|-----------|
| SQL injection | Parameterized queries only (pgx), no string concatenation |
| XSS | Content-Security-Policy, input sanitization, HTTP-only cookies |
| Clickjacking | X-Frame-Options: DENY |
| Man-in-the-middle | HSTS enforcement, TLS-only in production |
| Denial of service | Rate limiting, request size limits, connection limits |

## Audit Logging

All security-relevant events are recorded in the `audit_events` table:

### Logged Events

| Event Type | Details Captured |
|-----------|-----------------|
| `user.login` | User ID, IP, user agent, organization |
| `user.login_failed` | Username attempted, IP, failure reason |
| `user.logout` | User ID, session terminated |
| `user.created` | New user ID, created by (admin) |
| `user.updated` | Changed fields (values redacted for sensitive fields) |
| `user.deleted` | User ID, deleted by (admin) |
| `user.locked` | User ID, reason (brute force, admin action) |
| `role.assigned` | User ID, role ID, assigned by |
| `role.revoked` | User ID, role ID, revoked by |
| `client.created` | Client ID, created by |
| `token.issued` | Client ID, user ID, scopes, grant type |
| `token.revoked` | Token family, revoked by, reason |
| `session.created` | User ID, IP, user agent |
| `session.revoked` | Session ID, revoked by |

### Audit Log Rules

- Audit records are **append-only**. No updates or deletes in normal operation.
- Sensitive values (passwords, tokens, secrets) are **never logged** — only identifiers and metadata.
- Audit records include **timestamps with timezone** for accurate forensic timelines.
- Log retention is configurable. Default: 90 days. Logs can be exported before deletion.

## Security Reporting

If you discover a security vulnerability in Rampart, please report it responsibly:

1. **Do not** open a public GitHub issue.
2. Email security concerns to the project maintainers (see SECURITY.md in the repository root).
3. Include a description of the vulnerability, steps to reproduce, and potential impact.
4. You will receive acknowledgment within 48 hours and a resolution timeline within 7 days.
