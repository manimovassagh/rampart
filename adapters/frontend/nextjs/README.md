# @rampart-auth/nextjs

Rampart authentication adapter for Next.js App Router. Provides Edge Middleware auth guards, server-side JWT validation, and client-side session hooks.

## Install

```bash
npm install @rampart-auth/nextjs @rampart-auth/web @rampart-auth/react
```

## Edge Middleware

Protect routes with JWT validation at the edge. Create `middleware.ts` in your project root:

```ts
import { withRampartAuth } from "@rampart-auth/nextjs/middleware";

export default withRampartAuth({
  issuer: "https://auth.example.com",
  publicPaths: ["/", "/login", "/api/public/*"],
  cookieName: "rampart_token", // default
  loginPath: "/login",         // default
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
```

Unauthenticated requests are redirected to `/login?callbackUrl=/original-path`.

## Server Components & Route Handlers

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

### Validate a token directly

```ts
import { validateToken } from "@rampart-auth/nextjs/server";

const claims = await validateToken(token, "https://auth.example.com");
if (!claims) {
  // invalid or expired
}
```

## Client-Side Session Hook

Create a session API route:

```ts
// app/api/auth/session/route.ts
import { cookies } from "next/headers";
import { getServerAuth } from "@rampart-auth/nextjs/server";

export async function GET() {
  const auth = await getServerAuth(await cookies(), process.env.RAMPART_ISSUER!);
  if (!auth) return Response.json({ claims: null });
  return Response.json({ claims: auth.claims });
}
```

Then use the hook in client components:

```tsx
"use client";

import { useRampartSession } from "@rampart-auth/nextjs/client";

export function UserGreeting() {
  const { claims, isLoading, isAuthenticated } = useRampartSession();

  if (isLoading) return <p>Loading...</p>;
  if (!isAuthenticated) return <p>Not signed in</p>;

  return <p>Hello, {claims!.preferred_username}</p>;
}
```

## Client-Side Auth (SPA flows)

The `@rampart-auth/react` hooks are re-exported from `@rampart-auth/nextjs/client` for convenience:

```tsx
"use client";

import { RampartProvider, useAuth, ProtectedRoute } from "@rampart-auth/nextjs/client";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <RampartProvider
      issuer="https://auth.example.com"
      clientId="my-app"
      redirectUri="http://localhost:3000/callback"
    >
      {children}
    </RampartProvider>
  );
}
```

## Types

```ts
import type { RampartClaims, RampartMiddlewareConfig, ServerAuth } from "@rampart-auth/nextjs";
```

## Build

```bash
npm run build   # uses tsup
npm run lint    # type-check
```
