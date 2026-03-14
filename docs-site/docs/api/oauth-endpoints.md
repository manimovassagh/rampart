---
sidebar_position: 3
title: OAuth 2.0 Endpoints
description: Full OAuth 2.0 endpoint reference for Rampart -- authorization, token, revocation, introspection, UserInfo, and device authorization endpoints.
---

# OAuth 2.0 Endpoints

This page provides the complete reference for all OAuth 2.0 and OpenID Connect endpoints exposed by Rampart. These endpoints implement [RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749) (OAuth 2.0), [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636) (PKCE), [RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009) (Token Revocation), [RFC 7662](https://datatracker.ietf.org/doc/html/rfc7662) (Token Introspection), [RFC 8628](https://datatracker.ietf.org/doc/html/rfc8628) (Device Authorization), and [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html).

## Endpoint Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/oauth/authorize` | Authorization endpoint -- initiates user authentication |
| POST | `/oauth/token` | Token endpoint -- issues and refreshes tokens |
| POST | `/oauth/revoke` | Revocation endpoint -- invalidates tokens |
| POST | `/oauth/introspect` | Introspection endpoint -- validates and inspects tokens |
| GET/POST | `/oauth/userinfo` | UserInfo endpoint -- returns claims about the authenticated user |
| POST | `/oauth/device/code` | Device authorization endpoint -- initiates device flow |
| GET | `/oauth/device` | Device verification page -- user enters device code |
| GET | `/.well-known/openid-configuration` | OIDC discovery document |
| GET | `/.well-known/jwks.json` | JSON Web Key Set for token verification |

---

## Authorization Endpoint

### GET /oauth/authorize

Initiates the OAuth 2.0 authorization flow. The client redirects the user's browser to this endpoint. Rampart authenticates the user, obtains consent, and redirects back with an authorization code.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `response_type` | string | Yes | Must be `code` |
| `client_id` | string | Yes | Registered OAuth client identifier |
| `redirect_uri` | string | Yes | Must exactly match a registered redirect URI |
| `scope` | string | No | Space-separated scopes (default: `openid`) |
| `state` | string | Recommended | Opaque CSRF protection value, returned unchanged |
| `nonce` | string | No | Value included in the ID token to prevent replay attacks |
| `code_challenge` | string | Conditional | Base64url-encoded SHA-256 hash of the code verifier (required for public clients) |
| `code_challenge_method` | string | Conditional | Must be `S256` |
| `prompt` | string | No | `none`, `login`, `consent`, or `select_account` |
| `login_hint` | string | No | Pre-fills the username/email field |
| `organization_id` | string | No | Restricts login to a specific organization |
| `max_age` | integer | No | Maximum authentication age in seconds |
| `ui_locales` | string | No | Preferred UI locales (space-separated BCP 47 tags) |
| `acr_values` | string | No | Requested authentication context class references |

**Example request URL:**

```
https://your-rampart-instance/oauth/authorize
  ?response_type=code
  &client_id=my-spa
  &redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback
  &scope=openid%20profile%20email
  &state=af0ifjsldkj
  &nonce=n-0S6_WzA2Mj
  &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
  &code_challenge_method=S256
```

**Successful redirect:**

```
HTTP/1.1 302 Found
Location: https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=af0ifjsldkj
```

**Error redirect:**

```
HTTP/1.1 302 Found
Location: https://app.example.com/callback?error=access_denied&error_description=The+user+denied+the+request&state=af0ifjsldkj
```

**Error codes:**

| Error | Description |
|-------|-------------|
| `invalid_request` | Missing or invalid parameter |
| `unauthorized_client` | Client is not authorized for this flow |
| `access_denied` | User denied consent |
| `unsupported_response_type` | Only `code` is supported |
| `invalid_scope` | Requested scope is not allowed |
| `server_error` | Internal error |
| `login_required` | `prompt=none` was used but the user is not authenticated |
| `consent_required` | `prompt=none` was used but consent has not been granted |
| `interaction_required` | `prompt=none` was used but user interaction is needed (e.g., MFA) |

### Prompt Behavior

| Value | Behavior |
|-------|----------|
| `none` | No UI is shown. Fails with `login_required` or `consent_required` if interaction is needed. Used for silent session checks. |
| `login` | Forces re-authentication even if the user has an active session. |
| `consent` | Forces the consent screen even if previously consented. |
| `select_account` | Shows an account picker if the user has multiple sessions. |

---

## Token Endpoint

### POST /oauth/token

Issues access tokens, ID tokens, and refresh tokens. Supports multiple grant types.

**Content-Type:** `application/x-www-form-urlencoded`

See the [Authentication Endpoints](./authentication.md#post-oauthtoken) page for detailed documentation of each grant type with full request/response examples.

### Supported Grant Types

| Grant Type | `grant_type` Value | Use Case |
|------------|-------------------|----------|
| Authorization Code | `authorization_code` | Web apps, SPAs, mobile apps |
| Client Credentials | `client_credentials` | Service-to-service |
| Refresh Token | `refresh_token` | Renewing expired access tokens |
| Resource Owner Password | `password` | Trusted first-party apps (disabled by default) |
| Device Authorization | `urn:ietf:params:oauth:grant-type:device_code` | CLI tools, smart TVs, IoT |

### Access Token Claims

Rampart issues JWTs as access tokens with the following standard claims:

```json
{
  "iss": "https://your-rampart-instance",
  "sub": "usr_1234567890",
  "aud": "my-web-app",
  "exp": 1709514000,
  "iat": 1709510400,
  "nbf": 1709510400,
  "jti": "tok_abc123def456",
  "scope": "openid profile email",
  "client_id": "my-web-app",
  "org_id": "org_default",
  "roles": ["user", "editor"]
}
```

| Claim | Type | Description |
|-------|------|-------------|
| `iss` | string | Issuer -- your Rampart instance URL |
| `sub` | string | Subject -- user ID or client ID |
| `aud` | string/array | Audience -- the client(s) this token is intended for |
| `exp` | integer | Expiration time (Unix timestamp) |
| `iat` | integer | Issued-at time (Unix timestamp) |
| `nbf` | integer | Not-before time (Unix timestamp) |
| `jti` | string | Unique token identifier |
| `scope` | string | Space-separated granted scopes |
| `client_id` | string | The client that requested this token |
| `org_id` | string | The organization context |
| `roles` | array | User's roles within the organization |

### ID Token Claims

When the `openid` scope is requested, an ID token is returned with identity claims:

```json
{
  "iss": "https://your-rampart-instance",
  "sub": "usr_1234567890",
  "aud": "my-web-app",
  "exp": 1709514000,
  "iat": 1709510400,
  "nonce": "n-0S6_WzA2Mj",
  "auth_time": 1709510300,
  "at_hash": "MTIzNDU2Nzg5MA",
  "name": "Jane Doe",
  "given_name": "Jane",
  "family_name": "Doe",
  "email": "jane@example.com",
  "email_verified": true
}
```

Claims included depend on the requested scopes:

| Scope | Claims Added |
|-------|-------------|
| `openid` | `sub`, `iss`, `aud`, `exp`, `iat`, `nonce`, `auth_time` |
| `profile` | `name`, `given_name`, `family_name`, `preferred_username`, `updated_at` |
| `email` | `email`, `email_verified` |
| `phone` | `phone_number`, `phone_number_verified` |

---

## Revocation Endpoint

### POST /oauth/revoke

Invalidates an access token or refresh token. Implements [RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009).

**Content-Type:** `application/x-www-form-urlencoded`

Client authentication is required.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes | The token to revoke |
| `token_type_hint` | string | No | `access_token` or `refresh_token` |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my-web-app:my-client-secret" \
  -d "token=dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4" \
  -d "token_type_hint=refresh_token"
```

**Response:**

Always returns HTTP 200 OK with an empty body, regardless of whether the token was valid. This prevents token fishing.

```
HTTP/1.1 200 OK
Content-Length: 0
```

**Revocation cascade:**
- Revoking a **refresh token** also invalidates all access tokens issued from it.
- Revoking an **access token** adds it to a deny list (expires when the token would have).

---

## Introspection Endpoint

### POST /oauth/introspect

Validates a token and returns its metadata. Implements [RFC 7662](https://datatracker.ietf.org/doc/html/rfc7662).

**Content-Type:** `application/x-www-form-urlencoded`

Client authentication is required. The calling client must have the `token_introspection` capability enabled.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes | The token to introspect |
| `token_type_hint` | string | No | `access_token` or `refresh_token` |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "resource-server:rs-secret-456" \
  -d "token=eyJhbGciOiJSUzI1NiIs..."
```

**Active token response (200 OK):**

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
  "jti": "tok_abc123def456"
}
```

**Inactive token response (200 OK):**

```json
{
  "active": false
}
```

A token is reported as inactive if it is expired, revoked, malformed, or issued by a different Rampart instance.

---

## UserInfo Endpoint

### GET /oauth/userinfo

### POST /oauth/userinfo

Returns claims about the authenticated user. The access token must include the `openid` scope. Both GET and POST methods are supported per the OpenID Connect specification.

**Request (GET):**

```bash
curl -X GET https://your-rampart-instance/oauth/userinfo \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..."
```

**Request (POST):**

```bash
curl -X POST https://your-rampart-instance/oauth/userinfo \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Content-Type: application/x-www-form-urlencoded"
```

**Response (200 OK):**

```json
{
  "sub": "usr_1234567890",
  "name": "Jane Doe",
  "given_name": "Jane",
  "family_name": "Doe",
  "preferred_username": "jane.doe",
  "email": "jane@example.com",
  "email_verified": true,
  "updated_at": 1709510400,
  "org_id": "org_default"
}
```

The claims returned depend on the scopes granted to the access token:

| Scope | Claims |
|-------|--------|
| `openid` | `sub` |
| `profile` | `name`, `given_name`, `family_name`, `preferred_username`, `updated_at` |
| `email` | `email`, `email_verified` |
| `phone` | `phone_number`, `phone_number_verified` |

**Error responses:**

| HTTP Status | Error | Description |
|-------------|-------|-------------|
| 401 | `invalid_token` | Token is missing, expired, or revoked |
| 403 | `insufficient_scope` | Token does not include the `openid` scope |

```json
{
  "error": "invalid_token",
  "error_description": "The access token is expired."
}
```

---

## Device Authorization Endpoint

### POST /oauth/device/code

Initiates the device authorization flow for input-constrained devices. Implements [RFC 8628](https://datatracker.ietf.org/doc/html/rfc8628).

This endpoint returns a device code and a user code. The device displays the user code and a verification URL. The user visits the URL on a separate device (phone, laptop) and enters the code to authorize.

**Content-Type:** `application/x-www-form-urlencoded`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `client_id` | string | Yes | The registered client identifier |
| `scope` | string | No | Space-separated list of requested scopes |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/device/code \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=cli-tool" \
  -d "scope=openid profile"
```

**Response (200 OK):**

```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "WDJB-MJHT",
  "verification_uri": "https://your-rampart-instance/oauth/device",
  "verification_uri_complete": "https://your-rampart-instance/oauth/device?user_code=WDJB-MJHT",
  "expires_in": 600,
  "interval": 5
}
```

| Field | Type | Description |
|-------|------|-------------|
| `device_code` | string | Code used by the device to poll for tokens (kept secret on the device) |
| `user_code` | string | Short code displayed to the user for manual entry |
| `verification_uri` | string | URL the user visits to enter the code |
| `verification_uri_complete` | string | URL with the user code pre-filled (can be shown as a QR code) |
| `expires_in` | integer | Seconds until the device code expires |
| `interval` | integer | Minimum seconds between polling requests |

### Device Flow Step-by-Step

**Step 1: Request device and user codes**

```bash
curl -X POST https://your-rampart-instance/oauth/device/code \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=cli-tool" \
  -d "scope=openid profile"
```

**Step 2: Display instructions to the user**

Show the user something like:

```
To sign in, visit: https://your-rampart-instance/oauth/device
Enter code: WDJB-MJHT
```

Or display a QR code for `verification_uri_complete`.

**Step 3: Poll for tokens**

The device polls `POST /oauth/token` using the device code:

```bash
# Poll every 5 seconds (the interval from the response)
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS" \
  -d "client_id=cli-tool"
```

**While waiting:**

```json
{
  "error": "authorization_pending",
  "error_description": "The user has not yet authorized the device."
}
```

**If polling too fast:**

```json
{
  "error": "slow_down",
  "error_description": "Polling too frequently. Increase the interval by 5 seconds.",
  "interval": 10
}
```

**Step 4: User authorizes**

The user visits the verification URL, logs in, enters the user code, and grants consent.

**Step 5: Device receives tokens**

The next poll returns a successful token response:

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

### GET /oauth/device

The device verification page. This is a user-facing web page, not an API endpoint. The user visits this URL in their browser to enter the user code and authorize the device.

If the `user_code` query parameter is present (from `verification_uri_complete`), the code is pre-filled.

---

## Complete PKCE Flow Example

Here is a complete Authorization Code with PKCE flow from start to finish, using JavaScript.

### Step 1: Generate PKCE Values

```javascript
// Generate a cryptographically random code verifier
function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64urlEncode(array);
}

// Generate the code challenge from the verifier
async function generateCodeChallenge(verifier) {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return base64urlEncode(new Uint8Array(digest));
}

function base64urlEncode(buffer) {
  return btoa(String.fromCharCode(...buffer))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

const codeVerifier = generateCodeVerifier();
const codeChallenge = await generateCodeChallenge(codeVerifier);

// Store the verifier securely (e.g., sessionStorage)
sessionStorage.setItem("pkce_verifier", codeVerifier);
```

### Step 2: Redirect to Authorization Endpoint

```javascript
const state = crypto.randomUUID();
sessionStorage.setItem("oauth_state", state);

const params = new URLSearchParams({
  response_type: "code",
  client_id: "my-spa",
  redirect_uri: "https://app.example.com/callback",
  scope: "openid profile email",
  state: state,
  code_challenge: codeChallenge,
  code_challenge_method: "S256",
});

window.location.href =
  `https://your-rampart-instance/oauth/authorize?${params}`;
```

### Step 3: Handle the Callback

```javascript
// On your callback page (https://app.example.com/callback)
const params = new URLSearchParams(window.location.search);
const code = params.get("code");
const returnedState = params.get("state");

// Verify state matches
const savedState = sessionStorage.getItem("oauth_state");
if (returnedState !== savedState) {
  throw new Error("State mismatch -- possible CSRF attack");
}

// Retrieve the code verifier
const codeVerifier = sessionStorage.getItem("pkce_verifier");
```

### Step 4: Exchange Code for Tokens

```javascript
const tokenResponse = await fetch(
  "https://your-rampart-instance/oauth/token",
  {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code: code,
      redirect_uri: "https://app.example.com/callback",
      client_id: "my-spa",
      code_verifier: codeVerifier,
    }),
  }
);

const tokens = await tokenResponse.json();
// tokens.access_token -- use for API calls
// tokens.id_token -- user identity claims
// tokens.refresh_token -- use to get new access tokens
```

### Step 5: Use the Access Token

```javascript
const apiResponse = await fetch("https://api.example.com/data", {
  headers: {
    Authorization: `Bearer ${tokens.access_token}`,
  },
});
```

### Step 6: Refresh When Expired

```javascript
const refreshResponse = await fetch(
  "https://your-rampart-instance/oauth/token",
  {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: new URLSearchParams({
      grant_type: "refresh_token",
      refresh_token: tokens.refresh_token,
      client_id: "my-spa",
    }),
  }
);

const newTokens = await refreshResponse.json();
// Update stored tokens with newTokens
```

---

## Endpoint Authentication Summary

| Endpoint | Authentication Required | Method |
|----------|------------------------|--------|
| `GET /oauth/authorize` | None (user authenticates interactively) | -- |
| `POST /oauth/token` | Confidential clients only | Basic or POST body |
| `POST /oauth/revoke` | Yes | Basic or POST body |
| `POST /oauth/introspect` | Yes | Basic or POST body |
| `GET/POST /oauth/userinfo` | Yes | Bearer token |
| `POST /oauth/device/code` | None (public clients) or client auth | Basic or POST body |
