---
sidebar_position: 4
title: Next.js
description: Integrate Rampart authentication into Next.js applications with server-side verification, middleware, and client-side auth context.
---

# Next.js Adapter

The `@rampart-auth/nextjs` adapter provides full-stack authentication for Next.js applications using the App Router. It covers server-side token verification in Server Components, edge middleware for route protection, and a client-side auth context for interactive pages.

## Installation

```bash
npm install @rampart-auth/nextjs
```

```bash
yarn add @rampart-auth/nextjs
```

```bash
pnpm add @rampart-auth/nextjs
```

## Quick Start

### 1. Configure the Provider

Create a Rampart configuration file:

```typescript
// lib/rampart.ts
import { createRampartAuth } from "@rampart-auth/nextjs";

export const rampart = createRampartAuth({
  issuerUrl: process.env.RAMPART_URL!,
  clientId: process.env.RAMPART_CLIENT_ID!,
  clientSecret: process.env.RAMPART_CLIENT_SECRET!,
  redirectUri: process.env.RAMPART_REDIRECT_URI!,
  postLogoutRedirectUri: process.env.NEXT_PUBLIC_APP_URL!,
  scopes: ["openid", "profile", "email"],
});
```

### 2. Add the Auth API Route

```typescript
// app/api/auth/[...rampart]/route.ts
import { rampart } from "@/lib/rampart";

export const { GET, POST } = rampart.handlers();
```

This creates the following routes automatically:

| Route | Purpose |
|-------|---------|
| `/api/auth/login` | Initiates the Authorization Code flow |
| `/api/auth/callback` | Handles the OAuth redirect |
| `/api/auth/logout` | Clears the session and redirects to Rampart logout |
| `/api/auth/session` | Returns the current session (for client-side) |
| `/api/auth/refresh` | Refreshes the access token |

### 3. Add Middleware

```typescript
// middleware.ts
import { rampart } from "@/lib/rampart";

export const middleware = rampart.middleware({
  publicPaths: ["/", "/about", "/api/health"],
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
```

### 4. Wrap the Layout with AuthProvider

```tsx
// app/layout.tsx
import { RampartProvider } from "@rampart-auth/nextjs/client";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <RampartProvider>{children}</RampartProvider>
      </body>
    </html>
  );
}
```

## Environment Variables

Add these to your `.env.local`:

```bash
RAMPART_URL=https://auth.example.com
RAMPART_CLIENT_ID=my-nextjs-app
RAMPART_CLIENT_SECRET=your-client-secret
RAMPART_REDIRECT_URI=http://localhost:3000/api/auth/callback
NEXT_PUBLIC_APP_URL=http://localhost:3000
```

## Server-Side Authentication

### Server Components

Use `getSession()` in Server Components to access the authenticated user:

```tsx
// app/dashboard/page.tsx
import { rampart } from "@/lib/rampart";

export default async function DashboardPage() {
  const session = await rampart.getSession();

  if (!session) {
    redirect("/api/auth/login");
  }

  return (
    <div>
      <h1>Dashboard</h1>
      <p>Welcome, {session.user.name}</p>
      <p>Email: {session.user.email}</p>
      <p>Roles: {session.user.roles.join(", ")}</p>
    </div>
  );
}
```

### Server Actions

Verify authentication in Server Actions:

```typescript
// app/actions.ts
"use server";

import { rampart } from "@/lib/rampart";

export async function createTask(formData: FormData) {
  const session = await rampart.getSession();
  if (!session) {
    throw new Error("Unauthorized");
  }

  const title = formData.get("title") as string;

  // Use the access token to call your API
  const res = await fetch(`${process.env.API_URL}/tasks`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ title, userId: session.user.sub }),
  });

  if (!res.ok) {
    throw new Error("Failed to create task");
  }

  return res.json();
}
```

### Route Handlers

Protect API routes:

```typescript
// app/api/tasks/route.ts
import { rampart } from "@/lib/rampart";
import { NextResponse } from "next/server";

export async function GET() {
  const session = await rampart.getSession();
  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  // Fetch tasks for the authenticated user
  return NextResponse.json({
    tasks: [
      { id: "1", title: "Review PR", assignee: session.user.sub },
    ],
  });
}
```

## Middleware

The middleware intercepts requests and redirects unauthenticated users to the login page.

```typescript
// middleware.ts
import { rampart } from "@/lib/rampart";

export const middleware = rampart.middleware({
  // Routes that don't require authentication
  publicPaths: [
    "/",
    "/about",
    "/pricing",
    "/api/health",
    "/api/webhooks/(.*)",
  ],

  // Routes that require specific roles
  rolePaths: {
    "/admin/(.*)": ["admin"],
    "/billing/(.*)": ["admin", "billing"],
  },

  // Where to redirect unauthenticated users (default: /api/auth/login)
  loginPath: "/api/auth/login",

  // Where to redirect users who lack required roles
  forbiddenPath: "/403",
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
```

## Client-Side Authentication

### `useAuth()` Hook

Access authentication state in Client Components:

```tsx
"use client";

import { useAuth } from "@rampart-auth/nextjs/client";

export function UserMenu() {
  const { user, isAuthenticated, isLoading } = useAuth();

  if (isLoading) return <div>Loading...</div>;

  if (!isAuthenticated) {
    return <a href="/api/auth/login">Log in</a>;
  }

  return (
    <div>
      <span>{user.name}</span>
      <a href="/api/auth/logout">Log out</a>
    </div>
  );
}
```

### `useAccessToken()` Hook

Get a fresh access token for API calls from the client:

```tsx
"use client";

import { useAccessToken } from "@rampart-auth/nextjs/client";
import { useState, useEffect } from "react";

export function TaskList() {
  const getToken = useAccessToken();
  const [tasks, setTasks] = useState([]);

  useEffect(() => {
    async function load() {
      const token = await getToken();
      const res = await fetch("/api/tasks", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) setTasks(await res.json());
    }
    load();
  }, [getToken]);

  return (
    <ul>
      {tasks.map((t: any) => (
        <li key={t.id}>{t.title}</li>
      ))}
    </ul>
  );
}
```

## Session Object

The session object returned by `getSession()` and the `useAuth()` hook:

```typescript
interface RampartSession {
  user: {
    sub: string;           // User ID
    email?: string;        // Email address
    name?: string;         // Display name
    roles: string[];       // Assigned roles
    orgId?: string;        // Organization ID
  };
  accessToken: string;     // Current access token
  refreshToken: string;    // Refresh token
  idToken: string;         // ID token
  expiresAt: number;       // Access token expiration timestamp
}
```

## Full Working Example

```typescript
// lib/rampart.ts
import { createRampartAuth } from "@rampart-auth/nextjs";

export const rampart = createRampartAuth({
  issuerUrl: process.env.RAMPART_URL!,
  clientId: process.env.RAMPART_CLIENT_ID!,
  clientSecret: process.env.RAMPART_CLIENT_SECRET!,
  redirectUri: process.env.RAMPART_REDIRECT_URI!,
  postLogoutRedirectUri: process.env.NEXT_PUBLIC_APP_URL!,
  scopes: ["openid", "profile", "email"],
});
```

```typescript
// app/api/auth/[...rampart]/route.ts
import { rampart } from "@/lib/rampart";
export const { GET, POST } = rampart.handlers();
```

```typescript
// middleware.ts
import { rampart } from "@/lib/rampart";

export const middleware = rampart.middleware({
  publicPaths: ["/", "/about"],
  rolePaths: { "/admin/(.*)": ["admin"] },
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
```

```tsx
// app/layout.tsx
import { RampartProvider } from "@rampart-auth/nextjs/client";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <RampartProvider>{children}</RampartProvider>
      </body>
    </html>
  );
}
```

```tsx
// app/page.tsx
import { rampart } from "@/lib/rampart";
import { redirect } from "next/navigation";
import { LogoutButton } from "@/components/LogoutButton";

export default async function HomePage() {
  const session = await rampart.getSession();

  if (!session) {
    return (
      <div>
        <h1>Welcome to My App</h1>
        <a href="/api/auth/login">Log in</a>
      </div>
    );
  }

  return (
    <div>
      <h1>Welcome, {session.user.name}</h1>
      <p>You are logged in as {session.user.email}</p>
      <nav>
        <a href="/dashboard">Dashboard</a>
        {session.user.roles.includes("admin") && <a href="/admin">Admin</a>}
      </nav>
      <LogoutButton />
    </div>
  );
}
```

```tsx
// components/LogoutButton.tsx
"use client";

export function LogoutButton() {
  return (
    <a href="/api/auth/logout">
      <button>Log out</button>
    </a>
  );
}
```

```tsx
// app/dashboard/page.tsx
import { rampart } from "@/lib/rampart";
import { redirect } from "next/navigation";

export default async function DashboardPage() {
  const session = await rampart.getSession();
  if (!session) redirect("/api/auth/login");

  const res = await fetch(`${process.env.API_URL}/tasks`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
    cache: "no-store",
  });

  const { tasks } = await res.json();

  return (
    <div>
      <h1>Dashboard</h1>
      <h2>Your Tasks</h2>
      <ul>
        {tasks.map((t: { id: string; title: string }) => (
          <li key={t.id}>{t.title}</li>
        ))}
      </ul>
    </div>
  );
}
```

## Security Considerations

- **Store `RAMPART_CLIENT_SECRET` only in server-side environment variables.** Never prefix it with `NEXT_PUBLIC_`.
- **Use HTTP-only cookies for session storage** (the adapter handles this automatically).
- **Set `SameSite=Lax`** on session cookies to prevent CSRF.
- **Enable HTTPS in production** — session cookies require secure transport.
- The middleware runs at the edge, so route protection happens before your application code executes.
