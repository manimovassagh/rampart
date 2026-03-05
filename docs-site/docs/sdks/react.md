---
sidebar_position: 3
title: React SPA
description: Integrate Rampart authentication into React single-page applications using the Authorization Code flow with PKCE.
---

# React SPA Adapter

The `@rampart/react` adapter provides React components and hooks for integrating Rampart authentication into single-page applications. It implements the Authorization Code flow with PKCE — the recommended approach for public clients that cannot securely store a client secret.

## Installation

```bash
npm install @rampart/react
```

```bash
yarn add @rampart/react
```

```bash
pnpm add @rampart/react
```

## Quick Start

Wrap your application with `RampartProvider` and use the `useAuth` hook to access authentication state.

```tsx
import React from "react";
import ReactDOM from "react-dom/client";
import { RampartProvider } from "@rampart/react";
import App from "./App";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <RampartProvider
    issuerUrl="https://auth.example.com"
    clientId="my-spa"
    redirectUri="http://localhost:5173/callback"
    scopes={["openid", "profile", "email"]}
  >
    <App />
  </RampartProvider>
);
```

```tsx
// App.tsx
import { useAuth } from "@rampart/react";

function App() {
  const { user, isAuthenticated, isLoading, login, logout } = useAuth();

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (!isAuthenticated) {
    return <button onClick={login}>Log in</button>;
  }

  return (
    <div>
      <p>Welcome, {user.name}!</p>
      <button onClick={logout}>Log out</button>
    </div>
  );
}

export default App;
```

## Configuration

```tsx
<RampartProvider
  // Required
  issuerUrl="https://auth.example.com"
  clientId="my-spa"
  redirectUri="http://localhost:5173/callback"

  // Optional
  realm="default"                              // Organization/realm
  scopes={["openid", "profile", "email"]}      // Requested scopes
  postLogoutRedirectUri="http://localhost:5173" // Where to go after logout
  silentRefresh={true}                         // Auto-refresh tokens (default: true)
  silentRefreshInterval={60}                   // Refresh check interval in seconds
  storage="sessionStorage"                     // "localStorage" | "sessionStorage" (default)
  onError={(error) => console.error(error)}    // Global error handler
>
  <App />
</RampartProvider>
```

## Hooks

### `useAuth()`

The primary hook for accessing authentication state and actions.

```tsx
import { useAuth } from "@rampart/react";

function MyComponent() {
  const {
    // State
    isAuthenticated,    // boolean — is the user logged in?
    isLoading,          // boolean — is auth state being determined?
    user,               // User object or null
    accessToken,        // Current access token string or null
    error,              // Error object or null

    // Actions
    login,              // () => void — redirect to Rampart login
    logout,             // () => void — clear session and redirect to logout
    getAccessToken,     // () => Promise<string> — get a fresh access token
  } = useAuth();

  return <div>{isAuthenticated ? user.name : "Not logged in"}</div>;
}
```

### `useAccessToken()`

Returns a fresh access token, automatically refreshing if needed. Useful for making authenticated API calls.

```tsx
import { useAccessToken } from "@rampart/react";

function TaskList() {
  const getToken = useAccessToken();
  const [tasks, setTasks] = React.useState([]);

  React.useEffect(() => {
    async function fetchTasks() {
      const token = await getToken();
      const res = await fetch("/api/tasks", {
        headers: { Authorization: `Bearer ${token}` },
      });
      setTasks(await res.json());
    }
    fetchTasks();
  }, [getToken]);

  return (
    <ul>
      {tasks.map((task) => (
        <li key={task.id}>{task.title}</li>
      ))}
    </ul>
  );
}
```

### `useRoles()`

Check if the current user has specific roles.

```tsx
import { useRoles } from "@rampart/react";

function AdminPanel() {
  const { hasRole, hasAnyRole, roles } = useRoles();

  if (!hasRole("admin")) {
    return <p>Access denied. You need the admin role.</p>;
  }

  return (
    <div>
      <h2>Admin Panel</h2>
      <p>Your roles: {roles.join(", ")}</p>
      {hasAnyRole("super-admin", "owner") && (
        <button>Dangerous Action</button>
      )}
    </div>
  );
}
```

## Components

### `ProtectedRoute`

Wraps a route so that only authenticated users can access it. Unauthenticated users are redirected to the Rampart login page.

```tsx
import { ProtectedRoute } from "@rampart/react";
import { BrowserRouter, Routes, Route } from "react-router-dom";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route path="/" element={<Home />} />
        <Route path="/about" element={<About />} />

        {/* Protected routes */}
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />
        <Route
          path="/settings"
          element={
            <ProtectedRoute>
              <Settings />
            </ProtectedRoute>
          }
        />

        {/* Role-protected route */}
        <Route
          path="/admin"
          element={
            <ProtectedRoute roles={["admin"]} fallback={<AccessDenied />}>
              <AdminPanel />
            </ProtectedRoute>
          }
        />

        {/* Callback route — handles the OAuth redirect */}
        <Route path="/callback" element={<AuthCallback />} />
      </Routes>
    </BrowserRouter>
  );
}
```

### `AuthCallback`

Handles the OAuth 2.0 redirect callback. Place this at your `redirectUri` route.

```tsx
import { AuthCallback } from "@rampart/react";

// In your router:
<Route
  path="/callback"
  element={
    <AuthCallback
      onSuccess={() => navigate("/dashboard")}
      onError={(err) => navigate(`/error?message=${err.message}`)}
    >
      <p>Completing login...</p>
    </AuthCallback>
  }
/>
```

## PKCE Flow Details

The adapter implements the full Authorization Code flow with PKCE (RFC 7636):

1. **Login initiated** — generates a cryptographic `code_verifier` and derives a `code_challenge` using SHA-256
2. **Redirect to Rampart** — sends `code_challenge` and `code_challenge_method=S256` in the authorization request
3. **User authenticates** — at the Rampart login page
4. **Callback received** — Rampart redirects back with an authorization `code`
5. **Token exchange** — the adapter sends the `code` and `code_verifier` to the token endpoint
6. **Tokens stored** — access token, refresh token, and ID token are stored in the configured storage
7. **Silent refresh** — before the access token expires, the adapter uses the refresh token to obtain new tokens

No client secret is ever used or stored in the browser.

## Token Refresh

By default, the adapter automatically refreshes tokens before they expire. You can control this behavior:

```tsx
<RampartProvider
  silentRefresh={true}          // Enable auto-refresh (default: true)
  silentRefreshInterval={60}    // Check every 60 seconds
  onTokenRefreshError={(err) => {
    // Token refresh failed — user needs to log in again
    console.error("Refresh failed:", err);
  }}
>
```

To manually trigger a refresh:

```tsx
const { getAccessToken } = useAuth();

// This returns a fresh token, refreshing if needed
const token = await getAccessToken();
```

## Full Working Example

```tsx
// main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { RampartProvider } from "@rampart/react";
import App from "./App";

const RAMPART_URL = import.meta.env.VITE_RAMPART_URL || "https://auth.example.com";
const CLIENT_ID = import.meta.env.VITE_RAMPART_CLIENT_ID || "task-app";
const REDIRECT_URI = import.meta.env.VITE_REDIRECT_URI || "http://localhost:5173/callback";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <RampartProvider
        issuerUrl={RAMPART_URL}
        clientId={CLIENT_ID}
        redirectUri={REDIRECT_URI}
        scopes={["openid", "profile", "email", "tasks:read", "tasks:write"]}
        postLogoutRedirectUri="http://localhost:5173"
      >
        <App />
      </RampartProvider>
    </BrowserRouter>
  </React.StrictMode>
);
```

```tsx
// App.tsx
import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth, ProtectedRoute, AuthCallback } from "@rampart/react";

function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/callback" element={<AuthCallback onSuccess={() => {}} />} />
      <Route
        path="/tasks"
        element={
          <ProtectedRoute>
            <TaskList />
          </ProtectedRoute>
        }
      />
    </Routes>
  );
}

function Home() {
  const { isAuthenticated, login, user } = useAuth();

  return (
    <div>
      <h1>Task Manager</h1>
      {isAuthenticated ? (
        <div>
          <p>Welcome back, {user.name}!</p>
          <a href="/tasks">Go to tasks</a>
        </div>
      ) : (
        <button onClick={login}>Log in to get started</button>
      )}
    </div>
  );
}

function TaskList() {
  const { user, logout, getAccessToken } = useAuth();
  const [tasks, setTasks] = React.useState<{ id: string; title: string }[]>([]);
  const [newTask, setNewTask] = React.useState("");

  React.useEffect(() => {
    async function load() {
      const token = await getAccessToken();
      const res = await fetch("/api/tasks", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setTasks(data.tasks);
      }
    }
    load();
  }, [getAccessToken]);

  async function addTask(e: React.FormEvent) {
    e.preventDefault();
    const token = await getAccessToken();
    const res = await fetch("/api/tasks", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ title: newTask }),
    });
    if (res.ok) {
      const task = await res.json();
      setTasks((prev) => [...prev, task]);
      setNewTask("");
    }
  }

  return (
    <div>
      <header>
        <span>Logged in as {user.email}</span>
        <button onClick={logout}>Log out</button>
      </header>

      <h2>My Tasks</h2>
      <ul>
        {tasks.map((t) => (
          <li key={t.id}>{t.title}</li>
        ))}
      </ul>

      <form onSubmit={addTask}>
        <input
          value={newTask}
          onChange={(e) => setNewTask(e.target.value)}
          placeholder="New task..."
        />
        <button type="submit">Add</button>
      </form>
    </div>
  );
}

export default App;
```

### Environment Variables (Vite)

Create a `.env` file in your project root:

```bash
VITE_RAMPART_URL=https://auth.example.com
VITE_RAMPART_CLIENT_ID=task-app
VITE_REDIRECT_URI=http://localhost:5173/callback
```

## Security Considerations

- **Never store client secrets in frontend code.** The React adapter uses PKCE, which does not require a client secret.
- **Use `sessionStorage` (default) over `localStorage`** unless you need tokens to persist across tabs. `sessionStorage` is cleared when the tab is closed.
- **Set short access token lifetimes** (5-15 minutes) and rely on silent refresh for seamless UX.
- **Always use HTTPS in production.** Token transmission over HTTP is insecure.
- **Configure CORS on your API** to only accept requests from your SPA's origin.
