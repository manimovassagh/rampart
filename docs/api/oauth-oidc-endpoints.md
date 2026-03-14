# OAuth 2.0 / OpenID Connect Endpoints

Standard protocol endpoints that follow their respective RFCs. These paths are fixed and not versioned.

## Endpoint Summary

| Method | Path | Description | Spec |
|--------|------|-------------|------|
| GET | `/oauth/authorize` | Authorization endpoint | [RFC 6749 §3.1](https://tools.ietf.org/html/rfc6749#section-3.1) |
| POST | `/oauth/token` | Token endpoint | [RFC 6749 §3.2](https://tools.ietf.org/html/rfc6749#section-3.2) |
| POST | `/oauth/revoke` | Token revocation | [RFC 7009](https://tools.ietf.org/html/rfc7009) |
| POST | `/oauth/introspect` | Token introspection *(planned)* | [RFC 7662](https://tools.ietf.org/html/rfc7662) |
| POST | `/oauth/device` | Device authorization *(planned)* | [RFC 8628](https://tools.ietf.org/html/rfc8628) |
| GET | `/me` | Current user profile | Proprietary |
| GET | `/oidc/logout` | End session *(planned)* | [OIDC RP-Initiated Logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html) |
| GET | `/.well-known/openid-configuration` | Discovery document | [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html) |
| GET | `/.well-known/jwks.json` | JSON Web Key Set | [RFC 7517](https://tools.ietf.org/html/rfc7517) |

---

## Authorization Endpoint

```
GET /oauth/authorize
```

Initiates an authorization flow. The user is redirected to the login page if not already authenticated.

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `response_type` | Yes | `code` for Authorization Code flow |
| `client_id` | Yes | The registered client identifier |
| `redirect_uri` | Yes | Must match a registered redirect URI |
| `scope` | Yes | Space-delimited scopes (must include `openid` for OIDC) |
| `state` | Recommended | Opaque value for CSRF protection |
| `code_challenge` | Required for public clients | PKCE code challenge |
| `code_challenge_method` | Required with code_challenge | Must be `S256` |
| `nonce` | Recommended for OIDC | Binds ID token to session |
| `prompt` | Optional | `none`, `login`, `consent`, `select_account` |
| `login_hint` | Optional | Email or username hint |
| `acr_values` | Optional | Requested authentication context |

### Example Request

```http
GET /oauth/authorize?response_type=code
  &client_id=my-spa
  &redirect_uri=https://app.example.com/callback
  &scope=openid profile email
  &state=abc123
  &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
  &code_challenge_method=S256
  &nonce=xyz789
```

### Success Response

Redirects to `redirect_uri` with:

```
https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=abc123
```

### Error Response

Redirects with error parameters per RFC 6749 §4.1.2.1:

```
https://app.example.com/callback?error=access_denied&error_description=User+denied+consent&state=abc123
```

---

## Token Endpoint

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded
```

Exchanges an authorization code or refresh token for tokens.

### Authorization Code Exchange

```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=SplxlOBeZQQYbYS6WxSbIA
&redirect_uri=https://app.example.com/callback
&client_id=my-spa
&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
```

### Refresh Token

```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=tGzv3JOkF0XG5Qx2TlKWIA
&client_id=my-spa
```

### Success Response

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "tGzv3JOkF0XG5Qx2TlKWIA",
  "id_token": "eyJhbGciOiJSUzI1NiIs...",
  "scope": "openid profile email"
}
```

### Error Response

```json
{
  "error": "invalid_grant",
  "error_description": "The authorization code has expired."
}
```

---

## Token Revocation

```
POST /oauth/revoke
Content-Type: application/x-www-form-urlencoded
```

Revokes an access or refresh token. Per RFC 7009, returns 200 even if the token is invalid.

### Request

```http
POST /oauth/revoke
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:client_secret)

token=tGzv3JOkF0XG5Qx2TlKWIA
&token_type_hint=refresh_token
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `token` | Yes | The token to revoke |
| `token_type_hint` | Optional | `access_token` or `refresh_token` |

### Response

```http
HTTP/1.1 200 OK
```

---

## Token Introspection *(Planned)*

> **Note:** This endpoint is not yet implemented. It is planned for a future release.

```
POST /oauth/introspect
Content-Type: application/x-www-form-urlencoded
```

Returns metadata about a token. Requires client authentication.

### Request

```http
POST /oauth/introspect
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:client_secret)

token=eyJhbGciOiJSUzI1NiIs...
```

### Active Token Response

```json
{
  "active": true,
  "scope": "openid profile email",
  "client_id": "my-spa",
  "username": "jane@example.com",
  "token_type": "Bearer",
  "exp": 1709500000,
  "iat": 1709496400,
  "sub": "usr_abc123",
  "iss": "https://auth.example.com",
  "aud": "my-spa"
}
```

### Inactive Token Response

```json
{
  "active": false
}
```

---

## Device Authorization *(Planned)*

> **Note:** This endpoint is not yet implemented. It is planned for a future release.

```
POST /oauth/device
Content-Type: application/x-www-form-urlencoded
```

Initiates the device authorization flow (RFC 8628) for input-constrained devices like CLIs and smart TVs.

### Request

```http
POST /oauth/device
Content-Type: application/x-www-form-urlencoded

client_id=my-cli
&scope=openid profile
```

### Response

```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "WDJB-MJHT",
  "verification_uri": "https://auth.example.com/device",
  "verification_uri_complete": "https://auth.example.com/device?user_code=WDJB-MJHT",
  "expires_in": 600,
  "interval": 5
}
```

The device polls `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:device_code` until the user completes authentication.

---

## User Profile Endpoint

```
GET /me
Authorization: Bearer <access_token>
```

Returns information about the authenticated user.

### Response

```json
{
  "sub": "usr_abc123",
  "name": "Jane Doe",
  "given_name": "Jane",
  "family_name": "Doe",
  "email": "jane@example.com",
  "email_verified": true,
  "picture": "https://example.com/jane.jpg",
  "updated_at": 1709496400
}
```

### Scope-to-Claims Mapping

| Scope | Claims |
|-------|--------|
| `openid` | `sub` |
| `profile` | `name`, `given_name`, `family_name`, `picture`, `updated_at` |
| `email` | `email`, `email_verified` |
| `phone` | `phone_number`, `phone_number_verified` |

---

## End Session (Logout) *(Planned)*

> **Note:** OIDC RP-Initiated Logout is not yet implemented. Use `POST /logout` with a refresh token to revoke sessions.

```
GET /oidc/logout
```

Ends the user's session and optionally redirects back to the client.

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `id_token_hint` | Recommended | Previously issued ID token |
| `post_logout_redirect_uri` | Optional | Where to redirect after logout |
| `state` | Optional | Opaque value passed to redirect URI |

### Example

```http
GET /oidc/logout?id_token_hint=eyJhbGci...&post_logout_redirect_uri=https://app.example.com&state=abc
```

---

## Discovery Document

```
GET /.well-known/openid-configuration
```

Returns the OIDC Discovery metadata. All clients should use this to discover endpoints.

### Response

```json
{
  "issuer": "https://auth.example.com",
  "authorization_endpoint": "https://auth.example.com/oauth/authorize",
  "token_endpoint": "https://auth.example.com/oauth/token",
  "userinfo_endpoint": "https://auth.example.com/oidc/userinfo",
  "revocation_endpoint": "https://auth.example.com/oauth/revoke",
  "end_session_endpoint": "https://auth.example.com/logout",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "scopes_supported": ["openid", "profile", "email", "phone", "offline_access"],
  "response_types_supported": ["code"],
  "grant_types_supported": [
    "authorization_code",
    "refresh_token"
  ],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256", "ES256"],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "private_key_jwt",
    "none"
  ],
  "code_challenge_methods_supported": ["S256"],
  "claims_supported": [
    "sub", "iss", "aud", "exp", "iat", "nonce",
    "name", "given_name", "family_name", "email", "email_verified",
    "phone_number", "phone_number_verified", "picture", "updated_at"
  ]
}
```

---

## JSON Web Key Set

```
GET /.well-known/jwks.json
```

Returns the public keys used to verify token signatures. Clients use these to validate JWTs locally.

### Response

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "rampart-rsa-2026",
      "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM...",
      "e": "AQAB"
    },
    {
      "kty": "EC",
      "use": "sig",
      "alg": "ES256",
      "kid": "rampart-ec-2026",
      "crv": "P-256",
      "x": "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
      "y": "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0"
    }
  ]
}
```

Keys are rotated periodically. Clients should cache keys and refetch when they encounter an unknown `kid`.
