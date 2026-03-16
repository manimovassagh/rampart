---
sidebar_position: 2
title: Authentication Endpoints
description: Complete reference for Rampart's authentication endpoints -- registration, login, logout, token refresh, user profile, password reset, email verification, and MFA.
---

# Authentication Endpoints

This page covers Rampart's core authentication endpoints for user registration, login/logout, token management, password reset, email verification, and multi-factor authentication. These are JSON API endpoints separate from the [OAuth 2.0 endpoints](./oauth-endpoints.md).

**Content-Type:** All request and response bodies use `application/json`.

---

## POST /register

Create a new user account via self-registration.

**Auth required:** No

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Unique username (3--128 chars, lowercase alphanumeric, dots, hyphens, underscores) |
| `email` | string | Yes | Valid email address, unique within the organization |
| `password` | string | Yes | Must meet the organization's password policy |
| `given_name` | string | Yes | User's first name |
| `family_name` | string | Yes | User's last name |
| `org_slug` | string | No | Organization slug to register under (default: default organization) |

### Request

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

### Response (201 Created)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "org_id": "770e8400-e29b-41d4-a716-446655440001",
  "username": "jane.doe",
  "email": "jane@example.com",
  "email_verified": false,
  "given_name": "Jane",
  "family_name": "Doe",
  "enabled": true,
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T10:00:00Z"
}
```

### Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `bad_request` | Invalid JSON or missing required fields |
| 409 | `conflict` | Username or email already exists |
| 422 | `validation_error` | Field validation failed (weak password, invalid email, etc.) |

**Notes:**
- Inputs are normalized: email and username are lowercased, all fields are trimmed.
- If the organization has email verification enabled, a verification email is sent automatically after registration.
- Response time is constant (~250ms minimum) to prevent user enumeration via timing.

---

## POST /login

Authenticate a user with username/email and password. Returns access and refresh tokens.

**Auth required:** No

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `identifier` | string | Yes | Username or email address |
| `password` | string | Yes | User's password |
| `org_slug` | string | No | Organization slug (default: default organization) |

### Request

```bash
curl -X POST https://your-rampart-instance/login \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "jane@example.com",
    "password": "SecureP@ssw0rd!"
  }'
```

### Response (200 OK) -- Standard Login

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "org_id": "770e8400-e29b-41d4-a716-446655440001",
    "username": "jane.doe",
    "email": "jane@example.com",
    "email_verified": true,
    "given_name": "Jane",
    "family_name": "Doe",
    "enabled": true,
    "created_at": "2026-03-05T10:00:00Z",
    "updated_at": "2026-03-05T12:00:00Z"
  }
}
```

### Response (200 OK) -- MFA Required

When the user has MFA enabled, the server returns an MFA challenge instead of tokens:

```json
{
  "mfa_required": true,
  "mfa_token": "eyJhbGciOiJSUzI1NiIs...",
  "mfa_methods": ["totp", "webauthn"],
  "message": "MFA verification required."
}
```

Use the `mfa_token` with `POST /mfa/totp/verify` or `POST /mfa/webauthn/login/begin` to complete authentication.

### Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `bad_request` | Missing identifier or password |
| 401 | `unauthorized` | Invalid credentials (user not found, wrong password, account disabled, or account locked) |
| 403 | `email_not_verified` | Email verification required before login |

**Notes:**
- The error message is intentionally vague ("Invalid credentials.") to prevent user enumeration.
- Failed login attempts are tracked. After exceeding the max attempts (default: 5), the account is locked for a configurable duration (default: 15 minutes).
- Response time is constant (~250ms minimum) to prevent timing attacks.
- The `identifier` field accepts either a username or email address.

---

## POST /token/refresh

Exchange a refresh token for a new access token and refresh token.

**Auth required:** No

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `refresh_token` | string | Yes | A valid, unused refresh token |

### Request

```bash
curl -X POST https://your-rampart-instance/token/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4"}'
```

### Response (200 OK)

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...new-access-token",
  "refresh_token": "bmV3LXJlZnJlc2gtdG9rZW4",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `bad_request` | Invalid JSON |
| 401 | `unauthorized` | Refresh token is missing, invalid, expired, or already rotated |

**Notes:**
- Rampart implements **refresh token rotation**: each use issues a new refresh token and invalidates the old one. If a previously used refresh token is presented again, Rampart treats it as potential token theft and invalidates the session (replay detection).

---

## POST /logout

End a user session by invalidating the refresh token.

**Auth required:** No

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `refresh_token` | string | Yes | The refresh token to invalidate |

### Request

```bash
curl -X POST https://your-rampart-instance/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4"}'
```

### Response (204 No Content)

Returns an empty response on success. Also returns 204 if the refresh token is not found (prevents token fishing).

---

## GET /me

Returns the authenticated user's profile information from the JWT claims, plus linked social accounts.

**Auth required:** Yes (Bearer token)

### Request

```bash
curl -X GET https://your-rampart-instance/me \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..."
```

### Response (200 OK)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "org_id": "770e8400-e29b-41d4-a716-446655440001",
  "preferred_username": "jane.doe",
  "email": "jane@example.com",
  "email_verified": true,
  "given_name": "Jane",
  "family_name": "Doe",
  "social_accounts": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440002",
      "provider": "google",
      "email": "jane@gmail.com",
      "name": "Jane Doe"
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | User ID (UUID) |
| `org_id` | string | Organization ID (UUID) |
| `preferred_username` | string | The user's username |
| `email` | string | The user's email address |
| `email_verified` | boolean | Whether the email has been verified |
| `given_name` | string | First name |
| `family_name` | string | Last name |
| `social_accounts` | array | Linked social/federated identities (empty if none) |

### Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 401 | `unauthorized` | Missing or invalid Bearer token |

**Notes:**
- Response includes `Cache-Control: no-store` and `Pragma: no-cache` headers.

---

## POST /forgot-password

Request a password reset email. Always returns 200 to prevent email enumeration.

**Auth required:** No

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | Yes | The email address to send the reset link to |

### Request

```bash
curl -X POST https://your-rampart-instance/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email": "jane@example.com"}'
```

### Response (200 OK)

```json
{
  "message": "If an account with that email exists, a password reset link has been sent."
}
```

**Notes:**
- Always returns 200 regardless of whether the email exists. This prevents email enumeration.
- The reset token expires after 1 hour.
- Requires SMTP to be configured for email delivery.

---

## POST /reset-password

Reset a user's password using a reset token from the forgot-password email.

**Auth required:** No

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | The reset token from the email link |
| `new_password` | string | Yes | The new password (must meet organization password policy) |

### Request

```bash
curl -X POST https://your-rampart-instance/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "token": "a1b2c3d4e5f6...",
    "new_password": "NewSecureP@ssw0rd!"
  }'
```

### Response (200 OK)

```json
{
  "message": "Password has been reset successfully. You can now log in with your new password."
}
```

### Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `invalid_request` | Missing token or new_password |
| 400 | `invalid_token` | Token is invalid, expired, or already used |
| 422 | `validation_error` | New password does not meet the organization's password policy |

**Notes:**
- All existing sessions for the user are revoked when the password is reset.
- The reset token is single-use.

---

## POST /verify-email/send

Send an email verification link. Always returns 200 to prevent email enumeration.

**Auth required:** No

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | Yes | The email address to send the verification link to |

### Request

```bash
curl -X POST https://your-rampart-instance/verify-email/send \
  -H "Content-Type: application/json" \
  -d '{"email": "jane@example.com"}'
```

### Response (200 OK)

```json
{
  "message": "If an account with that email exists, a verification link has been sent."
}
```

---

## GET /verify-email

Verify an email address using the token from the verification email.

**Auth required:** No

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes | The verification token from the email link |

### Request

```bash
curl -X GET "https://your-rampart-instance/verify-email?token=a1b2c3d4e5f6..."
```

### Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `invalid_token` | Token is invalid, expired, or already used |

**Notes:**
- The verification token expires after 24 hours.
- The token is single-use.

---

## MFA Endpoints

### POST /mfa/totp/enroll

Begin TOTP enrollment for the authenticated user. Returns a TOTP secret and provisioning URI for generating a QR code.

**Auth required:** Yes (Bearer token)

```bash
curl -X POST https://your-rampart-instance/mfa/totp/enroll \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json"
```

### POST /mfa/totp/verify-setup

Confirm TOTP setup by providing a valid TOTP code from the authenticator app.

**Auth required:** Yes (Bearer token)

```bash
curl -X POST https://your-rampart-instance/mfa/totp/verify-setup \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"code": "123456"}'
```

### POST /mfa/totp/disable

Disable TOTP for the authenticated user.

**Auth required:** Yes (Bearer token)

```bash
curl -X POST https://your-rampart-instance/mfa/totp/disable \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"code": "123456"}'
```

### POST /mfa/totp/verify

Verify a TOTP code during the login flow. This endpoint is called after `POST /login` returns an MFA challenge.

**Auth required:** No (uses MFA token from login response)

```bash
curl -X POST https://your-rampart-instance/mfa/totp/verify \
  -H "Content-Type: application/json" \
  -d '{
    "mfa_token": "eyJhbGciOiJSUzI1NiIs...",
    "code": "123456"
  }'
```

**Response (200 OK):** Returns the same token response as a successful login (access token, refresh token, user object).

---

## WebAuthn / Passkey Endpoints

### POST /mfa/webauthn/register/begin

Begin registering a new WebAuthn credential (passkey) for the authenticated user.

**Auth required:** Yes (Bearer token)

### POST /mfa/webauthn/register/complete

Complete WebAuthn credential registration by submitting the authenticator's response.

**Auth required:** Yes (Bearer token)

### GET /mfa/webauthn/credentials

List all WebAuthn credentials for the authenticated user.

**Auth required:** Yes (Bearer token)

```bash
curl -X GET https://your-rampart-instance/mfa/webauthn/credentials \
  -H "Authorization: Bearer <access_token>"
```

### DELETE /mfa/webauthn/credentials/\{id\}

Delete a specific WebAuthn credential.

**Auth required:** Yes (Bearer token)

```bash
curl -X DELETE https://your-rampart-instance/mfa/webauthn/credentials/credential-id-here \
  -H "Authorization: Bearer <access_token>"
```

### POST /mfa/webauthn/login/begin

Begin a WebAuthn login challenge. Called after `POST /login` returns an MFA challenge with `webauthn` in `mfa_methods`.

**Auth required:** No (uses MFA token from login response)

```bash
curl -X POST https://your-rampart-instance/mfa/webauthn/login/begin \
  -H "Content-Type: application/json" \
  -d '{"mfa_token": "eyJhbGciOiJSUzI1NiIs..."}'
```

### POST /mfa/webauthn/login/complete

Complete the WebAuthn login challenge by submitting the authenticator's assertion.

**Auth required:** No (uses MFA token)

**Response (200 OK):** Returns the same token response as a successful login.

---

## Security Considerations

- **Timing safety:** Registration and login responses take a minimum of 250ms to prevent user enumeration via timing analysis.
- **Account lockout:** After exceeding the maximum failed login attempts (default: 5), the account is locked for a configurable duration (default: 15 minutes).
- **Refresh token rotation:** Each refresh token is single-use. Reuse of an old token triggers session revocation (replay detection).
- **Email enumeration prevention:** The `/forgot-password` and `/verify-email/send` endpoints always return 200 regardless of whether the email exists.
- **Password policy:** Passwords are validated against the organization's password policy (configurable min length, uppercase, lowercase, digit, special character requirements).
