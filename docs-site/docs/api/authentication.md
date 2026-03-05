---
sidebar_position: 2
title: Authentication Endpoints
description: Complete reference for Rampart's OAuth 2.0 authentication endpoints -- token issuance, authorization, revocation, and introspection with full request and response examples.
---

# Authentication Endpoints

This page covers the core authentication endpoints used to obtain, validate, and revoke tokens. These endpoints implement the OAuth 2.0 and OpenID Connect specifications (RFC 6749, RFC 7009, RFC 7662).

## POST /oauth/token

The token endpoint issues access tokens, ID tokens, and refresh tokens. It supports multiple grant types depending on the use case.

**Content-Type:** `application/x-www-form-urlencoded`

---

### Authorization Code Grant

Used by web applications and native apps after the user completes the authorization flow. This is the most common grant type for user-facing applications.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `authorization_code` |
| `code` | string | Yes | The authorization code received from the authorize endpoint |
| `redirect_uri` | string | Yes | Must match the redirect URI used in the authorization request |
| `client_id` | string | Yes | The client identifier |
| `client_secret` | string | Conditional | Required for confidential clients |
| `code_verifier` | string | Conditional | PKCE code verifier (required for public clients, recommended for all) |

**Request (confidential client):**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=SplxlOBeZQQYbYS6WxSbIA" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "client_id=my-web-app" \
  -d "client_secret=my-client-secret"
```

**Request (public client with PKCE):**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=SplxlOBeZQQYbYS6WxSbIA" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "client_id=my-spa" \
  -d "code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InJhbXBhcnQtMSJ9.eyJpc3MiOiJodHRwczovL3lvdXItcmFtcGFydC1pbnN0YW5jZSIsInN1YiI6InVzcl8xMjM0NTY3ODkwIiwiYXVkIjoibXktd2ViLWFwcCIsImV4cCI6MTcwOTUxNDAwMCwiaWF0IjoxNzA5NTEwNDAwLCJzY29wZSI6Im9wZW5pZCBwcm9maWxlIGVtYWlsIn0.signature",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4gZXhhbXBsZQ",
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InJhbXBhcnQtMSJ9.eyJpc3MiOiJodHRwczovL3lvdXItcmFtcGFydC1pbnN0YW5jZSIsInN1YiI6InVzcl8xMjM0NTY3ODkwIiwiYXVkIjoibXktd2ViLWFwcCIsImV4cCI6MTcwOTUxNDAwMCwiaWF0IjoxNzA5NTEwNDAwLCJub25jZSI6ImFiYzEyMyIsIm5hbWUiOiJKYW5lIERvZSIsImVtYWlsIjoiamFuZUBleGFtcGxlLmNvbSJ9.signature",
  "scope": "openid profile email"
}
```

The `id_token` is included only when the `openid` scope was requested. The `refresh_token` is included only when the client is configured to support refresh tokens.

---

### Client Credentials Grant

Used for service-to-service authentication where no user is involved. The client authenticates directly with its own credentials.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `client_credentials` |
| `client_id` | string | Yes | The client identifier |
| `client_secret` | string | Yes | The client secret |
| `scope` | string | No | Space-separated list of requested scopes |

**Request (body parameters):**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=billing-service" \
  -d "client_secret=svc-secret-abc123" \
  -d "scope=users:read invoices:write"
```

**Request (HTTP Basic authentication):**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "billing-service:svc-secret-abc123" \
  -d "grant_type=client_credentials" \
  -d "scope=users:read invoices:write"
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InJhbXBhcnQtMSJ9.eyJpc3MiOiJodHRwczovL3lvdXItcmFtcGFydC1pbnN0YW5jZSIsInN1YiI6ImJpbGxpbmctc2VydmljZSIsImF1ZCI6Imh0dHBzOi8veW91ci1yYW1wYXJ0LWluc3RhbmNlIiwiZXhwIjoxNzA5NTE0MDAwLCJpYXQiOjE3MDk1MTA0MDAsInNjb3BlIjoidXNlcnM6cmVhZCBpbnZvaWNlczp3cml0ZSJ9.signature",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "users:read invoices:write"
}
```

Client credentials grants do not return a refresh token or an ID token, since there is no end-user session.

---

### Refresh Token Grant

Used to obtain a new access token using a previously issued refresh token, without requiring the user to re-authenticate.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `refresh_token` |
| `refresh_token` | string | Yes | The refresh token issued during the original token request |
| `client_id` | string | Yes | The client identifier |
| `client_secret` | string | Conditional | Required for confidential clients |
| `scope` | string | No | Subset of originally granted scopes (defaults to original scopes if omitted) |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4gZXhhbXBsZQ" \
  -d "client_id=my-web-app" \
  -d "client_secret=my-client-secret"
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...new-access-token",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "bmV3LXJlZnJlc2gtdG9rZW4tZXhhbXBsZQ",
  "id_token": "eyJhbGciOiJSUzI1NiIs...new-id-token",
  "scope": "openid profile email"
}
```

**Refresh token rotation:** Rampart implements refresh token rotation by default. Each time a refresh token is used, a new refresh token is issued and the previous one is invalidated. If a previously used refresh token is presented again, Rampart treats it as a potential token theft and revokes all tokens in the chain (replay detection).

---

### Resource Owner Password Credentials Grant

Used for trusted first-party applications where the user provides credentials directly to the client. This grant type is discouraged for third-party applications and is **disabled by default**.

:::caution
The password grant is considered a legacy flow. Use Authorization Code with PKCE instead whenever possible. Enable this grant only for trusted first-party clients that cannot use a browser-based flow.
:::

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `password` |
| `username` | string | Yes | The user's username or email |
| `password` | string | Yes | The user's password |
| `client_id` | string | Yes | The client identifier |
| `client_secret` | string | Conditional | Required for confidential clients |
| `scope` | string | No | Space-separated list of requested scopes |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password" \
  -d "username=jane.doe" \
  -d "password=SecureP@ssw0rd!" \
  -d "client_id=admin-cli" \
  -d "client_secret=cli-secret" \
  -d "scope=openid profile"
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "cmVmcmVzaC10b2tlbi1leGFtcGxl",
  "id_token": "eyJhbGciOiJSUzI1NiIs...",
  "scope": "openid profile"
}
```

**Error response (400 Bad Request):**

```json
{
  "error": "invalid_grant",
  "error_description": "Invalid username or password."
}
```

The error message is intentionally vague to prevent user enumeration attacks.

---

### Device Authorization Grant

Used for input-constrained devices (smart TVs, CLI tools, IoT devices) that cannot use a browser-based flow directly. The client polls this endpoint after initiating the device flow via `POST /oauth/device/code`.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `urn:ietf:params:oauth:grant-type:device_code` |
| `device_code` | string | Yes | The device code from the device authorization response |
| `client_id` | string | Yes | The client identifier |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS" \
  -d "client_id=device-app"
```

**Response while waiting (400 Bad Request):**

```json
{
  "error": "authorization_pending",
  "error_description": "The user has not yet authorized the device."
}
```

**Response if polling too fast (400 Bad Request):**

```json
{
  "error": "slow_down",
  "error_description": "Polling too frequently. Increase the interval by 5 seconds.",
  "interval": 10
}
```

**Response after user authorizes (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "ZGV2aWNlLXJlZnJlc2gtdG9rZW4",
  "id_token": "eyJhbGciOiJSUzI1NiIs...",
  "scope": "openid profile"
}
```

**Response if device code expired (400 Bad Request):**

```json
{
  "error": "expired_token",
  "error_description": "The device code has expired. Please restart the device authorization flow."
}
```

---

### Token Error Responses

All grant types may return the following errors:

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| `invalid_request` | 400 | Missing required parameter or malformed request |
| `invalid_client` | 401 | Client authentication failed |
| `invalid_grant` | 400 | Code, refresh token, or credentials are invalid or expired |
| `unauthorized_client` | 400 | Client is not authorized for this grant type |
| `unsupported_grant_type` | 400 | Grant type not supported or not enabled for this client |
| `invalid_scope` | 400 | Requested scope is invalid or not allowed for this client |
| `authorization_pending` | 400 | Device code: user has not yet authorized |
| `slow_down` | 400 | Device code: client is polling too frequently |
| `expired_token` | 400 | Device code: the device code has expired |

---

## GET /oauth/authorize

The authorization endpoint initiates the user-facing OAuth 2.0 authorization flow. The client redirects the user's browser to this endpoint. After authentication and consent, Rampart redirects back to the client with an authorization code.

**This is a browser-based redirect endpoint, not a JSON API.**

### Request Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `response_type` | string | Yes | Must be `code` for Authorization Code flow |
| `client_id` | string | Yes | The registered client identifier |
| `redirect_uri` | string | Yes | Must match a registered redirect URI for the client |
| `scope` | string | No | Space-separated list of scopes (default: `openid`) |
| `state` | string | Recommended | Opaque value to prevent CSRF -- returned unchanged in the redirect |
| `nonce` | string | No | Value included in the ID token to prevent replay attacks |
| `code_challenge` | string | Conditional | PKCE challenge (required for public clients, recommended for all) |
| `code_challenge_method` | string | Conditional | Must be `S256` (plain is not supported for security reasons) |
| `prompt` | string | No | `none`, `login`, or `consent` -- controls authentication UI behavior |
| `login_hint` | string | No | Pre-fills the username/email field on the login page |
| `organization_id` | string | No | Scopes the login to a specific organization and applies its theme |
| `max_age` | integer | No | Maximum authentication age in seconds. Forces re-authentication if the user's session is older. |
| `ui_locales` | string | No | Space-separated list of preferred locales (e.g., `en de fr`) |
| `acr_values` | string | No | Requested authentication context class references |

### Example: Authorization Code with PKCE

**Step 1: Generate PKCE values (client-side)**

```bash
# Generate code_verifier (43-128 characters, unreserved URI characters)
CODE_VERIFIER=$(openssl rand -base64 32 | tr -d '=' | tr '/+' '_-')

# Generate code_challenge (SHA256 hash of verifier, base64url-encoded)
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | \
  openssl dgst -sha256 -binary | base64 | tr -d '=' | tr '/+' '_-')

echo "Verifier: $CODE_VERIFIER"
echo "Challenge: $CODE_CHALLENGE"
```

**Step 2: Redirect user to authorize**

```
GET https://your-rampart-instance/oauth/authorize
  ?response_type=code
  &client_id=my-spa
  &redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback
  &scope=openid%20profile%20email
  &state=xyzABC123
  &nonce=n-0S6_WzA2Mj
  &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
  &code_challenge_method=S256
```

**Step 3: User authenticates and consents**

Rampart displays the login page (themed per organization). After successful authentication and consent, the user is redirected back:

```
HTTP/1.1 302 Found
Location: https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyzABC123
```

**Step 4: Exchange code for tokens**

Use `POST /oauth/token` with the `authorization_code` grant type, including the `code_verifier`.

### Prompt Parameter Behavior

| Value | Behavior |
|-------|----------|
| `none` | No login or consent UI is shown. Returns `login_required` or `consent_required` error if user interaction is needed. Useful for silent authentication checks. |
| `login` | Forces re-authentication even if the user has an active session. |
| `consent` | Forces the consent screen even if the user previously consented to the requested scopes. |

### Successful Redirect

```
HTTP/1.1 302 Found
Location: https://app.example.com/callback
  ?code=SplxlOBeZQQYbYS6WxSbIA
  &state=xyzABC123
```

The authorization code is single-use and expires after 10 minutes.

### Error Redirect

If the request is invalid but the `redirect_uri` can be validated, errors are returned as query parameters on the redirect:

```
https://app.example.com/callback
  ?error=invalid_scope
  &error_description=The+requested+scope+%27admin%27+is+not+allowed+for+this+client
  &state=xyzABC123
```

If the `redirect_uri` itself is invalid, unregistered, or missing, Rampart displays an error page directly instead of redirecting. This prevents open redirect attacks.

---

## POST /oauth/revoke

The revocation endpoint invalidates an access token or refresh token. Implements [RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009).

**Content-Type:** `application/x-www-form-urlencoded`

Client authentication is required via HTTP Basic or body parameters.

### Request Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes | The token to revoke |
| `token_type_hint` | string | No | `access_token` or `refresh_token` -- helps the server locate the token faster |

### Revoke a Refresh Token

```bash
curl -X POST https://your-rampart-instance/oauth/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-web-app:my-client-secret" \
  -d "token=dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4gZXhhbXBsZQ" \
  -d "token_type_hint=refresh_token"
```

### Revoke an Access Token

```bash
curl -X POST https://your-rampart-instance/oauth/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-web-app:my-client-secret" \
  -d "token=eyJhbGciOiJSUzI1NiIs..." \
  -d "token_type_hint=access_token"
```

### Response (200 OK)

Per RFC 7009, the revocation endpoint **always** returns HTTP 200 OK, even if the token was already invalid, expired, or not recognized. This prevents token fishing attacks.

```
HTTP/1.1 200 OK
Content-Length: 0
```

**Revocation behavior:**

- When a **refresh token** is revoked, all access tokens issued from that refresh token are also invalidated.
- When an **access token** is revoked, it is added to a Redis-backed deny list. The deny list entry expires when the token would have naturally expired.
- Revocation is logged as a `token.revoked` audit event.

---

## POST /oauth/introspect

The introspection endpoint allows resource servers to validate a token and retrieve its metadata. Implements [RFC 7662](https://datatracker.ietf.org/doc/html/rfc7662).

**Content-Type:** `application/x-www-form-urlencoded`

Client authentication is required. Only clients with the `token_introspection` capability can call this endpoint.

### Request Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes | The token to introspect |
| `token_type_hint` | string | No | `access_token` or `refresh_token` |

### Request

```bash
curl -X POST https://your-rampart-instance/oauth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "resource-server:rs-secret-456" \
  -d "token=eyJhbGciOiJSUzI1NiIs..." \
  -d "token_type_hint=access_token"
```

### Response: Active Token (200 OK)

```json
{
  "active": true,
  "scope": "openid profile email",
  "client_id": "my-web-app",
  "username": "jane.doe",
  "token_type": "Bearer",
  "exp": 1709514000,
  "iat": 1709510400,
  "nbf": 1709510400,
  "sub": "usr_1234567890",
  "aud": "my-web-app",
  "iss": "https://your-rampart-instance",
  "jti": "tok_abc123def456",
  "organization_id": "org_default"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `active` | boolean | Whether the token is currently valid |
| `scope` | string | Space-separated list of granted scopes |
| `client_id` | string | The client that requested this token |
| `username` | string | The username of the token's subject (if applicable) |
| `token_type` | string | The type of token (`Bearer`) |
| `exp` | integer | Expiration time (Unix timestamp) |
| `iat` | integer | Issued-at time (Unix timestamp) |
| `nbf` | integer | Not-before time (Unix timestamp) |
| `sub` | string | Subject identifier (user ID or client ID) |
| `aud` | string | Intended audience |
| `iss` | string | Issuer URL |
| `jti` | string | Unique token identifier |
| `organization_id` | string | The organization this token belongs to |

### Response: Inactive Token (200 OK)

If the token is expired, revoked, malformed, or not recognized, the response contains only `active: false`. This prevents information leakage about why the token is inactive.

```json
{
  "active": false
}
```

### Introspection vs. Local JWT Validation

For most use cases, resource servers should validate JWTs **locally** using the public keys from the [JWKS endpoint](./oidc-discovery.md#get-well-knownjwksjson). Local validation is faster and does not require a network call for every request.

Use introspection when you need:

| Scenario | Why Introspection |
|----------|-------------------|
| Real-time revocation checking | JWTs remain valid until expiry; introspection checks the deny list |
| Opaque tokens | Opaque tokens cannot be validated locally |
| Token metadata | Metadata not encoded in the JWT claims |
| Centralized policy decisions | Server-side token validation with current state |

---

## Token Lifetimes

Default token lifetimes are configurable per client in the admin console or via the Admin API:

| Token Type | Default Lifetime | Configurable Range | Notes |
|------------|------------------|--------------------|-------|
| Access token | 1 hour | 5 minutes to 24 hours | Short-lived by design |
| Refresh token | 30 days | 1 hour to 365 days | Rotated on each use |
| ID token | 1 hour | 5 minutes to 24 hours | Matches access token lifetime |
| Authorization code | 10 minutes | 1 minute to 10 minutes | Single-use |
| Device code | 10 minutes | 1 minute to 30 minutes | Configurable per client |

### Absolute vs. Sliding Expiration

Refresh tokens support two expiration modes:

- **Absolute expiration** (default): The refresh token expires at a fixed time regardless of usage. After 30 days, the user must re-authenticate.
- **Sliding expiration**: Each use of the refresh token extends its lifetime by the configured duration. The token only expires after a period of inactivity. A maximum absolute lifetime can be set as a safety limit.

Configure this per client via `refresh_token_expiration_mode` (`absolute` or `sliding`).

---

## Security Considerations

- **Always use HTTPS** in production. Rampart rejects non-HTTPS redirect URIs unless the `RAMPART_DEV_MODE=true` environment variable is set.
- **Always use PKCE** for public clients (SPAs, mobile apps). Rampart enforces PKCE for public clients and strongly recommends it for confidential clients as well.
- **Store tokens securely.** Keep access tokens in memory only. Store refresh tokens in secure HTTP-only cookies or platform-specific secure storage (Keychain, Keystore).
- **Use short-lived access tokens.** Rely on refresh tokens for session continuity rather than long-lived access tokens.
- **Validate the `state` parameter** in your client to prevent CSRF attacks on the authorization flow.
- **Validate the `nonce` claim** in ID tokens to prevent replay attacks.
- **Validate the `iss` and `aud` claims** in tokens to ensure they were issued by your Rampart instance for your client.
- The password grant is disabled by default. Enable it only for trusted first-party clients that cannot use a browser-based flow.
- Failed authentication attempts are rate-limited per IP and logged as audit events.
