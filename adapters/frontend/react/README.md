# @rampart/react

React hooks and components for [Rampart](https://github.com/manimovassagh/rampart) authentication. Wraps `@rampart/web` to provide a provider/hook pattern for login, logout, token management, and route protection.

## Install

```bash
npm install @rampart/react @rampart/web
```

`react` (>=18) is a peer dependency.

## Quick Start

```tsx
import { RampartProvider } from "@rampart/react";
import App from "./App";

function Root() {
  return (
    <RampartProvider
      issuer="http://localhost:8080"
      clientId="my-app"
      redirectUri="http://localhost:3000/callback"
    >
      <App />
    </RampartProvider>
  );
}
```

```tsx
import { useAuth } from "@rampart/react";

function Dashboard() {
  const { user, isAuthenticated, logout } = useAuth();

  if (!isAuthenticated) return <p>Not logged in</p>;

  return (
    <div>
      <p>Hello, {user?.preferred_username}</p>
      <button onClick={logout}>Log out</button>
    </div>
  );
}
```

## Full Login + Callback Example

```tsx
import { useEffect } from "react";
import { useAuth } from "@rampart/react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";

function LoginPage() {
  const { loginWithRedirect } = useAuth();
  return <button onClick={loginWithRedirect}>Log in</button>;
}

function CallbackPage() {
  const { handleCallback, isAuthenticated } = useAuth();

  useEffect(() => {
    handleCallback();
  }, [handleCallback]);

  if (isAuthenticated) return <Navigate to="/" />;
  return <p>Processing login...</p>;
}

function Home() {
  const { user, authFetch, logout } = useAuth();

  async function loadData() {
    const res = await authFetch("/api/me");
    console.log(await res.json());
  }

  return (
    <div>
      <p>Welcome, {user?.email}</p>
      <button onClick={loadData}>Fetch profile</button>
      <button onClick={logout}>Log out</button>
    </div>
  );
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/callback" element={<CallbackPage />} />
        <Route path="/" element={<Home />} />
      </Routes>
    </BrowserRouter>
  );
}
```

## API

### `RampartProvider`

Context provider that initializes the Rampart client and manages authentication state. Wrap your application (or a subtree) with this component.

| Prop          | Type        | Default | Description                                                        |
|---------------|-------------|---------|--------------------------------------------------------------------|
| `issuer`      | `string`    | --      | Rampart server URL (e.g. `http://localhost:8080`)                  |
| `clientId`    | `string`    | --      | OAuth 2.0 client ID registered in Rampart                         |
| `redirectUri` | `string`    | --      | URL Rampart redirects to after login (must match client config)    |
| `scope`       | `string?`   | --      | OAuth scopes (e.g. `"openid profile email"`)                      |
| `persist`     | `boolean?`  | `true`  | Persist tokens to `localStorage` and restore on page reload        |
| `children`    | `ReactNode` | --      | Child components                                                   |

### `useAuth()`

Hook that returns authentication state and actions. Must be called within a `<RampartProvider>`.

| Property             | Type                                             | Description                                            |
|----------------------|--------------------------------------------------|--------------------------------------------------------|
| `user`               | `RampartUser \| null`                            | Current user, or `null` if not authenticated           |
| `isAuthenticated`    | `boolean`                                        | `true` when `user` is not `null`                       |
| `isLoading`          | `boolean`                                        | `true` while restoring tokens from storage on mount    |
| `loginWithRedirect`  | `() => Promise<void>`                            | Redirects the browser to the Rampart login page        |
| `handleCallback`     | `(url?: string) => Promise<void>`                | Exchanges the authorization code after redirect        |
| `logout`             | `() => Promise<void>`                            | Clears tokens and user state                           |
| `getAccessToken`     | `() => string \| null`                           | Returns the current access token, or `null`            |
| `authFetch`          | `(url: string, init?: RequestInit) => Promise<Response>` | `fetch` wrapper that attaches the Bearer token |

### `ProtectedRoute`

Component that conditionally renders children based on authentication and role checks.

| Prop              | Type          | Default | Description                                                 |
|-------------------|---------------|---------|-------------------------------------------------------------|
| `children`        | `ReactNode`   | --      | Content to render when authorized                           |
| `roles`           | `string[]?`   | --      | If set, user must have at least one of these roles          |
| `fallback`        | `ReactNode?`  | `null`  | Rendered when user is not authenticated or lacks roles      |
| `loadingFallback` | `ReactNode?`  | `null`  | Rendered while authentication state is loading              |

```tsx
import { ProtectedRoute } from "@rampart/react";

<ProtectedRoute roles={["admin"]} fallback={<p>Access denied</p>}>
  <AdminPanel />
</ProtectedRoute>
```

## TypeScript

The package ships with full type definitions. `RampartUser`, `RampartTokens`, and other `@rampart/web` types are re-exported from `@rampart/react` for convenience.

## License

MIT
