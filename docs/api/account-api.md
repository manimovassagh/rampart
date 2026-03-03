# Account API

Self-service API for authenticated end users to manage their own account. All endpoints require a valid user bearer token.

```
Base: /api/v1/account
Auth: Bearer <user_access_token>
```

---

## Profile

### Get Profile

```http
GET /api/v1/account/profile
Authorization: Bearer <access_token>
```

**Response** `200 OK`

```json
{
  "id": "usr_abc123",
  "username": "jane.doe",
  "email": "jane@example.com",
  "email_verified": true,
  "given_name": "Jane",
  "family_name": "Doe",
  "picture": "https://example.com/jane.jpg",
  "phone_number": "+1234567890",
  "phone_number_verified": false,
  "mfa_enabled": true,
  "org_id": "org_abc123",
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-03-01T14:30:00Z"
}
```

### Update Profile

```http
PUT /api/v1/account/profile
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "given_name": "Jane",
  "family_name": "Smith",
  "picture": "https://example.com/jane-new.jpg",
  "phone_number": "+1234567890"
}
```

Only the provided fields are updated. Users cannot change their own `email`, `username`, or `enabled` status through this endpoint — those require admin action or a verified email change flow.

**Response** `200 OK` — returns the updated profile.

---

## Password

### Change Password

```http
PUT /api/v1/account/password
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "current_password": "old-password-123",
  "new_password": "new-password-456"
}
```

The new password must meet the organization's password policy. The current password is required to prevent unauthorized changes from stolen tokens.

**Response** `204 No Content`

**Error Responses**

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `rampart_weak_password` | New password doesn't meet policy |
| 401 | `rampart_invalid_password` | Current password is incorrect |

---

## Sessions

### List Own Sessions

```http
GET /api/v1/account/sessions
Authorization: Bearer <access_token>
```

**Response** `200 OK`

```json
{
  "data": [
    {
      "id": "ses_abc123",
      "ip_address": "203.0.113.42",
      "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)...",
      "started_at": "2026-03-01T14:30:00Z",
      "last_active_at": "2026-03-03T09:15:00Z",
      "expires_at": "2026-03-03T14:30:00Z",
      "current": true
    },
    {
      "id": "ses_def456",
      "ip_address": "198.51.100.1",
      "user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)...",
      "started_at": "2026-02-28T08:00:00Z",
      "last_active_at": "2026-03-02T18:00:00Z",
      "expires_at": "2026-03-03T08:00:00Z",
      "current": false
    }
  ]
}
```

The `current` field indicates the session making this request.

### Revoke Own Session

```http
DELETE /api/v1/account/sessions/{session_id}
Authorization: Bearer <access_token>
```

Users can revoke any of their own sessions except the current one (use the logout endpoint for that).

**Response** `204 No Content`

---

## Multi-Factor Authentication (MFA)

### Enroll MFA

```http
POST /api/v1/account/mfa/enroll
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "type": "totp"
}
```

| Type | Description |
|------|-------------|
| `totp` | Time-based One-Time Password (Google Authenticator, Authy, etc.) |
| `webauthn` | WebAuthn / Passkey (future) |

**Response** `200 OK`

```json
{
  "type": "totp",
  "secret": "JBSWY3DPEHPK3PXP",
  "uri": "otpauth://totp/Rampart:jane@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Rampart",
  "qr_code": "data:image/png;base64,iVBOR...",
  "recovery_codes": [
    "ABCD-1234-EFGH",
    "IJKL-5678-MNOP",
    "QRST-9012-UVWX",
    "YZAB-3456-CDEF",
    "GHIJ-7890-KLMN",
    "OPQR-1234-STUV",
    "WXYZ-5678-ABCD",
    "EFGH-9012-IJKL"
  ]
}
```

MFA is not active until verified (see next endpoint). Recovery codes are shown only once.

### Verify MFA Setup

```http
POST /api/v1/account/mfa/verify
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "code": "123456"
}
```

Verifies the TOTP setup by confirming the user can generate a valid code. After successful verification, MFA is active on the account.

**Response** `204 No Content`

**Error Response**

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `rampart_invalid_mfa_code` | The provided code is incorrect |

### Remove MFA

```http
DELETE /api/v1/account/mfa
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "current_password": "my-password-123"
}
```

Requires the user's current password as confirmation. This removes all MFA methods from the account.

**Response** `204 No Content`
