# Device Authorization Flow

Authentication for input-constrained devices that can't easily handle browser redirects — CLIs, smart TVs, game consoles, IoT devices. Follows RFC 8628.

## Sequence Diagram

```mermaid
sequenceDiagram
    participant Device as Device / CLI
    participant User as End User (Browser)
    participant Rampart as Rampart Server
    participant DB as PostgreSQL
    participant Cache as Redis

    Note over Device: Step 1 — Request device code

    Device->>Rampart: POST /oauth/device<br/>client_id=rampart-cli<br/>&scope=openid profile

    Rampart->>DB: Validate client_id and allowed grant types
    DB-->>Rampart: Client record

    Rampart->>Rampart: Generate device_code (cryptographic random, 32 bytes)
    Rampart->>Rampart: Generate user_code (8 chars, easy to type: WDJB-MJHT)

    Rampart->>Cache: Store device authorization<br/>{device_code, user_code, client_id, scope, status: "pending"}<br/>TTL: 10 minutes

    Rampart->>DB: Log event: device_flow.started

    Rampart-->>Device: 200 {<br/>  device_code, user_code,<br/>  verification_uri: "https://auth.example.com/device",<br/>  verification_uri_complete: "https://auth.example.com/device?user_code=WDJB-MJHT",<br/>  expires_in: 600,<br/>  interval: 5<br/>}

    Note over Device,User: Step 2 — Device displays code to user

    Device->>Device: Display to user:<br/>"Go to https://auth.example.com/device<br/>and enter code: WDJB-MJHT"

    Note over User,Rampart: Step 3 — User authenticates in browser

    User->>Rampart: GET /device (or scan QR code for verification_uri_complete)
    Rampart-->>User: Code entry page

    User->>Rampart: POST /device/verify<br/>{user_code: "WDJB-MJHT"}

    Rampart->>Cache: Lookup by user_code
    Cache-->>Rampart: Device authorization record

    alt Code not found or expired
        Rampart-->>User: "Invalid or expired code"
    end

    alt User not authenticated
        Rampart-->>User: Redirect to login page
        User->>Rampart: Login (password + optional MFA)
    end

    Rampart-->>User: Consent page<br/>"CLI tool is requesting access to: profile, email"

    User->>Rampart: Approve

    Rampart->>Cache: Update device authorization<br/>{status: "approved", user_id: usr_abc123}
    Rampart->>DB: Log event: device_flow.approved

    Rampart-->>User: "You have approved the device. You can close this window."

    Note over Device,Rampart: Step 4 — Device polls for token

    loop Every 5 seconds (per interval)
        Device->>Rampart: POST /oauth/token<br/>grant_type=urn:ietf:params:oauth:grant-type:device_code<br/>&device_code=DEVICE_CODE<br/>&client_id=rampart-cli

        Rampart->>Cache: Check device authorization status

        alt Status: pending
            Rampart-->>Device: 400 {"error": "authorization_pending"}
        end

        alt Status: denied
            Rampart-->>Device: 400 {"error": "access_denied"}
        end

        alt Expired
            Rampart-->>Device: 400 {"error": "expired_token"}
        end

        alt Polling too fast
            Rampart-->>Device: 400 {"error": "slow_down"}
        end
    end

    Note over Rampart: Status: approved — issue tokens

    Rampart->>Cache: Delete device authorization (single-use)
    Rampart->>DB: Fetch signing key
    Rampart->>Rampart: Generate access_token + id_token
    Rampart->>DB: Store refresh_token (hashed)
    Rampart->>DB: Log event: token.issued (device_flow)

    Rampart-->>Device: 200 {access_token, id_token, refresh_token, expires_in}

    Note over Device: Device is now authenticated
```

## User Code Format

The user code is designed for easy typing on constrained input devices:

- 8 characters, split into two groups: `WDJB-MJHT`
- Uppercase letters only (no ambiguous characters: 0/O, 1/I/L removed)
- Character set: `BCDFGHJKMNPQRSTVWXZ` (20 characters, ~34 bits of entropy)
- Hyphen separator for readability

## Polling Behavior

| Response | Action |
|----------|--------|
| `authorization_pending` | User hasn't completed auth yet. Wait `interval` seconds and retry. |
| `slow_down` | Polling too fast. Increase interval by 5 seconds. |
| `access_denied` | User denied the request. Stop polling. |
| `expired_token` | Device code expired. Start over. |
| Token response | Success. Stop polling. |

## Security Considerations

| Concern | Mitigation |
|---------|------------|
| User code brute force | Rate limiting on code entry, limited character set size is sufficient for short-lived codes |
| Device code theft | Codes expire after 10 minutes, single-use |
| Polling abuse | `slow_down` response, rate limiting per client |
| Phishing (fake verification URI) | Users should verify the domain in their browser |
| Session fixation | User must authenticate fresh — no pre-existing session reuse |

## Use Cases

| Device | Example |
|--------|---------|
| **CLI tools** | `rampart-cli login` — developer authenticates locally |
| **Smart TVs** | Streaming app login via phone |
| **IoT devices** | Devices with no browser capability |
| **CI/CD pipelines** | Interactive auth for initial setup |
