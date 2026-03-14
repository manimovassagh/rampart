# Token Refresh Flow

Exchanges a refresh token for a new access token (and optionally a new refresh token). This allows long-lived sessions without requiring the user to re-authenticate.

## Sequence Diagram

```mermaid
sequenceDiagram
    participant App as Client Application
    participant Rampart as Rampart Server
    participant DB as PostgreSQL
    participant Cache as PostgreSQL Cache

    Note over App: Access token expired or about to expire

    App->>Rampart: POST /oauth/token<br/>grant_type=refresh_token<br/>&refresh_token=CURRENT_REFRESH_TOKEN<br/>&client_id=my-spa

    Note over Rampart: Validate the refresh token

    Rampart->>Rampart: Hash the refresh token (SHA-256)
    Rampart->>DB: Lookup refresh token by hash
    DB-->>Rampart: Token record (user_id, client_id, scope, revoked, expires_at)

    alt Token not found
        Rampart-->>App: 400 {"error": "invalid_grant",<br/>"error_description": "Refresh token is invalid"}
    end

    alt Token revoked
        Note over Rampart,DB: Possible token theft — revoke all tokens for this user/client
        Rampart->>DB: Revoke ALL refresh tokens for user + client
        Rampart->>DB: Log event: token.revoked (suspected_theft)
        Rampart-->>App: 400 {"error": "invalid_grant",<br/>"error_description": "Refresh token has been revoked"}
    end

    alt Token expired
        Rampart-->>App: 400 {"error": "invalid_grant",<br/>"error_description": "Refresh token has expired"}
    end

    Rampart->>DB: Verify client_id matches token's client
    Rampart->>DB: Verify user still exists and is enabled

    alt User disabled or deleted
        Rampart->>DB: Revoke refresh token
        Rampart-->>App: 400 {"error": "invalid_grant"}
    end

    Note over Rampart: Refresh token rotation

    Rampart->>DB: Mark current refresh token as revoked
    Rampart->>DB: Generate and store new refresh token (hashed)

    Note over Rampart: Generate new access token

    Rampart->>DB: Fetch signing key
    Rampart->>DB: Fetch user claims (for ID token if openid scope)
    Rampart->>Rampart: Generate new access_token (JWT)
    Rampart->>Rampart: Generate new id_token (if openid scope)

    Rampart->>Cache: Update session last_active_at
    Rampart->>DB: Log event: token.refreshed

    Rampart-->>App: 200 {access_token, refresh_token, id_token, expires_in}

    Note over App: Replace stored tokens with new ones
```

## Refresh Token Rotation

Rampart implements **refresh token rotation** as a security best practice:

1. Every time a refresh token is used, it is **invalidated** and a **new refresh token** is issued.
2. If a previously-used (revoked) refresh token is presented, Rampart treats this as **token theft** and revokes the entire token family.

This limits the damage of a leaked refresh token — an attacker can use it at most once before the legitimate client's next refresh detects the theft.

```
Refresh #1: RT_A → issues RT_B (RT_A marked revoked)
Refresh #2: RT_B → issues RT_C (RT_B marked revoked)

If attacker uses RT_A again:
  → RT_A is already revoked → THEFT DETECTED
  → ALL tokens (RT_B, RT_C) revoked
  → User must re-authenticate
```

## Token Lifetimes

| Token | Default Lifetime | Configurable |
|-------|-----------------|--------------|
| Access token | 15 minutes (900s) | Yes, via `RAMPART_ACCESS_TOKEN_TTL` |
| Refresh token | 7 days (604800s) | Yes, via `RAMPART_REFRESH_TOKEN_TTL` |
| ID token | Same as access token | Yes |

## Security Considerations

| Concern | Mitigation |
|---------|------------|
| Refresh token theft | Token rotation — each token is single-use |
| Replay of old token | Revokes entire token family on reuse detection |
| Stolen token window | Short access token lifetime limits exposure |
| Disabled user | User status checked on every refresh |
| Client mismatch | Refresh token is bound to the issuing client |

## Error Responses

| Scenario | Error Code | Description |
|----------|-----------|-------------|
| Token not found | `invalid_grant` | The token doesn't exist |
| Token revoked | `invalid_grant` | Token was already used (rotation) or explicitly revoked |
| Token expired | `invalid_grant` | Past the token's expiration time |
| User disabled | `invalid_grant` | The user account has been disabled |
| Wrong client | `invalid_grant` | Token was issued to a different client |
