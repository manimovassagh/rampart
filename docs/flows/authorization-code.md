# Authorization Code Flow + PKCE

The primary authentication flow for browser-based and mobile applications. Uses PKCE (RFC 7636) to protect against authorization code interception — required for all public clients, recommended for confidential clients.

## Sequence Diagram

```mermaid
sequenceDiagram
    participant User as End User (Browser)
    participant App as Client Application
    participant Rampart as Rampart Server
    participant DB as PostgreSQL
    participant Cache as PostgreSQL Cache

    Note over App: Step 1 — Generate PKCE challenge
    App->>App: Generate code_verifier (random 43-128 chars)
    App->>App: code_challenge = BASE64URL(SHA256(code_verifier))

    Note over User,Rampart: Step 2 — Authorization request
    App->>Rampart: GET /oauth/authorize?response_type=code<br/>&client_id=my-spa<br/>&redirect_uri=https://app.example.com/callback<br/>&scope=openid profile email<br/>&state=random_state<br/>&code_challenge=...&code_challenge_method=S256<br/>&nonce=random_nonce

    Rampart->>DB: Validate client_id, redirect_uri, scopes
    DB-->>Rampart: Client record

    alt Client invalid or redirect_uri mismatch
        Rampart-->>User: Error page (never redirect with error to unregistered URI)
    end

    alt User not authenticated
        Rampart-->>User: Redirect to login page
        User->>Rampart: POST credentials (username + password)
        Rampart->>DB: Lookup user, verify password (argon2id)
        DB-->>Rampart: User record

        alt MFA enabled
            Rampart-->>User: MFA challenge page
            User->>Rampart: POST MFA code (TOTP)
            Rampart->>DB: Verify MFA
        end

        Rampart->>Cache: Create session
    end

    alt Consent not yet granted
        Rampart-->>User: Consent page (requested scopes)
        User->>Rampart: Approve consent
    end

    Note over Rampart,Cache: Step 3 — Issue authorization code
    Rampart->>Cache: Store auth code + code_challenge + user_id + scopes<br/>(TTL: 10 minutes, single-use)
    Rampart-->>User: 302 Redirect to redirect_uri?code=AUTH_CODE&state=random_state

    Note over App,Rampart: Step 4 — Exchange code for tokens
    App->>App: Verify state matches
    App->>Rampart: POST /oauth/token<br/>grant_type=authorization_code<br/>&code=AUTH_CODE<br/>&redirect_uri=https://app.example.com/callback<br/>&client_id=my-spa<br/>&code_verifier=original_verifier

    Rampart->>Cache: Retrieve and delete auth code (single-use)
    Cache-->>Rampart: code_challenge, user_id, scopes

    Rampart->>Rampart: Verify: BASE64URL(SHA256(code_verifier)) == code_challenge
    Rampart->>Rampart: Verify: redirect_uri matches original request

    alt PKCE verification fails
        Rampart-->>App: 400 {"error": "invalid_grant"}
    end

    Note over Rampart,DB: Step 5 — Generate tokens
    Rampart->>DB: Fetch signing key
    Rampart->>Rampart: Generate access_token (JWT, signed RS256/ES256)
    Rampart->>Rampart: Generate id_token (JWT with nonce, user claims)
    Rampart->>DB: Store refresh_token (hashed)
    Rampart->>DB: Log event: token.issued

    Rampart-->>App: 200 {access_token, id_token, refresh_token, expires_in}

    Note over App: Step 6 — Use tokens
    App->>App: Validate id_token (signature, nonce, exp, iss, aud)
    App->>Rampart: GET /oidc/userinfo (Authorization: Bearer access_token)
    Rampart-->>App: {sub, name, email, ...}
```

## Security Considerations

| Concern | Mitigation |
|---------|------------|
| Authorization code interception | PKCE with S256 required for all public clients |
| CSRF on callback | `state` parameter verified by client |
| Open redirect | `redirect_uri` must exactly match registered URI |
| Code replay | Authorization codes are single-use, deleted on first exchange |
| Code expiry | Authorization codes expire after 10 minutes |
| Token binding | `nonce` in ID token binds to the original auth request |
| Client impersonation | Confidential clients must authenticate at token endpoint |

## Parameters Reference

### Authorization Request

| Parameter | Required | Description |
|-----------|----------|-------------|
| `response_type` | Yes | Must be `code` |
| `client_id` | Yes | Registered client identifier |
| `redirect_uri` | Yes | Must exactly match a registered redirect URI |
| `scope` | Yes | Space-delimited. Include `openid` for OIDC. |
| `state` | Recommended | Random value for CSRF protection |
| `code_challenge` | Required (public) | `BASE64URL(SHA256(code_verifier))` |
| `code_challenge_method` | Required with challenge | Must be `S256` (plain not supported) |
| `nonce` | Recommended | Random value bound to ID token |
| `prompt` | Optional | `none`, `login`, `consent`, `select_account` |

### Token Request

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | `authorization_code` |
| `code` | Yes | The authorization code from step 3 |
| `redirect_uri` | Yes | Must match the original authorization request |
| `client_id` | Yes | The client identifier |
| `code_verifier` | Required (public) | The original PKCE verifier |
