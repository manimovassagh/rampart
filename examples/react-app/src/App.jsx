import React, { createContext, useContext, useCallback, useEffect, useState } from "react";
import { Routes, Route, Link, useNavigate, Navigate, useSearchParams } from "react-router-dom";
import {
  login,
  logout as rawLogout,
  handleCallback,
  getAccessToken,
  getUserInfo,
  isAuthenticated,
  forgotPassword,
  resetPassword,
  directLogin,
  verifyMFA,
  socialLogin,
} from "./rampart";

const RAMPART_URL = "http://localhost:8080";

// --- Auth Context ---

const AuthContext = createContext();

function AuthProvider({ children }) {
  const [authed, setAuthed] = useState(isAuthenticated());
  const refreshAuth = useCallback(() => setAuthed(isAuthenticated()), []);
  const logout = useCallback(() => {
    rawLogout();
    setAuthed(false);
  }, []);

  return (
    <AuthContext.Provider value={{ authed, refreshAuth, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

function useAuth() {
  return useContext(AuthContext);
}

// --- Social Icons ---

function GoogleIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
    </svg>
  );
}

function GitHubIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/>
    </svg>
  );
}

function AppleIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
      <path d="M17.05 20.28c-.98.95-2.05.88-3.08.4-1.09-.5-2.08-.48-3.24 0-1.44.62-2.2.44-3.06-.4C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.54 4.09zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.29 2.58-2.34 4.5-3.74 4.25z"/>
    </svg>
  );
}

// --- Navigation ---

function Nav() {
  const { authed, logout } = useAuth();
  const user = getUserInfo();
  const roles = user?.roles || [];

  return (
    <nav className="nav">
      <div className="nav-brand">Rampart Demo</div>
      <div className="nav-links">
        <Link to="/">Home</Link>
        {authed && <Link to="/dashboard">Dashboard</Link>}
        {authed && <Link to="/api-test">API</Link>}
        {authed && roles.includes("admin") && <Link to="/admin">Admin</Link>}
      </div>
      <div className="nav-right">
        {authed && roles.length > 0 && roles.map((role) => (
          <span key={role} className={`badge badge-${role}`}>{role}</span>
        ))}
        {authed ? (
          <button className="btn btn-secondary btn-sm" onClick={() => { logout(); window.location.href = "/"; }}>
            Logout
          </button>
        ) : (
          <button className="btn btn-primary btn-sm" onClick={login}>
            Login
          </button>
        )}
      </div>
    </nav>
  );
}

// --- Protected Route ---

function ProtectedRoute({ children, requiredRole }) {
  const { authed } = useAuth();
  if (!authed) return <Navigate to="/" replace />;
  if (requiredRole) {
    const user = getUserInfo();
    const roles = user?.roles || [];
    if (!roles.includes(requiredRole)) {
      return (
        <div className="container">
          <div className="card card-error">
            <h2>Access Denied</h2>
            <p style={{ color: "var(--text-secondary)", marginBottom: "1rem" }}>
              You need the <strong>{requiredRole}</strong> role to view this page.
            </p>
            <Link to="/" className="btn btn-secondary btn-sm">Back to Home</Link>
          </div>
        </div>
      );
    }
  }
  return children;
}

// --- Home Page ---

function HomePage() {
  const { authed } = useAuth();
  const user = getUserInfo();

  return (
    <div className="container">
      <div className="hero">
        <h1>Rampart React Demo</h1>
        <p>A sample React SPA secured with Rampart IAM</p>
      </div>

      {authed && user ? (
        <div className="card card-elevated">
          <h2>Welcome back, {user.given_name || user.preferred_username}</h2>
          <table className="info-table">
            <tbody>
              {user.email && (
                <tr><td className="label">Email</td><td>{user.email}</td></tr>
              )}
              {user.preferred_username && (
                <tr><td className="label">Username</td><td>{user.preferred_username}</td></tr>
              )}
              {user.roles && (
                <tr>
                  <td className="label">Roles</td>
                  <td>{user.roles.map((r) => (
                    <span key={r} className={`badge badge-${r}`}>{r}</span>
                  ))}</td>
                </tr>
              )}
              {user.exp && (
                <tr><td className="label">Session expires</td><td>{new Date(user.exp * 1000).toLocaleString()}</td></tr>
              )}
            </tbody>
          </table>
          <div className="btn-group" style={{ marginTop: "1rem" }}>
            <Link to="/dashboard" className="btn btn-primary btn-sm">View Dashboard</Link>
            <Link to="/api-test" className="btn btn-outline btn-sm">Test API Calls</Link>
          </div>
        </div>
      ) : (
        <div className="card card-elevated" style={{ textAlign: "center", padding: "2rem" }}>
          <h2 style={{ marginBottom: "0.5rem" }}>Get started</h2>
          <p style={{ color: "var(--text-secondary)", marginBottom: "1.5rem" }}>
            Sign in with OAuth 2.0 PKCE, social providers, or username & password.
          </p>
          <div className="btn-group" style={{ justifyContent: "center" }}>
            <button className="btn btn-primary" onClick={login}>Sign in with Rampart</button>
            <Link to="/login" className="btn btn-secondary">Email & Password</Link>
          </div>
        </div>
      )}
    </div>
  );
}

// --- Sign In Page (was "Direct Login") ---

function SignInPage() {
  const navigate = useNavigate();
  const { refreshAuth } = useAuth();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [mfaState, setMfaState] = useState(null);
  const [mfaCode, setMfaCode] = useState("");

  const handleLogin = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const result = await directLogin(identifier, password);
      if (result.mfaRequired) {
        setMfaState(result.mfaToken);
        setLoading(false);
      } else {
        refreshAuth();
        navigate("/", { replace: true });
      }
    } catch (err) {
      setError(err.message);
      setLoading(false);
    }
  };

  const handleMFA = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await verifyMFA(mfaState, mfaCode);
      refreshAuth();
      navigate("/", { replace: true });
    } catch (err) {
      setError(err.message);
      setLoading(false);
    }
  };

  if (mfaState) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div className="logo-area">
            <h1>Two-Factor Auth</h1>
            <p>Enter the 6-digit code from your authenticator app</p>
          </div>
          {error && <div className="error-inline">{error}</div>}
          <form onSubmit={handleMFA}>
            <div className="form-group">
              <label>Verification Code</label>
              <input
                type="text"
                value={mfaCode}
                onChange={(e) => setMfaCode(e.target.value)}
                placeholder="000000"
                autoComplete="one-time-code"
                autoFocus
                style={{ textAlign: "center", letterSpacing: "0.5em", fontSize: "1.25rem" }}
              />
            </div>
            <button className="btn btn-primary btn-full" disabled={loading}>
              {loading ? "Verifying..." : "Verify"}
            </button>
          </form>
          <p className="hint" style={{ marginTop: "1rem", textAlign: "center" }}>
            You can also use a backup code
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="logo-area">
          <h1>Sign In</h1>
          <p>Sign in to continue to the app</p>
        </div>

        {error && <div className="error-inline">{error}</div>}

        <form onSubmit={handleLogin}>
          <div className="form-group">
            <label>Username or Email</label>
            <input
              type="text"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              placeholder="you@example.com"
              autoFocus
            />
          </div>
          <div className="form-group">
            <label>Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <Link to="/forgot-password" className="forgot-link">Forgot password?</Link>
          </div>
          <button className="btn btn-primary btn-full" disabled={loading}>
            {loading ? "Signing in..." : "Sign In"}
          </button>
        </form>

        <div className="divider"><span>or continue with</span></div>

        <div className="social-buttons">
          <button className="social-btn social-btn-google" onClick={() => socialLogin("google")}>
            <GoogleIcon /> Sign in with Google
          </button>
          <button className="social-btn social-btn-github" onClick={() => socialLogin("github")}>
            <GitHubIcon /> Sign in with GitHub
          </button>
          <button className="social-btn social-btn-apple" onClick={() => socialLogin("apple")}>
            <AppleIcon /> Sign in with Apple
          </button>
        </div>

        <p className="hint" style={{ marginTop: "1.5rem", textAlign: "center" }}>
          <Link to="/">Back to Home</Link>
        </p>
      </div>
    </div>
  );
}

// --- Forgot Password ---

function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await forgotPassword(email);
      setSubmitted(true);
    } catch (err) {
      setError(err.message);
    }
    setLoading(false);
  };

  if (submitted) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div className="logo-area">
            <h1>Check your email</h1>
            <p>If an account with <strong>{email}</strong> exists, we sent a password reset link.</p>
          </div>
          <div className="success-inline">Reset link sent. Check your inbox and spam folder.</div>
          <div className="btn-group" style={{ flexDirection: "column" }}>
            <button className="btn btn-secondary btn-full" onClick={() => setSubmitted(false)}>Try a different email</button>
            <Link to="/reset-password" className="btn btn-outline btn-full">I have a reset token</Link>
            <Link to="/login" className="btn btn-primary btn-full">Back to Sign In</Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="logo-area">
          <h1>Forgot Password</h1>
          <p>Enter your email and we'll send a reset link</p>
        </div>
        {error && <div className="error-inline">{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Email address</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              required
              autoFocus
            />
          </div>
          <button className="btn btn-primary btn-full" disabled={loading}>
            {loading ? "Sending..." : "Send Reset Link"}
          </button>
        </form>
        <p className="hint" style={{ marginTop: "1rem", textAlign: "center" }}>
          <Link to="/login">Back to Sign In</Link>
        </p>
      </div>
    </div>
  );
}

// --- Reset Password ---

function ResetPasswordPage() {
  const [searchParams] = useSearchParams();
  const [token, setToken] = useState(searchParams.get("token") || "");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    if (newPassword !== confirmPassword) { setError("Passwords do not match."); return; }
    if (newPassword.length < 8) { setError("Password must be at least 8 characters."); return; }
    setLoading(true);
    try {
      await resetPassword(token, newPassword);
      setSuccess(true);
    } catch (err) {
      setError(err.message);
    }
    setLoading(false);
  };

  if (success) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div className="logo-area">
            <h1>Password Reset</h1>
            <p>Your password has been updated successfully.</p>
          </div>
          <div className="success-inline">You can now sign in with your new password.</div>
          <Link to="/login" className="btn btn-primary btn-full">Go to Sign In</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="logo-area">
          <h1>Reset Password</h1>
          <p>Enter your reset token and new password</p>
        </div>
        {error && <div className="error-inline">{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Reset Token</label>
            <input type="text" value={token} onChange={(e) => setToken(e.target.value)} placeholder="Paste token from email" required />
          </div>
          <div className="form-group">
            <label>New Password</label>
            <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} placeholder="Min 8 characters" required />
          </div>
          <div className="form-group">
            <label>Confirm Password</label>
            <input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} required />
          </div>
          <button className="btn btn-primary btn-full" disabled={loading}>
            {loading ? "Resetting..." : "Reset Password"}
          </button>
        </form>
        <p className="hint" style={{ marginTop: "1rem", textAlign: "center" }}>
          <Link to="/login">Back to Sign In</Link>
        </p>
      </div>
    </div>
  );
}

// --- OAuth Callback ---

function CallbackPage() {
  const navigate = useNavigate();
  const { refreshAuth } = useAuth();
  const [error, setError] = useState(null);

  useEffect(() => {
    handleCallback()
      .then(() => {
        refreshAuth();
        navigate("/", { replace: true });
      })
      .catch((err) => setError(err.message));
  }, [navigate, refreshAuth]);

  if (error) {
    return (
      <div className="container">
        <div className="card card-error">
          <h2>Authentication Failed</h2>
          <p style={{ color: "var(--text-secondary)" }}>{error}</p>
          <Link to="/" className="btn btn-secondary btn-sm" style={{ marginTop: "1rem" }}>Back to Home</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-page">
      <div className="auth-card" style={{ textAlign: "center" }}>
        <h1 style={{ fontSize: "1.25rem", marginBottom: "0.5rem" }}>Completing login...</h1>
        <p style={{ color: "var(--text-secondary)" }}>Exchanging authorization code for tokens.</p>
      </div>
    </div>
  );
}

// --- Dashboard ---

function DashboardPage() {
  const user = getUserInfo();
  const token = getAccessToken();

  return (
    <div className="container">
      <div className="hero">
        <h1>Dashboard</h1>
        <p>Your profile and session details</p>
      </div>

      {user && (
        <>
          <div className="card">
            <h2>Profile</h2>
            <table className="info-table">
              <tbody>
                <tr><td className="label">User ID</td><td><code>{user.sub}</code></td></tr>
                <tr><td className="label">Username</td><td>{user.preferred_username}</td></tr>
                <tr><td className="label">Email</td><td>{user.email} {user.email_verified ? " (verified)" : ""}</td></tr>
                <tr><td className="label">Name</td><td>{user.given_name} {user.family_name}</td></tr>
                <tr><td className="label">Organization</td><td><code>{user.org_id}</code></td></tr>
                <tr>
                  <td className="label">Roles</td>
                  <td>{(user.roles || []).map((r) => (
                    <span key={r} className={`badge badge-${r}`} style={{ marginRight: "0.25rem" }}>{r}</span>
                  ))}</td>
                </tr>
                <tr><td className="label">Expires</td><td>{new Date(user.exp * 1000).toLocaleString()}</td></tr>
              </tbody>
            </table>
          </div>

          <div className="card">
            <h2>JWT Claims</h2>
            <pre className="json-block">{JSON.stringify(user, null, 2)}</pre>
          </div>

          <div className="card">
            <h2>Access Token</h2>
            <pre className="json-block" style={{ wordBreak: "break-all", whiteSpace: "pre-wrap" }}>{token}</pre>
          </div>
        </>
      )}
    </div>
  );
}

// --- API Test Page (role-based backend calls) ---

function APITestPage() {
  const token = getAccessToken();
  const user = getUserInfo();
  const roles = user?.roles || [];
  const [results, setResults] = useState({});

  const callAPI = async (key, method, path) => {
    setResults((prev) => ({ ...prev, [key]: { loading: true } }));
    try {
      const res = await fetch(`${RAMPART_URL}${path}`, {
        method,
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
      });
      const body = await res.text();
      let data;
      try { data = JSON.parse(body); } catch { data = body; }
      setResults((prev) => ({ ...prev, [key]: { status: res.status, ok: res.ok, data } }));
    } catch (err) {
      setResults((prev) => ({ ...prev, [key]: { status: 0, ok: false, data: err.message } }));
    }
  };

  const endpoints = [
    { key: "me", label: "Get My Profile", method: "GET", path: "/me", roles: [] },
    { key: "health", label: "Health Check", method: "GET", path: "/healthz", roles: [] },
    { key: "ready", label: "Readiness", method: "GET", path: "/readyz", roles: [] },
    { key: "discovery", label: "OIDC Discovery", method: "GET", path: "/.well-known/openid-configuration", roles: [] },
    { key: "jwks", label: "JWKS", method: "GET", path: "/.well-known/jwks.json", roles: [] },
    { key: "userinfo", label: "UserInfo (OIDC)", method: "GET", path: "/oauth/userinfo", roles: [] },
  ];

  return (
    <div className="container">
      <div className="hero">
        <h1>API Explorer</h1>
        <p>Test Rampart API endpoints with your access token</p>
      </div>

      {endpoints.map((ep) => {
        const result = results[ep.key];
        const needsRole = ep.roles.length > 0 && !ep.roles.some((r) => roles.includes(r));
        return (
          <div className="card" key={ep.key}>
            <div className="api-panel">
              <div className="endpoint">
                <span className="method">{ep.method}</span>
                <span className="url">{ep.path}</span>
                {result && !result.loading && (
                  <span className={`status ${result.ok ? "status-ok" : "status-err"}`}>
                    {result.status}
                  </span>
                )}
                {result?.loading && <span className="status status-loading">loading</span>}
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                <button
                  className={`btn btn-sm ${needsRole ? "btn-danger" : "btn-primary"}`}
                  onClick={() => callAPI(ep.key, ep.method, ep.path)}
                  disabled={result?.loading}
                >
                  {ep.label}
                </button>
                {needsRole && (
                  <span className="hint">Requires {ep.roles.join(", ")} role</span>
                )}
              </div>
              {result && !result.loading && (
                <pre className="json-block" style={{ marginTop: "0.75rem" }}>
                  {typeof result.data === "string" ? result.data : JSON.stringify(result.data, null, 2)}
                </pre>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// --- Admin Page ---

function AdminPage() {
  const user = getUserInfo();

  return (
    <div className="container">
      <div className="hero">
        <h1>Admin Panel</h1>
        <p>Restricted to users with the <strong>admin</strong> role</p>
      </div>

      <div className="card">
        <h2>Admin Access</h2>
        <p style={{ color: "var(--text-secondary)", marginBottom: "1rem" }}>
          Signed in as <strong>{user?.preferred_username}</strong> with admin privileges.
        </p>
        <a
          href={`${RAMPART_URL}/admin/`}
          target="_blank"
          rel="noreferrer"
          className="btn btn-primary btn-sm"
        >
          Open Rampart Admin Console
        </a>
      </div>
    </div>
  );
}

// --- App ---

export default function App() {
  return (
    <AuthProvider>
      <Nav />
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/login" element={<SignInPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route path="/callback" element={<CallbackPage />} />
        <Route
          path="/dashboard"
          element={<ProtectedRoute><DashboardPage /></ProtectedRoute>}
        />
        <Route
          path="/api-test"
          element={<ProtectedRoute><APITestPage /></ProtectedRoute>}
        />
        <Route
          path="/admin"
          element={<ProtectedRoute requiredRole="admin"><AdminPage /></ProtectedRoute>}
        />
      </Routes>
    </AuthProvider>
  );
}
