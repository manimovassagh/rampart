---
sidebar_position: 8
title: Web / JavaScript
description: Integrate Rampart authentication into any browser application using the @rampart-auth/web SDK with OAuth 2.0 PKCE.
---

# Web / JavaScript Adapter

The `@rampart-auth/web` adapter is a framework-agnostic browser authentication SDK for Rampart. It implements the OAuth 2.0 Authorization Code flow with PKCE using the Web Crypto API -- no backend proxy or Node.js polyfills required. Use it directly in any SPA, or as the foundation for framework-specific adapters like `@rampart-auth/react`.

## Installation

```bash
npm install @rampart-auth/web
```

```bash
yarn add @rampart-auth/web
```

```bash
pnpm add @rampart-auth/web
```

## Quick Start

```typescript
import { RampartClient } from "@rampart-auth/web";

const client = new RampartClient({
  issuer: "http://localhost:8080",
  clientId: "my-spa",
  redirectUri: "http://localhost:3000/callback",
});

// Start login — redirects the browser to Rampart
await client.loginWithRedirect();

// On the callback page — exchange the authorization code for tokens
await client.handleCallback();

// Fetch the authenticated user profile
const user = await client.getUser();
```

## Configuration

### `new RampartClient(config)`

Creates a client instance. Call once at app startup.

```typescript
const client = new RampartClient({
  // Required
  issuer: "https://auth.example.com",
  clientId: "my-spa",
  redirectUri: "http://localhost:3000/callback",

  // Optional
  scope: "openid profile email",       // OAuth 2.0 scopes (default: "openid")
  onTokenChange: (tokens) => {         // Called on every token change
    if (tokens) {
      localStorage.setItem("rampart_tokens", JSON.stringify(tokens));
    } else {
      localStorage.removeItem("rampart_tokens");
    }
  },
});
```

### `RampartClientConfig`

| Property        | Type       | Default    | Description                                                        |
|-----------------|------------|------------|--------------------------------------------------------------------|
| `issuer`        | `string`   | --          | Rampart server URL (e.g. `http://localhost:8080`)                  |
| `clientId`      | `string`   | --          | OAuth 2.0 client ID registered with the Rampart server             |
| `redirectUri`   | `string`   | --          | OAuth 2.0 redirect URI -- must exactly match a registered redirect  |
| `scope`         | `string?`  | `"openid"` | OAuth 2.0 scopes                                                   |
| `onTokenChange` | `function?`| --          | Called with `RampartTokens | null` on every token change           |

## OAuth 2.0 PKCE Flow

The adapter implements the full Authorization Code flow with PKCE (RFC 7636):

1. **Login initiated** -- `loginWithRedirect()` generates a cryptographic `code_verifier` and derives a `code_challenge` using SHA-256
2. **Redirect to Rampart** -- sends `code_challenge` and `code_challenge_method=S256` in the authorization request
3. **User authenticates** -- at the Rampart login page
4. **Callback received** -- Rampart redirects back with an authorization `code`
5. **Token exchange** -- `handleCallback()` sends the `code` and `code_verifier` to the token endpoint
6. **Tokens stored** -- access token and refresh token are stored in memory on the client

No client secret is ever used or stored in the browser.

## Methods

### `loginWithRedirect()`

Generates a PKCE code verifier and challenge, stores them in `sessionStorage`, and redirects the browser to the Rampart authorization endpoint.

```typescript
await client.loginWithRedirect();
```

### `handleCallback(url?)`

Handles the OAuth callback. Extracts `code` and `state` from the URL (defaults to `window.location.href`), validates the state against `sessionStorage`, and exchanges the code for tokens.

```typescript
// On your /callback page
const tokens = await client.handleCallback();
console.log(tokens.access_token);
```

### `getUser()`

Fetches the current user profile from the `/me` endpoint using `authFetch`.

```typescript
const user = await client.getUser();
console.log(user.email, user.roles);
```

### `authFetch(url, init?)`

`fetch` wrapper that attaches the `Authorization: Bearer` header automatically. On a 401 response, attempts one silent token refresh and retries the request.

```typescript
const res = await client.authFetch("https://api.example.com/tasks");
const data = await res.json();
```

### `refresh()`

Refreshes the access token using the stored refresh token. Clears tokens on failure.

```typescript
const newTokens = await client.refresh();
```

### `logout()`

Invalidates the refresh token on the server and clears local tokens.

```typescript
await client.logout();
```

### `isAuthenticated()`

Returns `true` if an access token is present and not expired. Checks expiry by decoding the JWT payload client-side (without cryptographic verification).

```typescript
if (client.isAuthenticated()) {
  // user is logged in
}
```

### `getAccessToken()` / `getTokens()` / `setTokens(tokens)`

```typescript
const token = client.getAccessToken();    // string | null
const tokens = client.getTokens();        // RampartTokens | null
client.setTokens(restoredTokens);         // restore from external storage
```

## Token Management

### Token Types

#### `RampartTokens`

| Field           | Type     | Description                    |
|-----------------|----------|--------------------------------|
| `access_token`  | `string` | JWT access token               |
| `refresh_token` | `string` | Refresh token                  |
| `token_type`    | `string` | Token type (typically `Bearer`)|
| `expires_in`    | `number` | Token lifetime in seconds      |

#### `RampartUser`

| Field                | Type       | Description              |
|----------------------|------------|--------------------------|
| `id`                 | `string`   | User ID (UUID)           |
| `org_id`             | `string`   | Organization ID (UUID)   |
| `preferred_username` | `string?`  | Preferred username       |
| `username`           | `string?`  | Username                 |
| `email`              | `string`   | Email address            |
| `email_verified`     | `boolean`  | Whether email is verified|
| `given_name`         | `string?`  | First name               |
| `family_name`        | `string?`  | Last name                |
| `roles`              | `string[]?`| Assigned roles           |
| `enabled`            | `boolean?` | Whether account is active|
| `created_at`         | `string?`  | ISO 8601 timestamp       |
| `updated_at`         | `string?`  | ISO 8601 timestamp       |

### Token Persistence

The client does not persist tokens by default -- they live only in memory. To persist across page reloads, use the `onTokenChange` callback:

```typescript
const client = new RampartClient({
  issuer: "http://localhost:8080",
  clientId: "my-spa",
  redirectUri: "http://localhost:3000/callback",
  onTokenChange: (tokens) => {
    if (tokens) {
      localStorage.setItem("rampart_tokens", JSON.stringify(tokens));
    } else {
      localStorage.removeItem("rampart_tokens");
    }
  },
});

// Restore tokens on startup
const stored = localStorage.getItem("rampart_tokens");
if (stored) {
  client.setTokens(JSON.parse(stored));
}
```

## Error Handling

All API errors are thrown as `RampartError` objects:

```typescript
interface RampartError {
  error: string;            // e.g. "invalid_callback", "state_mismatch"
  error_description: string;
  status: number;           // HTTP status, or 0 for client-side errors
}
```

Handle errors with try/catch:

```typescript
try {
  await client.handleCallback();
} catch (err) {
  if (err.error === "state_mismatch") {
    console.error("OAuth state mismatch -- possible CSRF attack");
  } else {
    console.error(err.error_description);
  }
}
```

## Full Working Example

```typescript
import { RampartClient } from "@rampart-auth/web";

// --- Initialize ---
const client = new RampartClient({
  issuer: "http://localhost:8080",
  clientId: "my-spa",
  redirectUri: window.location.origin + "/callback",
  scope: "openid profile email",
  onTokenChange: (tokens) => {
    if (tokens) {
      localStorage.setItem("rampart_tokens", JSON.stringify(tokens));
    } else {
      localStorage.removeItem("rampart_tokens");
    }
  },
});

// Restore tokens on page load
const stored = localStorage.getItem("rampart_tokens");
if (stored) {
  client.setTokens(JSON.parse(stored));
}

// --- Routing ---
const path = window.location.pathname;

if (path === "/callback") {
  // Handle the OAuth callback
  try {
    await client.handleCallback();
    window.location.href = "/dashboard";
  } catch (err) {
    document.body.textContent = `Login failed: ${err.error_description}`;
  }
} else if (path === "/dashboard") {
  if (!client.isAuthenticated()) {
    await client.loginWithRedirect();
  } else {
    const user = await client.getUser();
    document.body.innerHTML = `
      <h1>Welcome, ${user.email}</h1>
      <p>Roles: ${user.roles?.join(", ") ?? "none"}</p>
      <button id="logout">Log out</button>
    `;
    document.getElementById("logout")!.onclick = () => client.logout();

    // Make an authenticated API call
    const res = await client.authFetch("/api/tasks");
    const tasks = await res.json();
    console.log("Tasks:", tasks);
  }
} else {
  // Landing page
  document.body.innerHTML = `
    <h1>My App</h1>
    <button id="login">Log in</button>
  `;
  document.getElementById("login")!.onclick = () => client.loginWithRedirect();
}
```

## Security Considerations

- **Never store client secrets in frontend code.** The Web adapter uses PKCE, which does not require a client secret.
- **Use `sessionStorage` for PKCE state.** The adapter stores `code_verifier` and `state` in `sessionStorage` automatically.
- **Consider token storage tradeoffs.** In-memory storage (default) is the most secure but tokens are lost on page reload. `localStorage` persists across reloads but is accessible to XSS attacks. Choose based on your threat model.
- **Set short access token lifetimes** (5-15 minutes) and rely on `authFetch` automatic refresh for seamless UX.
- **Always use HTTPS in production.** Token transmission over HTTP is insecure.
- **Configure CORS on your API** to only accept requests from your SPA's origin.
