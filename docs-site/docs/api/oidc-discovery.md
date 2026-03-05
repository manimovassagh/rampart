---
sidebar_position: 5
title: OIDC Discovery
description: Rampart OIDC Discovery endpoints -- OpenID Connect configuration document, JSON Web Key Set (JWKS), key rotation, and client auto-configuration examples.
---

# OIDC Discovery

Rampart publishes standard OpenID Connect Discovery endpoints that allow clients and libraries to auto-configure themselves. These endpoints are publicly accessible and do not require authentication.

---

## GET /.well-known/openid-configuration

Returns the OpenID Connect Discovery document as defined by [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html). This document advertises all endpoints, supported features, signing algorithms, and capabilities of the Rampart instance.

**Request:**

```bash
curl -X GET https://your-rampart-instance/.well-known/openid-configuration
```

**Response (200 OK):**

```json
{
  "issuer": "https://your-rampart-instance",
  "authorization_endpoint": "https://your-rampart-instance/oauth/authorize",
  "token_endpoint": "https://your-rampart-instance/oauth/token",
  "userinfo_endpoint": "https://your-rampart-instance/oauth/userinfo",
  "jwks_uri": "https://your-rampart-instance/.well-known/jwks.json",
  "revocation_endpoint": "https://your-rampart-instance/oauth/revoke",
  "introspection_endpoint": "https://your-rampart-instance/oauth/introspect",
  "device_authorization_endpoint": "https://your-rampart-instance/oauth/device/code",
  "registration_endpoint": "https://your-rampart-instance/register",
  "end_session_endpoint": "https://your-rampart-instance/oauth/logout",
  "scopes_supported": [
    "openid",
    "profile",
    "email",
    "phone",
    "offline_access"
  ],
  "response_types_supported": [
    "code"
  ],
  "response_modes_supported": [
    "query",
    "fragment"
  ],
  "grant_types_supported": [
    "authorization_code",
    "client_credentials",
    "refresh_token",
    "urn:ietf:params:oauth:grant-type:device_code"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256",
    "ES256"
  ],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "none"
  ],
  "code_challenge_methods_supported": [
    "S256"
  ],
  "claims_supported": [
    "sub",
    "iss",
    "aud",
    "exp",
    "iat",
    "nbf",
    "nonce",
    "auth_time",
    "at_hash",
    "name",
    "given_name",
    "family_name",
    "preferred_username",
    "email",
    "email_verified",
    "phone_number",
    "phone_number_verified",
    "updated_at"
  ],
  "claims_parameter_supported": false,
  "request_parameter_supported": false,
  "request_uri_parameter_supported": false,
  "require_request_uri_registration": false,
  "revocation_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post"
  ],
  "introspection_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post"
  ]
}
```

### Field Reference

| Field | Description |
|-------|-------------|
| `issuer` | The base URL of the Rampart instance. Must exactly match the `iss` claim in all issued tokens. This value is the canonical identifier for the authorization server. |
| `authorization_endpoint` | URL where clients redirect users to authenticate and authorize. Used for the Authorization Code flow. |
| `token_endpoint` | URL where clients exchange authorization codes, refresh tokens, or client credentials for access tokens. |
| `userinfo_endpoint` | URL where clients retrieve claims about the authenticated user using a valid access token. |
| `jwks_uri` | URL of the JSON Web Key Set containing the public keys used to verify token signatures. |
| `revocation_endpoint` | URL where clients revoke access or refresh tokens (RFC 7009). |
| `introspection_endpoint` | URL where resource servers validate tokens and retrieve metadata (RFC 7662). |
| `device_authorization_endpoint` | URL where devices initiate the device authorization flow (RFC 8628). |
| `registration_endpoint` | URL for user self-registration. |
| `end_session_endpoint` | URL to initiate logout and end the user's session. |
| `scopes_supported` | List of OAuth 2.0 scopes that Rampart supports. |
| `response_types_supported` | Supported response types. Rampart supports `code` (Authorization Code flow). |
| `response_modes_supported` | Supported response modes for returning authorization responses. |
| `grant_types_supported` | Supported OAuth 2.0 grant types. |
| `subject_types_supported` | Subject identifier types. `public` means the `sub` claim is the same for all clients. |
| `id_token_signing_alg_values_supported` | Algorithms used to sign ID tokens. |
| `token_endpoint_auth_methods_supported` | Client authentication methods supported at the token endpoint. |
| `code_challenge_methods_supported` | PKCE challenge methods. Only `S256` is supported (not `plain`, for security reasons). |
| `claims_supported` | Claims that may be included in ID tokens and UserInfo responses. |

### Caching

The discovery document is stable and changes only when Rampart is reconfigured. Responses include caching headers:

```
Cache-Control: public, max-age=86400
```

Clients should cache this document and refresh it at most once per day.

---

## GET /.well-known/jwks.json

Returns the JSON Web Key Set ([RFC 7517](https://datatracker.ietf.org/doc/html/rfc7517)) containing the public keys used to verify JWT signatures. Resource servers and client libraries use this endpoint to validate access tokens and ID tokens without sharing secrets with the authorization server.

**Request:**

```bash
curl -X GET https://your-rampart-instance/.well-known/jwks.json
```

**Response (200 OK):**

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "rampart-rsa-2026-03",
      "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
      "e": "AQAB"
    },
    {
      "kty": "EC",
      "use": "sig",
      "alg": "ES256",
      "kid": "rampart-ec-2026-03",
      "crv": "P-256",
      "x": "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
      "y": "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0"
    }
  ]
}
```

### RSA Key Fields

| Field | Type | Description |
|-------|------|-------------|
| `kty` | string | Key type. `RSA` for RSA keys. |
| `use` | string | Key usage. `sig` indicates this key is used for signing (not encryption). |
| `alg` | string | Algorithm. `RS256` (RSASSA-PKCS1-v1_5 using SHA-256). |
| `kid` | string | Key ID. Used to match a specific key when multiple keys are present (e.g., during rotation). The `kid` in the JWT header corresponds to this value. |
| `n` | string | RSA modulus (Base64url-encoded). |
| `e` | string | RSA public exponent (Base64url-encoded). Typically `AQAB` (65537). |

### EC Key Fields

| Field | Type | Description |
|-------|------|-------------|
| `kty` | string | Key type. `EC` for elliptic curve keys. |
| `use` | string | Key usage. `sig` for signing. |
| `alg` | string | Algorithm. `ES256` (ECDSA using P-256 and SHA-256). |
| `kid` | string | Key ID. |
| `crv` | string | Curve name. `P-256` for ES256. |
| `x` | string | X coordinate (Base64url-encoded). |
| `y` | string | Y coordinate (Base64url-encoded). |

### Caching

The JWKS endpoint returns standard HTTP caching headers:

```
Cache-Control: public, max-age=3600
ETag: "jwks-v3-2026-03-01"
```

Clients should cache the JWKS and avoid fetching it on every token verification. Re-fetch only when:
1. The cache has expired (respect `max-age`).
2. A token's `kid` does not match any key in the cached JWKS.

---

## Key Rotation

Rampart supports zero-downtime key rotation. When signing keys are rotated, both the old and new keys are published in the JWKS simultaneously for a configurable transition period (default: 7 days).

### Rotation Process

1. **New key generated**: Rampart generates a new signing key pair and begins signing new tokens with the new key.
2. **Both keys published**: The JWKS contains both the old key (for verifying previously issued tokens) and the new key (for verifying newly issued tokens).
3. **Transition period**: During this period (default: 7 days), tokens signed with the old key are still valid and verifiable.
4. **Old key removed**: After the transition period, the old key is removed from the JWKS. Tokens signed with the old key will fail verification if they have not yet expired.

### Handling Rotation in Your Application

The recommended approach for resource servers and clients:

```
1. Fetch the JWKS on startup and cache it.
2. When verifying a token:
   a. Extract the "kid" from the token's JWT header.
   b. Look up the matching key in the cached JWKS.
   c. If the "kid" is found, verify the token signature.
   d. If the "kid" is NOT found, re-fetch the JWKS from the server.
   e. If the "kid" is still not found after re-fetching, reject the token.
3. Re-fetch the JWKS periodically (e.g., every hour) to stay current.
```

This approach handles key rotation seamlessly: when a new key appears, the first token signed with it triggers a JWKS refresh, and verification proceeds normally.

### Triggering Key Rotation

Key rotation can be triggered via the Admin API:

```bash
curl -X POST https://your-rampart-instance/api/v1/admin/keys/rotate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "algorithm": "RS256",
    "transition_period": "7d"
  }'
```

**Response (200 OK):**

```json
{
  "new_kid": "rampart-rsa-2026-03-05",
  "old_kid": "rampart-rsa-2026-03",
  "algorithm": "RS256",
  "transition_ends_at": "2026-03-12T10:00:00Z",
  "message": "New signing key generated. Both keys are published in the JWKS. The old key will be removed after the transition period."
}
```

---

## How Clients Use Discovery

Most OAuth 2.0 and OIDC client libraries accept a discovery URL and auto-configure all endpoints, keys, and capabilities. Here are examples for common platforms.

### JavaScript / TypeScript

```javascript
// Using openid-client (Node.js)
const { Issuer } = require("openid-client");

const rampartIssuer = await Issuer.discover(
  "https://your-rampart-instance/.well-known/openid-configuration"
);

console.log("Issuer:", rampartIssuer.issuer);
console.log("Token endpoint:", rampartIssuer.token_endpoint);

const client = new rampartIssuer.Client({
  client_id: "my-app",
  client_secret: "my-secret",
  redirect_uris: ["https://app.example.com/callback"],
  response_types: ["code"],
});
```

### Go

```go
package main

import (
    "context"
    "github.com/coreos/go-oidc/v3/oidc"
    "golang.org/x/oauth2"
)

func main() {
    ctx := context.Background()

    // Auto-discover endpoints from the OIDC configuration
    provider, err := oidc.NewProvider(ctx, "https://your-rampart-instance")
    if err != nil {
        panic(err)
    }

    // Configure OAuth2
    oauth2Config := oauth2.Config{
        ClientID:     "my-app",
        ClientSecret: "my-secret",
        RedirectURL:  "https://app.example.com/callback",
        Endpoint:     provider.Endpoint(),
        Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
    }

    // Create a token verifier (uses JWKS automatically)
    verifier := provider.Verifier(&oidc.Config{
        ClientID: "my-app",
    })

    // Verify an ID token
    idToken, err := verifier.Verify(ctx, rawIDToken)
    if err != nil {
        panic(err)
    }
}
```

### Python

```python
# Using PyJWT for token verification
import jwt
from jwt import PyJWKClient

# Fetch and cache JWKS automatically
jwks_client = PyJWKClient(
    "https://your-rampart-instance/.well-known/jwks.json",
    cache_jwk_set=True,
    lifespan=3600  # Cache for 1 hour
)

def verify_token(token: str) -> dict:
    """Verify a Rampart-issued JWT token."""
    # Automatically matches kid and fetches the correct key
    signing_key = jwks_client.get_signing_key_from_jwt(token)

    payload = jwt.decode(
        token,
        signing_key.key,
        algorithms=["RS256", "ES256"],
        audience="my-app",
        issuer="https://your-rampart-instance",
        options={
            "verify_exp": True,
            "verify_iss": True,
            "verify_aud": True,
        }
    )
    return payload
```

```python
# Using Authlib for full OIDC client
from authlib.integrations.requests_client import OAuth2Session

session = OAuth2Session(
    client_id="my-app",
    client_secret="my-secret",
    redirect_uri="https://app.example.com/callback",
    scope="openid profile email"
)

# Auto-discover endpoints
metadata = session.fetch_server_metadata(
    "https://your-rampart-instance"
)

# Get authorization URL
auth_url, state = session.create_authorization_url(
    metadata["authorization_endpoint"]
)
```

### Java / Spring Boot

```java
// application.yml -- Spring Security auto-discovers all endpoints
spring:
  security:
    oauth2:
      resourceserver:
        jwt:
          issuer-uri: https://your-rampart-instance
      client:
        registration:
          rampart:
            client-id: my-app
            client-secret: my-secret
            scope: openid,profile,email
            authorization-grant-type: authorization_code
            redirect-uri: "{baseUrl}/login/oauth2/code/rampart"
        provider:
          rampart:
            issuer-uri: https://your-rampart-instance
```

Spring Security reads the discovery document at startup and automatically configures the authorization endpoint, token endpoint, JWKS URI, and token verification.

### curl

```bash
# Fetch discovery document
curl -s https://your-rampart-instance/.well-known/openid-configuration | jq .

# Extract the JWKS URI from the discovery document
JWKS_URI=$(curl -s https://your-rampart-instance/.well-known/openid-configuration | jq -r .jwks_uri)

# Fetch the JWKS
curl -s "$JWKS_URI" | jq .

# List all key IDs
curl -s "$JWKS_URI" | jq '.keys[].kid'
```

---

## Verifying Tokens with the JWKS

### Token Verification Steps

When verifying a JWT (access token or ID token), follow these steps:

1. **Decode the JWT header** (without verifying the signature) to extract the `kid` and `alg` claims.
2. **Fetch the JWKS** from `/.well-known/jwks.json` (use a cached copy if available).
3. **Find the matching key** in the JWKS where `kid` matches the token's `kid`.
4. **Verify the signature** using the matching public key and the algorithm specified in the JWT header.
5. **Validate standard claims:**
   - `iss` must match your Rampart instance URL exactly.
   - `aud` must include your client ID.
   - `exp` must be in the future.
   - `nbf` (if present) must be in the past.
   - `iat` should be reasonable (not far in the future).
6. **Validate the `nonce`** (for ID tokens) if you provided one in the authorization request.

### JWT Header Example

```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "rampart-rsa-2026-03"
}
```

The `kid` value (`rampart-rsa-2026-03`) is used to look up the correct public key in the JWKS.

### Important Verification Notes

- **Always verify the signature.** Never trust a JWT without cryptographic verification.
- **Always validate the `iss` claim.** This prevents tokens from other authorization servers from being accepted.
- **Always validate the `aud` claim.** This prevents tokens intended for other clients from being accepted.
- **Use the `kid` for key lookup.** Do not try all keys in the JWKS -- match by `kid` first.
- **Handle key rotation.** If a `kid` is not found in your cached JWKS, re-fetch the JWKS once and try again.
- **Never use `alg: none`.** Rampart never issues unsigned tokens, and your verification code should reject them.
- **Pin the expected algorithms.** Only accept `RS256` and `ES256` (or whichever algorithms your Rampart instance is configured to use). Do not accept arbitrary algorithms from the JWT header.
