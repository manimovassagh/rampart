---
name: rampart-nextjs-setup
description: Add Rampart authentication to a Next.js app. Sets up server-side JWT validation, API route protection, middleware auth guards, and client-side auth context. Use when securing a Next.js app with Rampart.
argument-hint: [issuer-url]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Add Rampart Authentication to a Next.js App

Set up JWT-based auth in a Next.js app with server-side validation and client-side auth context.

## What This Skill Does

1. Installs `jose` for JWT verification
2. Creates server-side auth utilities (validate tokens, protect API routes)
3. Adds Next.js middleware for route protection
4. Sets up client-side auth context with login/logout
5. Handles OAuth 2.0 PKCE flow with Rampart

## Step-by-Step

### 1. Install dependencies

```bash
npm install jose
```

### 2. Create server-side auth utility

Create `lib/auth.ts`:

```typescript
import { jwtVerify, createRemoteJWKSSet } from "jose";

const issuer = process.env.RAMPART_ISSUER || "$ARGUMENTS" || "http://localhost:8080";
const JWKS = createRemoteJWKSSet(new URL("/.well-known/jwks.json", issuer));

export interface RampartClaims {
  sub: string;
  iss: string;
  exp: number;
  iat: number;
  org_id: string;
  preferred_username: string;
  email: string;
  email_verified: boolean;
  given_name?: string;
  family_name?: string;
}

export async function verifyToken(token: string): Promise<RampartClaims | null> {
  try {
    const { payload } = await jwtVerify(token, JWKS, { issuer });
    return payload as unknown as RampartClaims;
  } catch {
    return null;
  }
}

export function getTokenFromHeader(authHeader: string | null): string | null {
  if (!authHeader?.startsWith("Bearer ")) return null;
  return authHeader.slice(7);
}
```

### 3. Create API route helper

Create `lib/with-auth.ts`:

```typescript
import { NextRequest, NextResponse } from "next/server";
import { verifyToken, getTokenFromHeader, RampartClaims } from "./auth";

type AuthenticatedHandler = (
  req: NextRequest,
  context: { claims: RampartClaims }
) => Promise<NextResponse> | NextResponse;

export function withAuth(handler: AuthenticatedHandler) {
  return async (req: NextRequest) => {
    const token = getTokenFromHeader(req.headers.get("authorization"));
    if (!token) {
      return NextResponse.json(
        { error: "unauthorized", error_description: "Missing authorization header." },
        { status: 401 }
      );
    }
    const claims = await verifyToken(token);
    if (!claims) {
      return NextResponse.json(
        { error: "unauthorized", error_description: "Invalid or expired access token." },
        { status: 401 }
      );
    }
    return handler(req, { claims });
  };
}
```

### 4. Protect an API route (App Router)

Create `app/api/profile/route.ts`:

```typescript
import { NextResponse } from "next/server";
import { withAuth } from "@/lib/with-auth";

export const GET = withAuth(async (req, { claims }) => {
  return NextResponse.json({
    id: claims.sub,
    email: claims.email,
    username: claims.preferred_username,
    org: claims.org_id,
  });
});
```

### 5. Add Next.js middleware for page protection

Create `middleware.ts` in the project root:

```typescript
import { NextRequest, NextResponse } from "next/server";

const protectedPaths = ["/dashboard", "/settings", "/profile"];
const issuer = process.env.RAMPART_ISSUER || "http://localhost:8080";
const clientId = process.env.RAMPART_CLIENT_ID || "my-nextjs-app";

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;

  // Skip non-protected paths
  if (!protectedPaths.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  // Check for session cookie or token
  const token = req.cookies.get("rampart_token")?.value;
  if (token) return NextResponse.next();

  // Redirect to Rampart login
  const callbackUrl = new URL("/api/auth/callback", req.url);
  const loginUrl = new URL("/oauth/authorize", issuer);
  loginUrl.searchParams.set("response_type", "code");
  loginUrl.searchParams.set("client_id", clientId);
  loginUrl.searchParams.set("redirect_uri", callbackUrl.toString());
  loginUrl.searchParams.set("scope", "openid profile email");

  return NextResponse.redirect(loginUrl);
}

export const config = {
  matcher: ["/dashboard/:path*", "/settings/:path*", "/profile/:path*"],
};
```

### 6. Create the OAuth callback API route

Create `app/api/auth/callback/route.ts`:

```typescript
import { NextRequest, NextResponse } from "next/server";

const issuer = process.env.RAMPART_ISSUER || "http://localhost:8080";
const clientId = process.env.RAMPART_CLIENT_ID || "my-nextjs-app";

export async function GET(req: NextRequest) {
  const code = req.nextUrl.searchParams.get("code");
  if (!code) {
    return NextResponse.redirect(new URL("/", req.url));
  }

  const tokenRes = await fetch(`${issuer}/oauth/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: new URL("/api/auth/callback", req.url).toString(),
      client_id: clientId,
    }),
  });

  if (!tokenRes.ok) {
    return NextResponse.redirect(new URL("/?error=auth_failed", req.url));
  }

  const tokens = await tokenRes.json();
  const response = NextResponse.redirect(new URL("/dashboard", req.url));

  response.cookies.set("rampart_token", tokens.access_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: tokens.expires_in,
    path: "/",
  });

  return response;
}
```

### 7. Create logout route

Create `app/api/auth/logout/route.ts`:

```typescript
import { NextResponse } from "next/server";

export async function POST(req: Request) {
  const response = NextResponse.redirect(new URL("/", req.url));
  response.cookies.delete("rampart_token");
  return response;
}
```

### 8. Add environment variables

Add to `.env.local`:

```
RAMPART_ISSUER=http://localhost:8080
RAMPART_CLIENT_ID=my-nextjs-app
```

### 9. Register the client in Rampart

```
# Rampart Admin > Clients > Create Client
# - Name: My Next.js App
# - Client ID: my-nextjs-app
# - Type: Confidential (if using server-side) or Public
# - Redirect URI: http://localhost:3000/api/auth/callback
```

## Checklist

- [ ] `jose` installed
- [ ] `lib/auth.ts` and `lib/with-auth.ts` created
- [ ] API routes use `withAuth()` wrapper
- [ ] `middleware.ts` protects pages
- [ ] OAuth callback route handles token exchange
- [ ] Environment variables configured
- [ ] OAuth client registered in Rampart
