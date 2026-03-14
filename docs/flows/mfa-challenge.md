# MFA Challenge Flow

Multi-factor authentication during login. When a user has MFA enabled, they must provide a second factor after successful password verification.

## Login with MFA — Sequence Diagram

```mermaid
sequenceDiagram
    participant User as End User (Browser)
    participant Rampart as Rampart Server
    participant DB as PostgreSQL
    participant Cache as PostgreSQL Cache

    Note over User,Rampart: User has already entered correct password

    Rampart->>DB: Check if user has MFA enabled
    DB-->>Rampart: MFA methods: [totp]

    Rampart->>Cache: Store partial auth session<br/>{user_id, auth_step: "mfa_required", methods: ["totp"]}<br/>TTL: 5 minutes

    Rampart-->>User: MFA challenge page<br/>Available methods: TOTP

    alt TOTP Challenge
        User->>User: Open authenticator app, read 6-digit code
        User->>Rampart: POST /mfa/challenge<br/>{session_id, type: "totp", code: "123456"}

        Rampart->>Cache: Retrieve partial auth session
        Cache-->>Rampart: {user_id, auth_step: "mfa_required"}

        Rampart->>DB: Fetch user's TOTP secret (encrypted)
        Rampart->>Rampart: Decrypt TOTP secret
        Rampart->>Rampart: Validate TOTP code<br/>(current window ± 1 step for clock drift)

        alt Code invalid
            Rampart->>Cache: Increment attempt counter
            Rampart->>DB: Log event: user.mfa_failed

            alt Too many attempts (> 5)
                Rampart->>Cache: Delete partial auth session
                Rampart-->>User: "Too many failed attempts. Please log in again."
            else
                Rampart-->>User: "Invalid code. Please try again."<br/>(show remaining attempts)
            end
        end
    end

    alt Recovery Code
        User->>Rampart: POST /mfa/challenge<br/>{session_id, type: "recovery", code: "ABCD-1234-EFGH"}

        Rampart->>DB: Fetch recovery codes (hashed)
        Rampart->>Rampart: Compare against each stored hash (bcrypt)

        alt Code matches
            Rampart->>DB: Delete used recovery code
            Note over Rampart: Recovery codes are single-use
        else Code invalid
            Rampart-->>User: "Invalid recovery code"
        end
    end

    Note over Rampart: MFA verified successfully

    Rampart->>Cache: Delete partial auth session
    Rampart->>Cache: Create full session
    Rampart->>DB: Log event: user.mfa_success
    Rampart->>DB: Update user.last_login_at

    Rampart-->>User: 302 Redirect to OAuth authorize (continue flow)
```

## MFA Enrollment — Sequence Diagram

```mermaid
sequenceDiagram
    participant User as End User
    participant Rampart as Rampart Server
    participant DB as PostgreSQL

    User->>Rampart: POST /api/v1/account/mfa/enroll<br/>{type: "totp"}

    Rampart->>Rampart: Generate TOTP secret (20 bytes, base32)
    Rampart->>Rampart: Generate otpauth:// URI
    Rampart->>Rampart: Generate QR code (PNG, base64)
    Rampart->>Rampart: Generate 8 recovery codes<br/>(format: XXXX-XXXX-XXXX, cryptographic random)

    Rampart->>DB: Store MFA method (secret encrypted, verified=false)<br/>Store recovery codes (individually bcrypt-hashed)

    Rampart-->>User: {secret, uri, qr_code, recovery_codes}

    Note over User: User scans QR code with authenticator app
    Note over User: User saves recovery codes securely

    User->>Rampart: POST /api/v1/account/mfa/verify<br/>{code: "123456"}

    Rampart->>DB: Fetch MFA method (verified=false)
    Rampart->>Rampart: Decrypt secret, validate TOTP code

    alt Code valid
        Rampart->>DB: UPDATE mfa_method SET verified=true
        Rampart->>DB: UPDATE user SET mfa_enabled=true
        Rampart->>DB: Log event: user.mfa_enrolled
        Rampart-->>User: 204 No Content (MFA is now active)
    else Code invalid
        Rampart-->>User: 400 {"error": "rampart_invalid_mfa_code"}
    end
```

## Supported MFA Methods

| Method | Status | Description |
|--------|--------|-------------|
| **TOTP** | Phase 1 | Time-based One-Time Password (RFC 6238). Works with Google Authenticator, Authy, 1Password, etc. |
| **WebAuthn / Passkeys** | Phase 2 | Hardware security keys and platform authenticators (fingerprint, face). |
| **Recovery Codes** | Phase 1 | 8 single-use backup codes for account recovery. |

## TOTP Parameters

| Parameter | Value |
|-----------|-------|
| Algorithm | SHA-1 (per RFC 6238 for compatibility) |
| Digits | 6 |
| Period | 30 seconds |
| Validation window | ±1 step (allows 30s clock drift) |
| Secret length | 20 bytes (160 bits) |

## Security Considerations

| Concern | Mitigation |
|---------|------------|
| Brute force MFA codes | Max 5 attempts per challenge session, then force re-login |
| Recovery code theft | Codes are individually bcrypt-hashed, single-use |
| TOTP secret exposure | Encrypted at rest (AES-256-GCM), shown only during enrollment |
| Replay attack | TOTP codes valid only within the current ±1 time window |
| MFA bypass | Partial auth session expires after 5 minutes |
| Account lockout | Failed MFA doesn't lock the account — but logs the event and requires re-entering password |

## Organization MFA Policies

Admins can configure MFA policy per organization:

| Policy | Behavior |
|--------|----------|
| `disabled` | MFA not available |
| `optional` (default) | Users can self-enroll via Account API |
| `required` | Users must enroll MFA. Redirected to enrollment on login if not enrolled. |
