# @rampart-auth/web

Browser authentication SDK for [Rampart](https://github.com/manimovassagh/rampart). Implements the OAuth 2.0 Authorization Code flow with PKCE using the Web Crypto API — no backend proxy required.

## Install

```bash
npm install @rampart-auth/web
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

## API

### `new RampartClient(config: RampartClientConfig)`

Creates a client instance. Call once at app startup.

#### `RampartClientConfig`

| Property        | Type       | Default    | Description                                                        |
|-----------------|------------|------------|--------------------------------------------------------------------|
| `issuer`        | `string`   | —          | Rampart server URL (e.g. `http://localhost:8080`)                  |
| `clientId`      | `string`   | —          | OAuth 2.0 client ID registered with the Rampart server             |
| `redirectUri`   | `string`   | —          | OAuth 2.0 redirect URI — must exactly match a registered redirect  |
| `scope`         | `string?`  | `"openid"` | OAuth 2.0 scopes                                                   |
| `onTokenChange` | `function?`| —          | Called with `RampartTokens \| null` on every token change           |

### Methods

#### `loginWithRedirect(): Promise<void>`

Generates a PKCE code verifier and challenge, stores them in `sessionStorage`, and redirects the browser to the Rampart authorization endpoint.

#### `handleCallback(url?: string): Promise<RampartTokens>`

Handles the OAuth callback. Extracts `code` and `state` from the URL (defaults to `window.location.href`), validates the state against `sessionStorage`, and exchanges the code for tokens. Returns the tokens and stores them on the client.

#### `getUser(): Promise<RampartUser>`

Fetches the current user profile from the `/me` endpoint using `authFetch`.

#### `authFetch(url: string, init?: RequestInit): Promise<Response>`

`fetch` wrapper that attaches the `Authorization: Bearer` header automatically. On a 401 response, attempts one silent token refresh and retries the request.

#### `refresh(): Promise<RampartTokens>`

Refreshes the access token using the stored refresh token. Clears tokens on failure.

#### `logout(): Promise<void>`

Invalidates the refresh token on the server and clears local tokens.

#### `isAuthenticated(): boolean`

Returns `true` if an access token is present. Does not verify token expiry.

#### `getAccessToken(): string | null`

Returns the current access token, or `null`.

#### `getTokens(): RampartTokens | null`

Returns a copy of the current tokens, or `null`.

#### `setTokens(tokens: RampartTokens | null): void`

Restores tokens from external storage (e.g. `localStorage`). Triggers `onTokenChange`.

### `RampartTokens`

| Field           | Type     | Description                   |
|-----------------|----------|-------------------------------|
| `access_token`  | `string` | JWT access token              |
| `refresh_token` | `string` | Refresh token                 |
| `token_type`    | `string` | Token type (typically `Bearer`)|
| `expires_in`    | `number` | Token lifetime in seconds     |

### `RampartUser`

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

### `RampartError`

All API errors are thrown as `RampartError` objects:

```typescript
interface RampartError {
  error: string;            // e.g. "invalid_callback", "state_mismatch"
  error_description: string;
  status: number;           // HTTP status, or 0 for client-side errors
}
```

## Token Persistence

The client does not persist tokens by default — they live only in memory. To persist across page reloads, use the `onTokenChange` callback:

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

## License

MIT
