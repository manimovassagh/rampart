---
sidebar_position: 1
title: Authentication
description: How authentication works in Rampart — credential verification, password hashing, MFA, social login, token lifecycle, and session management.
---

# Authentication

Rampart provides a complete, production-grade authentication system that handles user registration, credential verification, multi-factor authentication, social login, token issuance, and session management. Every authentication event is audited and rate-limited by default.

## Authentication Flows

Rampart supports multiple authentication methods that can be combined per organization.

### Username/Password Authentication

The primary authentication flow uses email and password credentials.

**Registration:**

1. The client submits email, password, and profile fields to `POST /register`
2. Rampart validates input — email format, password policy compliance, required fields
3. Duplicate check — ensures the email is not already registered within the organization
4. Password is hashed (see [Password Hashing](#password-hashing) below)
5. A user record is created in PostgreSQL with a generated UUID
6. The default `user` role is assigned
7. A `user.created` audit event is emitted
8. The response includes the user profile (never the password hash)

**Login:**

1. The client sends email and password to `POST /login`
2. Rampart looks up the user by email within the target organization
3. The submitted password is verified against the stored hash
4. If MFA is enabled for the user, a challenge is issued (see [Multi-Factor Authentication](#multi-factor-authentication))
5. On success, Rampart generates an access token, refresh token, and (if requested) an ID token
6. A session is created in PostgreSQL with metadata (IP, user agent, geo, timestamp)
7. A `user.login` audit event is logged

**Security:** Failed login attempts never reveal whether an email exists. The error response is identical for "unknown email" and "wrong password" — this prevents user enumeration attacks. Failed attempts are rate-limited per IP and per account.

### Social Login

Rampart supports federated authentication through external identity providers using OAuth 2.0 and OpenID Connect.

**Supported providers:**

| Provider | Protocol | Configuration |
|----------|----------|---------------|
| Google | OIDC | Client ID + Secret |
| GitHub | OAuth 2.0 | Client ID + Secret |
| Microsoft / Azure AD | OIDC | Client ID + Secret + Tenant |
| Apple | OIDC | Client ID + Team ID + Key |
| Generic OIDC | OIDC | Discovery URL + Client ID + Secret |
| Generic OAuth 2.0 | OAuth 2.0 | Authorize/Token URLs + Client ID + Secret |

**Social login flow:**

1. The client redirects the user to `GET /oauth/authorize` with the provider identifier
2. Rampart redirects to the external provider's authorization endpoint
3. The user authenticates with the external provider
4. The provider redirects back to Rampart's callback URL with an authorization code
5. Rampart exchanges the code for tokens with the provider
6. Rampart extracts the user's identity from the provider's ID token or userinfo endpoint
7. If the user exists (matched by email or provider subject), Rampart links the account
8. If the user is new, Rampart creates an account (if self-registration is enabled for the org)
9. Rampart issues its own tokens and creates a session

**Account linking:** When a user who registered with email/password later logs in via a social provider with the same email, Rampart links the social identity to the existing account. The user can then log in using either method.

Social login providers are configured per organization through the Admin API or the admin console.

### Multi-Factor Authentication

Rampart supports multi-factor authentication (MFA) to add a second layer of verification after the primary credential check.

**Supported MFA methods:**

| Method | Standard | Description |
|--------|----------|-------------|
| TOTP | RFC 6238 | Time-based one-time passwords (Google Authenticator, Authy, etc.) |
| WebAuthn | FIDO2 | Hardware security keys, platform authenticators (Touch ID, Windows Hello) |

**MFA enrollment:**

1. The user initiates enrollment via `POST /api/account/mfa/enroll`
2. For TOTP: Rampart generates a secret and returns it as a `otpauth://` URI and QR code data
3. The user scans the QR code with their authenticator app
4. The user submits a verification code to `POST /api/account/mfa/verify` to confirm enrollment
5. Rampart stores the MFA configuration and generates recovery codes
6. Recovery codes are returned once — Rampart stores only their hashes

**MFA-protected login flow:**

```
Client                         Rampart                        User
  |                              |                              |
  |  POST /login (email+pass)   |                              |
  |----------------------------->|                              |
  |                              |  Verify password             |
  |                              |                              |
  |  202 { mfa_required, token } |                              |
  |<-----------------------------|                              |
  |                              |                              |
  |  POST /mfa/challenge         |                              |
  |  { mfa_token, totp_code }   |                              |
  |----------------------------->|                              |
  |                              |  Verify TOTP code            |
  |                              |                              |
  |  200 { access_token, ... }   |                              |
  |<-----------------------------|                              |
```

1. The user submits email and password
2. If credentials are valid and MFA is enabled, Rampart returns `202 Accepted` with an `mfa_token` (short-lived, single-use)
3. The client prompts the user for their second factor
4. The client submits the MFA token and the verification code
5. On success, Rampart issues the full token set and creates a session

**MFA policies** (configurable per organization):

- **Optional** — users may enable MFA but are not required to
- **Encouraged** — users are prompted to enable MFA after login but can skip
- **Required** — all users must enroll in MFA; login is blocked until enrollment is complete
- **Required for admins** — only users with admin roles must have MFA enabled

## Password Hashing

Rampart uses industry-standard password hashing algorithms to protect stored credentials.

### Argon2id (Default)

The default and only hashing algorithm for user passwords is **argon2id**, the winner of the Password Hashing Competition and the current OWASP recommendation:

- Memory-hard — resistant to GPU and ASIC attacks
- Default parameters: 64 MB memory, 3 iterations, 4 threads
- 16-byte cryptographically random salt per password
- 32-byte key length
- Output in PHC string format: `$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>`

These parameters are defined as constants in the `internal/auth` package.

OAuth client secrets use **bcrypt** for hashing, since they are high-entropy random strings that do not benefit from argon2id's memory-hardness.

### Password Policy

Password policies are configured per organization:

| Policy | Default | Description |
|--------|---------|-------------|
| `min_length` | 8 | Minimum password length |
| `max_length` | 128 | Maximum password length |
| `require_uppercase` | true | At least one uppercase letter |
| `require_lowercase` | true | At least one lowercase letter |
| `require_digit` | true | At least one numeric digit |
| `require_special` | false | At least one special character |
| `reject_common` | true | Reject passwords from the common password list |
| `reject_user_info` | true | Reject passwords containing the user's name or email |
| `history_count` | 0 | Number of previous passwords to remember (0 = disabled) |

Passwords are **never** stored in plaintext, logged, included in API responses, or transmitted in query strings.

## Token Lifecycle

Rampart issues three types of tokens during authentication. Each serves a distinct purpose and has its own lifecycle.

### Access Tokens

Access tokens are short-lived JWTs that authorize API requests.

| Property | Value |
|----------|-------|
| Format | JWT (JSON Web Token) |
| Signing algorithm | RS256 (RSA with SHA-256) |
| Default lifetime | 1 hour |
| Configuration | `RAMPART_ACCESS_TOKEN_TTL` |
| Storage | Client-side only (not stored by Rampart) |

**JWT structure:**

```
Header.Payload.Signature
```

**Header:**

```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "rampart-key-1"
}
```

**Payload:**

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "iss": "https://auth.example.com",
  "aud": "rampart",
  "exp": 1741176000,
  "iat": 1741172400,
  "nbf": 1741172400,
  "jti": "unique-token-id",
  "email": "admin@example.com",
  "roles": ["admin"],
  "org": "default"
}
```

**Verification steps** (performed by resource servers):

1. Fetch the public key from `GET /.well-known/jwks.json` (cache with respect to `Cache-Control` headers)
2. Verify the RS256 signature using the key matching the `kid` header
3. Check that `exp` is in the future
4. Check that `nbf` is in the past
5. Verify that `iss` matches the expected issuer
6. Verify that `aud` matches the expected audience
7. Optionally validate `roles` and `org` claims for authorization decisions

### Refresh Tokens

Refresh tokens are long-lived, opaque tokens used to obtain new access tokens without re-authentication.

| Property | Value |
|----------|-------|
| Format | Opaque (cryptographically random string) |
| Default lifetime | 7 days |
| Configuration | `RAMPART_REFRESH_TOKEN_TTL` |
| Storage | Server-side (PostgreSQL) |
| Usage | Single-use with automatic rotation |

**Refresh token rotation:**

```
Client                         Rampart
  |                              |
  |  POST /oauth/token           |
  |  grant_type=refresh_token    |
  |  refresh_token=RT_1          |
  |----------------------------->|
  |                              |  Validate RT_1
  |                              |  Invalidate RT_1
  |                              |  Issue new AT_2 + RT_2
  |  { access_token: AT_2,       |
  |    refresh_token: RT_2 }     |
  |<-----------------------------|
```

Each refresh token can only be used once. When a refresh token is exchanged, Rampart:

1. Validates the token exists and has not expired or been revoked
2. Invalidates the old refresh token immediately
3. Issues a new access token and a new refresh token
4. Links the new refresh token to the same session

**Replay detection:** If a previously used refresh token is presented again, Rampart treats this as a potential token theft. The entire token family (all refresh tokens in the chain) is revoked, and the session is terminated. This forces the legitimate user to re-authenticate.

### ID Tokens

ID tokens are JWTs issued when the `openid` scope is requested. They contain identity claims about the authenticated user and are consumed by the client application (not by resource servers).

| Property | Value |
|----------|-------|
| Format | JWT |
| Signing algorithm | RS256 |
| Default lifetime | 1 hour (matches access token) |
| Audience | The OAuth client ID |

See the [OAuth 2.0 and OpenID Connect](./oauth-oidc.md) page for full details on ID token claims and scopes.

### Token Lifecycle Diagram

```
                    Registration
                         |
                         v
  +-- Login (password + optional MFA) --+
  |                                      |
  v                                      v
Access Token (1h)                  Refresh Token (7d)
  |                                      |
  |  Used in Authorization header        |  Used to get new tokens
  |                                      |
  v                                      v
Expires                            POST /oauth/token
  |                                grant_type=refresh_token
  |                                      |
  |                                      v
  |                              New Access Token (1h)
  |                              New Refresh Token (7d)
  |                              Old Refresh Token invalidated
  |                                      |
  +--------------------------------------+
  |
  v
Logout --> All tokens revoked, session destroyed
```

**Summary:**

1. User authenticates (password, social login, or MFA)
2. Rampart issues an access token, refresh token, and optionally an ID token
3. The client uses the access token in `Authorization: Bearer <token>` headers
4. When the access token expires, the client exchanges the refresh token for a new token pair
5. The old refresh token is invalidated; replay attempts revoke the entire chain
6. On logout, all tokens are revoked and the session is destroyed
7. If the refresh token expires without being used, the user must re-authenticate

## Session Creation

Every successful authentication creates a server-side session in PostgreSQL. The session stores:

| Field | Description |
|-------|-------------|
| `session_id` | Unique session identifier (UUID) |
| `user_id` | The authenticated user's ID |
| `org_id` | The organization context |
| `ip_address` | Client IP at the time of login |
| `user_agent` | Client's User-Agent header |
| `created_at` | When the session was created |
| `last_active` | Last time the session was used |
| `expires_at` | When the session will be automatically cleaned up |
| `mfa_verified` | Whether MFA was completed for this session |

Sessions are used for:

- Tracking active logins across devices
- Enabling single-session or session-limit policies
- Supporting "log out everywhere" functionality
- Providing the session list in the admin console and user account pages

For full details on session management, see [Sessions](./sessions.md).

## Security Considerations

- **Timing-safe comparison** — password verification uses constant-time comparison to prevent timing attacks
- **Rate limiting** — login attempts are rate-limited per IP (default: 10/minute) and per account (default: 5/minute)
- **Account lockout** — after a configurable number of failed attempts (default: 10), the account is temporarily locked
- **Brute-force protection** — progressive delays are applied after repeated failures
- **No credential leakage** — passwords are never logged, returned in API responses, or stored in plaintext
- **Secure token transport** — tokens are transmitted only over HTTPS in production; cookies use `Secure`, `HttpOnly`, and `SameSite=Lax` flags
- **Audit trail** — every authentication event (success and failure) is recorded in the audit log with IP, user agent, and timestamp
