---
sidebar_position: 3
title: OAuth 2.0 Endpoints
description: Full OAuth 2.0 endpoint reference for Rampart -- authorization, consent, token, revocation, social login, and SAML endpoints.
---

# OAuth 2.0 Endpoints

This page provides the complete reference for all OAuth 2.0 and OpenID Connect endpoints exposed by Rampart. These endpoints implement [RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749) (OAuth 2.0), [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636) (PKCE), [RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009) (Token Revocation), and [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html).

## Endpoint Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET/POST | `/oauth/authorize` | Authorization endpoint -- initiates user authentication |
| POST | `/oauth/consent` | Consent endpoint -- user approves/denies scope requests |
| POST | `/oauth/token` | Token endpoint -- exchanges authorization codes and refresh tokens |
| POST | `/oauth/revoke` | Revocation endpoint -- invalidates refresh tokens (RFC 7009) |
| GET | `/oauth/social/{provider}` | Initiate social login (redirect to provider) |
| GET/POST | `/oauth/social/{provider}/callback` | Social login callback |
| GET | `/.well-known/openid-configuration` | OIDC discovery document |
| GET | `/.well-known/jwks.json` | JSON Web Key Set for token verification |

---

## Authorization Endpoint

### GET /oauth/authorize

### POST /oauth/authorize

Initiates the OAuth 2.0 authorization flow. The client redirects the user's browser to this endpoint. Rampart authenticates the user, obtains consent, and redirects back with an authorization code.

**This is a browser-based redirect endpoint, not a JSON API.** Both GET and POST methods are supported.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `response_type` | string | Yes | Must be `code` |
| `client_id` | string | Yes | Registered OAuth client identifier |
| `redirect_uri` | string | Yes | Must exactly match a registered redirect URI |
| `scope` | string | No | Space-separated scopes (default: `openid`) |
| `state` | string | Recommended | Opaque CSRF protection value, returned unchanged |
| `nonce` | string | No | Value included in the ID token to prevent replay attacks |
| `code_challenge` | string | Yes | Base64url-encoded SHA-256 hash of the code verifier (PKCE required) |
| `code_challenge_method` | string | Yes | Must be `S256` |
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

## Consent Endpoint

### POST /oauth/consent

Handles the user's consent decision (approve or deny) for the requested scopes. This endpoint is called by the consent form rendered during the authorization flow -- it is not called directly by clients.

**Content-Type:** `application/x-www-form-urlencoded`

**Parameters (form body):**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `decision` | string | Yes | `approve` or `deny` |
| `client_id` | string | Yes | The OAuth client ID |
| `scope` | string | Yes | The requested scopes |
| `state` | string | Yes | The state value from the authorization request |
| `code_challenge` | string | Yes | The PKCE code challenge |
| `nonce` | string | No | The nonce from the authorization request |
| `redirect_uri` | string | Yes | The redirect URI |
| `csrf_token` | string | Yes | CSRF protection token |

**Behavior:**
- If `decision=approve`: generates an authorization code and redirects to the client's `redirect_uri` with `code` and `state` parameters.
- If `decision=deny`: redirects to the client's `redirect_uri` with `error=access_denied`.

---

## Token Endpoint

### POST /oauth/token

Issues access tokens, refresh tokens, and (optionally) ID tokens. Supports `authorization_code` and `refresh_token` grant types.

**Content-Type:** `application/x-www-form-urlencoded`

### Authorization Code Grant

Exchanges an authorization code for tokens after the user completes the authorization flow.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `authorization_code` |
| `code` | string | Yes | The authorization code received from the authorize endpoint |
| `redirect_uri` | string | Yes | Must match the redirect URI used in the authorization request |
| `client_id` | string | Yes | The client identifier |
| `client_secret` | string | Conditional | Required for confidential clients |
| `code_verifier` | string | Yes | PKCE code verifier (required for all clients) |

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

**Request (confidential client):**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=SplxlOBeZQQYbYS6WxSbIA" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "client_id=my-web-app" \
  -d "client_secret=my-client-secret" \
  -d "code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4gZXhhbXBsZQ",
  "id_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

The `id_token` is included only when the `openid` scope was requested. The response includes `Cache-Control: no-store` and `Pragma: no-cache` headers.

**Notes:**
- PKCE (`code_verifier`) is **required** for all clients, not just public clients.
- Confidential clients must also provide `client_secret`.

---

### Refresh Token Grant

Exchanges a refresh token for a new access token and refresh token.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `refresh_token` |
| `refresh_token` | string | Yes | The refresh token issued during the original token request |
| `client_id` | string | Conditional | Required if the original token was issued to a specific client |
| `client_secret` | string | Conditional | Required for confidential clients |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4gZXhhbXBsZQ" \
  -d "client_id=my-spa"
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...new-access-token",
  "refresh_token": "bmV3LXJlZnJlc2gtdG9rZW4tZXhhbXBsZQ",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

**Notes:**
- Refresh token rotation is enforced: each use issues a new refresh token and invalidates the old one.
- If a previously used refresh token is presented, Rampart treats it as potential token theft and the session may be invalidated (replay detection).
- The `client_id` on refresh must match the client that originally obtained the authorization code. Confidential clients must re-authenticate with `client_secret`.

---

### Supported Grant Types

| Grant Type | `grant_type` Value | Use Case |
|------------|-------------------|----------|
| Authorization Code | `authorization_code` | Web apps, SPAs, mobile apps |
| Refresh Token | `refresh_token` | Renewing expired access tokens |

### Token Error Responses

All grant types may return the following errors:

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| `invalid_request` | 400 | Missing required parameter or malformed request |
| `invalid_client` | 401 | Client authentication failed |
| `invalid_grant` | 400 | Code, refresh token, or credentials are invalid or expired |
| `unauthorized_client` | 400 | Client is not authorized for this grant type |
| `unsupported_grant_type` | 400 | Only `authorization_code` and `refresh_token` are supported |

Error responses include additional fields for consistency:

```json
{
  "error": "invalid_grant",
  "error_description": "Invalid, expired, or already-used authorization code.",
  "status": 400,
  "request_id": "req_abc123def456"
}
```

---

### Access Token Claims

Rampart issues JWTs as access tokens with the following standard claims:

```json
{
  "iss": "https://your-rampart-instance",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "aud": "my-web-app",
  "exp": 1709514000,
  "iat": 1709510400,
  "nbf": 1709510400,
  "jti": "tok_abc123def456",
  "scope": "openid profile email",
  "org_id": "770e8400-e29b-41d4-a716-446655440001",
  "preferred_username": "jane.doe",
  "email": "jane@example.com",
  "email_verified": true,
  "given_name": "Jane",
  "family_name": "Doe",
  "roles": ["user", "editor"]
}
```

| Claim | Type | Description |
|-------|------|-------------|
| `iss` | string | Issuer -- your Rampart instance URL |
| `sub` | string | Subject -- user ID (UUID) |
| `aud` | string | Audience -- the client_id this token was issued for |
| `exp` | integer | Expiration time (Unix timestamp) |
| `iat` | integer | Issued-at time (Unix timestamp) |
| `nbf` | integer | Not-before time (Unix timestamp) |
| `jti` | string | Unique token identifier |
| `scope` | string | Space-separated granted scopes |
| `org_id` | string | Organization ID (UUID) |
| `preferred_username` | string | The user's username |
| `email` | string | The user's email address |
| `email_verified` | boolean | Whether the email is verified |
| `given_name` | string | User's first name |
| `family_name` | string | User's last name |
| `roles` | array | User's effective roles (direct + group-inherited) |

### ID Token Claims

When the `openid` scope is requested, an ID token is returned with identity claims:

```json
{
  "iss": "https://your-rampart-instance",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "aud": "my-web-app",
  "exp": 1709514000,
  "iat": 1709510400,
  "nonce": "n-0S6_WzA2Mj",
  "auth_time": 1709510300,
  "at_hash": "MTIzNDU2Nzg5MA",
  "org_id": "770e8400-e29b-41d4-a716-446655440001",
  "preferred_username": "jane.doe",
  "email": "jane@example.com",
  "email_verified": true,
  "given_name": "Jane",
  "family_name": "Doe"
}
```

---

## Revocation Endpoint

### POST /oauth/revoke

Invalidates a refresh token and its associated session. Implements [RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009).

**Content-Type:** `application/x-www-form-urlencoded`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes | The token to revoke (refresh token) |
| `token_type_hint` | string | No | Optional hint (currently only refresh tokens can be revoked) |

**Request:**

```bash
curl -X POST https://your-rampart-instance/oauth/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4"
```

**Response:**

Always returns HTTP 200 OK with an empty body, regardless of whether the token was valid. This prevents token fishing (per RFC 7009).

```
HTTP/1.1 200 OK
```

**Notes:**
- Revoking a refresh token deletes the associated session, effectively invalidating the access token as well.
- Access tokens are short-lived JWTs and cannot be individually revoked server-side. They remain valid until they expire.

---

## Social Login Endpoints

### GET /oauth/social/\{provider\}

Initiates social login by redirecting the user to the external identity provider (e.g., Google, GitHub, Apple).

**Path parameters:**

| Parameter | Description |
|-----------|-------------|
| `provider` | Social provider name (e.g., `google`, `github`, `apple`) |

**Example:**

```
GET https://your-rampart-instance/oauth/social/google
```

Redirects the user to Google's OAuth 2.0 authorization endpoint.

### GET /oauth/social/\{provider\}/callback

### POST /oauth/social/\{provider\}/callback

Handles the callback from the social identity provider after authentication. Both GET and POST methods are supported because some providers (e.g., Apple Sign In) use `response_mode=form_post` which delivers the code and state via a POST request.

After successful authentication, the user is redirected to the original client application with an authorization code.

---

## SAML Endpoints

### GET /saml/providers

List configured SAML identity providers.

**Auth required:** No

**Response (200 OK):** Returns the list of available SAML providers.

### GET /saml/\{providerID\}/metadata

Returns the SAML SP metadata XML for the specified provider. This is used to configure the SAML identity provider.

**Content-Type:** `application/xml`

**Example:**

```bash
curl -X GET https://your-rampart-instance/saml/my-idp/metadata
```

### GET /saml/\{providerID\}/login

Initiates SAML SSO login by redirecting the user to the configured identity provider.

**Example:**

```
GET https://your-rampart-instance/saml/my-idp/login
```

Redirects the user to the SAML IdP with a SAML AuthnRequest.

### POST /saml/\{providerID\}/acs

The Assertion Consumer Service (ACS) endpoint. Receives the SAML response from the identity provider after authentication.

**Content-Type:** `application/x-www-form-urlencoded`

This endpoint processes the SAML assertion, creates or links the user account, and redirects back to the client application.

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
// tokens.id_token -- user identity claims (if openid scope requested)
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
| `GET/POST /oauth/authorize` | None (user authenticates interactively) | -- |
| `POST /oauth/consent` | None (uses consent cookie set during authorize) | Cookie |
| `POST /oauth/token` | Confidential clients only | POST body (`client_secret`) |
| `POST /oauth/revoke` | None | -- |
| `GET /oauth/social/{provider}` | None | -- |
| `GET/POST /oauth/social/{provider}/callback` | None | -- |
| `GET /saml/providers` | None | -- |
| `GET /saml/{providerID}/metadata` | None | -- |
| `GET /saml/{providerID}/login` | None | -- |
| `POST /saml/{providerID}/acs` | None | -- |
