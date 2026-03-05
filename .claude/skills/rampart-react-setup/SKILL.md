---
name: rampart-react-setup
description: Add Rampart authentication to a React (Vite/CRA) app. Sets up OAuth 2.0 PKCE login flow, auth context, protected routes, and token management. Use when securing a React SPA with Rampart.
argument-hint: [issuer-url]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Add Rampart Authentication to a React App

Set up OAuth 2.0 Authorization Code + PKCE login flow in a React SPA using `@rampart/react`.

## What This Skill Does

1. Installs `@rampart/react` (or sets up a lightweight auth module)
2. Creates an `AuthProvider` context with login/logout/token state
3. Adds a `useAuth()` hook for accessing user info and tokens
4. Creates a `ProtectedRoute` component for guarding routes
5. Handles the OAuth callback and token refresh

## Step-by-Step

### 1. Install dependencies

```bash
npm install @rampart/react
# or if @rampart/react is not published yet:
npm install jose
```

### 2. Find the app entry point

Look for `App.tsx`, `main.tsx`, or `src/App.tsx`.

### 3. Create the auth configuration

Create `src/auth/config.ts`:

```typescript
export const authConfig = {
  issuer: "$ARGUMENTS" || process.env.VITE_RAMPART_ISSUER || "http://localhost:8080",
  clientId: process.env.VITE_RAMPART_CLIENT_ID || "my-app",
  redirectUri: window.location.origin + "/callback",
  scopes: ["openid", "profile", "email"],
};
```

### 4. Create the auth context

Create `src/auth/AuthProvider.tsx`:

```tsx
import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from "react";
import { authConfig } from "./config";

interface User {
  sub: string;
  email: string;
  preferred_username: string;
  org_id: string;
  given_name?: string;
  family_name?: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: () => void;
  logout: () => void;
  getAccessToken: () => string | null;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be inside AuthProvider");
  return ctx;
}

function generateCodeVerifier(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

async function generateCodeChallenge(verifier: string): Promise<string> {
  const data = new TextEncoder().encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const getAccessToken = useCallback(() => localStorage.getItem("rampart_access_token"), []);

  const login = useCallback(async () => {
    const verifier = generateCodeVerifier();
    const challenge = await generateCodeChallenge(verifier);
    sessionStorage.setItem("pkce_verifier", verifier);

    const params = new URLSearchParams({
      response_type: "code",
      client_id: authConfig.clientId,
      redirect_uri: authConfig.redirectUri,
      scope: authConfig.scopes.join(" "),
      code_challenge: challenge,
      code_challenge_method: "S256",
    });

    window.location.href = `${authConfig.issuer}/oauth/authorize?${params}`;
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem("rampart_access_token");
    localStorage.removeItem("rampart_refresh_token");
    setUser(null);
    window.location.href = "/";
  }, []);

  const fetchUser = useCallback(async (token: string) => {
    const res = await fetch(`${authConfig.issuer}/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      setUser(await res.json());
    } else {
      localStorage.removeItem("rampart_access_token");
      setUser(null);
    }
  }, []);

  useEffect(() => {
    const token = localStorage.getItem("rampart_access_token");
    if (token) {
      fetchUser(token).finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [fetchUser]);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, isLoading, login, logout, getAccessToken }}>
      {children}
    </AuthContext.Provider>
  );
}
```

### 5. Create the callback handler

Create `src/auth/Callback.tsx`:

```tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { authConfig } from "./config";

export function AuthCallback() {
  const navigate = useNavigate();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get("code");
    const verifier = sessionStorage.getItem("pkce_verifier");

    if (!code || !verifier) {
      navigate("/");
      return;
    }

    fetch(`${authConfig.issuer}/oauth/token`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "authorization_code",
        code,
        redirect_uri: authConfig.redirectUri,
        client_id: authConfig.clientId,
        code_verifier: verifier,
      }),
    })
      .then((res) => res.json())
      .then((data) => {
        localStorage.setItem("rampart_access_token", data.access_token);
        if (data.refresh_token) localStorage.setItem("rampart_refresh_token", data.refresh_token);
        sessionStorage.removeItem("pkce_verifier");
        navigate("/");
      })
      .catch(() => navigate("/"));
  }, [navigate]);

  return <div>Signing you in...</div>;
}
```

### 6. Create ProtectedRoute

Create `src/auth/ProtectedRoute.tsx`:

```tsx
import { useAuth } from "./AuthProvider";

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, login } = useAuth();

  if (isLoading) return <div>Loading...</div>;
  if (!isAuthenticated) {
    login();
    return <div>Redirecting to login...</div>;
  }
  return <>{children}</>;
}
```

### 7. Wire it up in the app

```tsx
import { AuthProvider } from "./auth/AuthProvider";
import { AuthCallback } from "./auth/Callback";
import { ProtectedRoute } from "./auth/ProtectedRoute";

function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/callback" element={<AuthCallback />} />
        <Route path="/dashboard" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
        <Route path="/" element={<Home />} />
      </Routes>
    </AuthProvider>
  );
}
```

### 8. Register the OAuth client in Rampart

Either via the admin dashboard or CLI:

```bash
# Via the Rampart admin console: Clients > Create Client
# - Name: My React App
# - Client ID: my-app
# - Type: Public
# - Redirect URI: http://localhost:5173/callback (Vite default)
```

### 9. Add environment variables

Create or update `.env`:

```
VITE_RAMPART_ISSUER=http://localhost:8080
VITE_RAMPART_CLIENT_ID=my-app
```

## Checklist

- [ ] Auth dependencies installed
- [ ] `AuthProvider` wraps the app
- [ ] `/callback` route handles OAuth redirect
- [ ] Protected routes use `<ProtectedRoute>`
- [ ] OAuth client registered in Rampart with correct redirect URI
- [ ] Environment variables set for issuer and client ID
