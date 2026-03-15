# @rampart-auth/nextjs

Rampart authentication adapter for Next.js App Router. Provides Edge Middleware auth guards, server-side JWT validation, and client-side session hooks.

## Install

```bash
npm install @rampart-auth/nextjs @rampart-auth/web @rampart-auth/react
```

Peer dependencies: `next >=14`, `react >=18`, `@rampart-auth/web >=0.1.0`, `@rampart-auth/react >=0.1.0`.

## Minimal Setup Checklist

A working integration requires three files:

| # | File | Purpose |
|---|------|---------|
| 1 | `middleware.ts` (project root) | Edge middleware — intercepts every request, validates the JWT from cookie or `Authorization` header, redirects unauthenticated users |
| 2 | `app/api/auth/session/route.ts` | Session API endpoint — exposes the validated claims to client components via `useRampartSession` |
| 3 | `app/layout.tsx` | `RampartProvider` wrapper — supplies auth context (`useAuth`, `ProtectedRoute`) to the client component tree |

All three files are shown in the sections below.

## Environment Variables

```bash
# Required — your Rampart server's issuer URL (no trailing slash)
RAMPART_ISSUER=https://auth.example.com

# Required for RampartProvider (SPA / PKCE flow)
NEXT_PUBLIC_RAMPART_ISSUER=https://auth.example.com
NEXT_PUBLIC_RAMPART_CLIENT_ID=my-app
NEXT_PUBLIC_RAMPART_REDIRECT_URI=http://localhost:3000/callback
```

## 1. Edge Middleware

Create `middleware.ts` in your project root:

```ts
import { withRampartAuth } from "@rampart-auth/nextjs/middleware";

export default withRampartAuth({
  issuer: process.env.RAMPART_ISSUER!,
  publicPaths: ["/", "/login", "/callback", "/api/public/*"],
  cookieName: "rampart_token", // default
  loginPath: "/login",         // default
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
```

### Middleware configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `issuer` | `string` | (required) | Rampart server URL. Trailing slashes are stripped automatically. JWKS is fetched from `{issuer}/.well-known/jwks.json`. |
| `publicPaths` | `string[]` | `[]` | Paths that skip auth. Supports prefix matching with a trailing `*` (e.g. `"/api/public/*"` matches `"/api/public/foo"`). Exact match otherwise. |
| `cookieName` | `string` | `"rampart_token"` | Cookie name that holds the access token. |
| `loginPath` | `string` | `"/login"` | Where to redirect unauthenticated users. |

### How it works

1. If the path matches `publicPaths`, the request passes through with no auth check.
2. Otherwise, the middleware extracts a token from the cookie (`cookieName`) or the `Authorization: Bearer <token>` header.
3. The token is verified against the issuer's JWKS using RS256.
4. On success, a `x-rampart-claims` response header is set with the JSON-encoded claims.
5. On failure (no token, expired, invalid signature), the user is redirected to `{loginPath}?callbackUrl={originalPath}`.

## 2. Server Components & Route Handlers

### getServerAuth

Read the authenticated user in Server Components or Route Handlers:

```ts
import { cookies } from "next/headers";
import { getServerAuth } from "@rampart-auth/nextjs/server";
import { redirect } from "next/navigation";

export default async function DashboardPage() {
  const auth = await getServerAuth(await cookies(), process.env.RAMPART_ISSUER!);

  if (!auth) {
    redirect("/login");
  }

  return <h1>Welcome, {auth.claims.preferred_username}</h1>;
}
```

`getServerAuth` accepts an optional third argument for the cookie name (default: `"rampart_token"`). It returns `ServerAuth | null`:

```ts
interface ServerAuth {
  claims: RampartClaims;  // decoded JWT claims
  token: string;          // raw JWT string (useful for calling backend APIs)
}
```

### validateToken

Validate an arbitrary token string directly:

```ts
import { validateToken } from "@rampart-auth/nextjs/server";

const claims = await validateToken(token, "https://auth.example.com");
if (!claims) {
  // invalid or expired
}
```

Returns `RampartClaims | null`.

## 3. Client-Side Session Hook

### Session API route

Create `app/api/auth/session/route.ts`:

```ts
import { cookies } from "next/headers";
import { getServerAuth } from "@rampart-auth/nextjs/server";

export async function GET() {
  const auth = await getServerAuth(await cookies(), process.env.RAMPART_ISSUER!);
  if (!auth) return Response.json({ claims: null });
  return Response.json({ claims: auth.claims });
}
```

### useRampartSession hook

Use the hook in client components:

```tsx
"use client";

import { useRampartSession } from "@rampart-auth/nextjs/client";

export function UserGreeting() {
  const { claims, isLoading, isAuthenticated, refresh } = useRampartSession();

  if (isLoading) return <p>Loading...</p>;
  if (!isAuthenticated) return <p>Not signed in</p>;

  return <p>Hello, {claims!.preferred_username}</p>;
}
```

The hook returns:

| Property | Type | Description |
|----------|------|-------------|
| `claims` | `RampartClaims \| null` | Decoded JWT claims, or `null` if not authenticated |
| `isLoading` | `boolean` | `true` while the session fetch is in flight |
| `isAuthenticated` | `boolean` | `true` when `claims` is not null |
| `refresh` | `() => Promise<void>` | Re-fetches the session from the server |

Options:

```ts
useRampartSession({ sessionEndpoint: "/api/custom/session" });
```

Default endpoint: `"/api/auth/session"`.

## 4. Client-Side Auth (SPA / PKCE Flow)

The `@rampart-auth/react` hooks are re-exported from `@rampart-auth/nextjs/client` for convenience. Wrap your layout:

```tsx
// app/layout.tsx
import { RampartProvider } from "@rampart-auth/nextjs/client";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <RampartProvider
          issuer={process.env.NEXT_PUBLIC_RAMPART_ISSUER!}
          clientId={process.env.NEXT_PUBLIC_RAMPART_CLIENT_ID!}
          redirectUri={process.env.NEXT_PUBLIC_RAMPART_REDIRECT_URI!}
        >
          {children}
        </RampartProvider>
      </body>
    </html>
  );
}
```

Available re-exports from `@rampart-auth/nextjs/client`:

- `RampartProvider` -- auth context provider
- `useAuth` -- returns `{ user, isAuthenticated, isLoading, login, logout, getToken }`
- `ProtectedRoute` -- renders children only when authenticated
- `RampartContext` -- raw React context (for advanced use)

## Types

```ts
import type {
  RampartClaims,
  RampartMiddlewareConfig,
  ServerAuth,
} from "@rampart-auth/nextjs";
```

### RampartClaims

| Field | Type | Always present |
|-------|------|---------------|
| `iss` | `string` | yes |
| `sub` | `string` | yes |
| `iat` | `number` | yes |
| `exp` | `number` | yes |
| `org_id` | `string` | yes |
| `preferred_username` | `string` | yes |
| `email` | `string` | yes |
| `email_verified` | `boolean` | yes |
| `given_name` | `string` | no |
| `family_name` | `string` | no |
| `roles` | `string[]` | no |

## Error Handling

Each layer handles auth failures differently:

### Middleware (`withRampartAuth`)

On auth failure (missing token, expired JWT, invalid signature), the middleware **silently redirects** to `{loginPath}?callbackUrl={originalPath}`. There is no error thrown or logged -- the user simply lands on the login page. After the user authenticates, your login page can read the `callbackUrl` query parameter to redirect back.

### Server (`getServerAuth` / `validateToken`)

Both functions return **`null` on any failure** -- they never throw. This means:

- Missing cookie --> `null`
- Expired token --> `null`
- Invalid signature --> `null`
- Network error fetching JWKS --> `null`

Always check the return value before accessing claims.

### Client (`useRampartSession`)

The hook calls `fetch()` against the session endpoint. On any error (network failure, non-2xx response), `claims` is set to `null` and `isAuthenticated` becomes `false`. No error is thrown to the component.

For SPA flows (`useAuth` from `@rampart-auth/react`), errors from the underlying `@rampart-auth/web` library are surfaced through the `error` property on the auth context.

## Troubleshooting

### Infinite redirect loop

**Symptom:** The browser keeps redirecting between your app and the login page.

**Cause:** The login page itself is not listed in `publicPaths`, so the middleware redirects it to... the login page.

**Fix:** Ensure `publicPaths` includes your login page and your OAuth callback path:

```ts
withRampartAuth({
  issuer: process.env.RAMPART_ISSUER!,
  publicPaths: ["/", "/login", "/callback"],
  //                   ^^^^^^^  ^^^^^^^^^
});
```

### Session API returning null when the user is logged in

**Symptom:** `useRampartSession` always shows `isAuthenticated: false` even though the middleware is not redirecting.

**Possible causes:**

1. **Cookie name mismatch.** The default cookie name is `"rampart_token"`. If your Rampart server sets a different cookie name, pass it to both the middleware config and `getServerAuth`:
   ```ts
   // middleware.ts
   withRampartAuth({ cookieName: "my_token", ... });

   // app/api/auth/session/route.ts
   getServerAuth(await cookies(), issuer, "my_token");
   ```

2. **Issuer URL mismatch.** The `issuer` passed to `getServerAuth` must exactly match the `iss` claim in the JWT (after trailing-slash stripping). Check your `RAMPART_ISSUER` env var.

3. **Session endpoint path.** By default, `useRampartSession` fetches from `/api/auth/session`. If your route handler is at a different path, pass it explicitly:
   ```ts
   useRampartSession({ sessionEndpoint: "/api/my-session" });
   ```

### CORS errors when calling the Rampart server

**Symptom:** Browser console shows `Access-Control-Allow-Origin` errors when the SPA flow tries to exchange tokens.

**Fix:** Configure your Rampart server to allow your Next.js origin. The `@rampart-auth/web` library makes direct browser requests to the issuer for token exchange, JWKS fetching, and userinfo -- all of these require CORS headers from the Rampart server.

### Token expired but no redirect

**Symptom:** Server components return `null` from `getServerAuth` but the middleware does not redirect.

**Cause:** Edge Middleware and Server Components run in different contexts. The middleware validates the token at the edge, but if the token expires between the edge check and the server render, `getServerAuth` will return `null`. Handle this in your server component:

```ts
const auth = await getServerAuth(await cookies(), process.env.RAMPART_ISSUER!);
if (!auth) {
  redirect("/login");
}
```

## Package Exports

| Import path | Runtime | Contents |
|-------------|---------|----------|
| `@rampart-auth/nextjs` | any | All types + re-exports from all sub-paths |
| `@rampart-auth/nextjs/middleware` | Edge | `withRampartAuth` |
| `@rampart-auth/nextjs/server` | Node.js | `getServerAuth`, `validateToken` |
| `@rampart-auth/nextjs/client` | Browser | `useRampartSession`, `RampartProvider`, `useAuth`, `ProtectedRoute`, `RampartContext` |

## Build

```bash
npm run build   # uses tsup
npm run lint    # type-check with tsc --noEmit
npm run test    # vitest
```
