# User Registration Flow

New user self-registration with email verification. Registration can be initiated via the login page or directly through the Admin API.

## Self-Registration Sequence

```mermaid
sequenceDiagram
    participant User as End User (Browser)
    participant App as Client Application
    participant Rampart as Rampart Server
    participant DB as PostgreSQL
    participant Email as Email Service

    Note over User,App: User clicks "Sign up" on the application

    App->>Rampart: GET /oauth/authorize?...&prompt=create
    Rampart-->>User: Registration page

    User->>Rampart: POST /register<br/>{username, email, password, given_name, family_name}

    Note over Rampart: Validate input

    Rampart->>Rampart: Validate email format
    Rampart->>Rampart: Validate password against org policy<br/>(min length, complexity, breach check)
    Rampart->>Rampart: Validate username format

    alt Validation fails
        Rampart-->>User: 400 + field-level errors
    end

    Rampart->>DB: Check if email or username already exists
    DB-->>Rampart: Exists / not exists

    alt Already exists
        Rampart-->>User: 409 {"error": "rampart_conflict",<br/>"error_description": "Email already registered"}
        Note over Rampart: Timing-safe response to prevent user enumeration
    end

    Note over Rampart,DB: Create user

    Rampart->>Rampart: Hash password (argon2id)
    Rampart->>DB: INSERT user (email_verified=false, enabled=true)
    Rampart->>DB: Log event: user.created

    Note over Rampart,Email: Email verification

    Rampart->>Rampart: Generate verification token (cryptographic random, 32 bytes)
    Rampart->>DB: Store verification token (hashed, TTL: 24 hours)
    Rampart->>Email: Send verification email<br/>Link: https://auth.example.com/verify?token=TOKEN

    Rampart-->>User: Registration success page<br/>"Check your email to verify your account"

    Note over User,Rampart: User clicks verification link

    User->>Rampart: GET /verify?token=TOKEN

    Rampart->>DB: Lookup verification token (hashed)

    alt Token invalid or expired
        Rampart-->>User: Error page + "Resend verification email" link
    end

    Rampart->>DB: UPDATE user SET email_verified=true
    Rampart->>DB: DELETE verification token
    Rampart->>DB: Log event: user.email_verified

    Rampart-->>User: Email verified — redirect to login

    Note over User,App: User can now complete the OAuth flow
```

## Admin-Created User

Admins can create users directly via the API. These users may bypass email verification if the admin marks `email_verified: true`.

```mermaid
sequenceDiagram
    participant Admin as Admin User
    participant Rampart as Rampart Server
    participant DB as PostgreSQL
    participant Email as Email Service

    Admin->>Rampart: POST /api/v1/admin/users<br/>{username, email, password, org_id, email_verified: false}

    Rampart->>Rampart: Validate input + password policy
    Rampart->>Rampart: Hash password (argon2id)
    Rampart->>DB: INSERT user
    Rampart->>DB: Log event: user.created (by_admin)

    alt email_verified is false
        Rampart->>Email: Send verification email
    end

    Rampart-->>Admin: 201 {user object}

    Note over Admin: Optionally set temporary password
    Admin->>Rampart: POST /api/v1/admin/users/{id}/reset-password<br/>{password: "temp123", temporary: true}

    Rampart->>DB: Update password hash, set must_change=true
    Rampart-->>Admin: 204 No Content

    Note over Admin: User must change password on first login
```

## Password Policy Enforcement

The registration flow enforces the organization's password policy:

| Rule | Default | Configurable |
|------|---------|--------------|
| Minimum length | 12 characters | Yes |
| Require uppercase | Yes | Yes |
| Require number | Yes | Yes |
| Require special character | Yes | Yes |
| Breached password check | Yes (via k-anonymity API) | Yes |
| Password history | Last 5 passwords | Yes |

## Anti-Abuse Measures

| Threat | Mitigation |
|--------|------------|
| User enumeration | Timing-safe responses — same latency for "exists" and "not exists" |
| Registration spam | Rate limiting on registration endpoint |
| Email bombing | Rate limit on verification emails per address |
| Weak passwords | Password policy enforcement + breached password check |
| Bot registration | CAPTCHA integration point (pluggable) |
