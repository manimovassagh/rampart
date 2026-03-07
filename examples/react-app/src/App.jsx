import React, { useEffect, useState } from "react";
import { Routes, Route, Link, useNavigate, Navigate } from "react-router-dom";
import {
  login,
  logout,
  handleCallback,
  getAccessToken,
  getUserInfo,
  isAuthenticated,
} from "./rampart";

// --- Navigation ---

function Nav() {
  const user = getUserInfo();
  const authed = isAuthenticated();
  const roles = user?.roles || [];

  return (
    <nav className="nav">
      <div className="nav-brand">Rampart Demo</div>
      <div className="nav-links">
        <Link to="/">Home</Link>
        {authed && <Link to="/dashboard">Dashboard</Link>}
        {authed && roles.includes("admin") && <Link to="/admin">Admin</Link>}
      </div>
      <div className="nav-right">
        {authed && roles.length > 0 && (
          <span className="role-badges">
            {roles.map((role) => (
              <span key={role} className={`badge badge-${role}`}>
                {role}
              </span>
            ))}
          </span>
        )}
        {authed ? (
          <button
            className="btn btn-secondary"
            onClick={() => {
              logout();
              window.location.href = "/";
            }}
          >
            Logout
          </button>
        ) : (
          <button className="btn btn-primary" onClick={login}>
            Login
          </button>
        )}
      </div>
    </nav>
  );
}

// --- Protected Route wrapper ---

function ProtectedRoute({ children, requiredRole }) {
  if (!isAuthenticated()) {
    return <Navigate to="/" replace />;
  }
  if (requiredRole) {
    const user = getUserInfo();
    const roles = user?.roles || [];
    if (!roles.includes(requiredRole)) {
      return (
        <div className="container">
          <div className="card error">
            <h2>Access Denied</h2>
            <p>You need the <strong>{requiredRole}</strong> role to view this page.</p>
            <Link to="/">Back to Home</Link>
          </div>
        </div>
      );
    }
  }
  return children;
}

// --- Pages ---

function HomePage() {
  const authed = isAuthenticated();
  const user = getUserInfo();

  return (
    <div className="container">
      <div className="hero">
        <h1>Rampart React Demo</h1>
        <p>A sample React SPA secured with Rampart IAM using OAuth 2.0 PKCE flow.</p>
      </div>

      {authed && user ? (
        <div className="card">
          <h2>Logged in</h2>
          <table className="info-table">
            <tbody>
              {user.sub && (
                <tr>
                  <td className="label">Subject</td>
                  <td>{user.sub}</td>
                </tr>
              )}
              {user.email && (
                <tr>
                  <td className="label">Email</td>
                  <td>{user.email}</td>
                </tr>
              )}
              {user.name && (
                <tr>
                  <td className="label">Name</td>
                  <td>{user.name}</td>
                </tr>
              )}
              {user.roles && (
                <tr>
                  <td className="label">Roles</td>
                  <td>
                    {user.roles.map((r) => (
                      <span key={r} className={`badge badge-${r}`}>
                        {r}
                      </span>
                    ))}
                  </td>
                </tr>
              )}
              {user.exp && (
                <tr>
                  <td className="label">Token Expires</td>
                  <td>{new Date(user.exp * 1000).toLocaleString()}</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="card">
          <h2>Not logged in</h2>
          <p>Click the Login button to authenticate with Rampart using the PKCE flow.</p>
          <button className="btn btn-primary" onClick={login}>
            Login with Rampart
          </button>
        </div>
      )}
    </div>
  );
}

function CallbackPage() {
  const navigate = useNavigate();
  const [error, setError] = useState(null);

  useEffect(() => {
    handleCallback()
      .then(() => navigate("/", { replace: true }))
      .catch((err) => setError(err.message));
  }, [navigate]);

  if (error) {
    return (
      <div className="container">
        <div className="card error">
          <h2>Authentication Failed</h2>
          <p>{error}</p>
          <Link to="/">Back to Home</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="container">
      <div className="card">
        <h2>Completing login...</h2>
        <p>Exchanging authorization code for tokens.</p>
      </div>
    </div>
  );
}

function DashboardPage() {
  const user = getUserInfo();
  const [profile, setProfile] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const token = getAccessToken();
    fetch("/api/user/profile", {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => {
        if (!res.ok) throw new Error(`Request failed: ${res.status}`);
        return res.json();
      })
      .then((data) => {
        setProfile(data);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  return (
    <div className="container">
      <h1>Dashboard</h1>
      <p>This page is accessible to any authenticated user.</p>

      <div className="card">
        <h2>User Profile</h2>
        {loading && <p>Loading profile from backend...</p>}
        {error && (
          <div className="error-inline">
            <p>Could not load profile from backend: {error}</p>
            <p className="hint">
              Make sure the Node.js backend is running on port 3001.
            </p>
          </div>
        )}
        {profile && (
          <pre className="json-block">{JSON.stringify(profile, null, 2)}</pre>
        )}
      </div>

      {user && (
        <div className="card">
          <h2>JWT Claims (client-side)</h2>
          <pre className="json-block">{JSON.stringify(user, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}

function AdminPage() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const token = getAccessToken();
    fetch("/api/admin/data", {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => {
        if (!res.ok) throw new Error(`Request failed: ${res.status}`);
        return res.json();
      })
      .then((d) => {
        setData(d);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  return (
    <div className="container">
      <h1>Admin Panel</h1>
      <p>This page is only accessible to users with the <strong>admin</strong> role.</p>

      <div className="card">
        <h2>Admin Data</h2>
        {loading && <p>Loading admin data from backend...</p>}
        {error && (
          <div className="error-inline">
            <p>Could not load admin data: {error}</p>
            <p className="hint">
              Make sure the Node.js backend is running on port 3001.
            </p>
          </div>
        )}
        {data && (
          <pre className="json-block">{JSON.stringify(data, null, 2)}</pre>
        )}
      </div>
    </div>
  );
}

// --- App ---

export default function App() {
  return (
    <>
      <Nav />
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/callback" element={<CallbackPage />} />
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin"
          element={
            <ProtectedRoute requiredRole="admin">
              <AdminPage />
            </ProtectedRoute>
          }
        />
      </Routes>
    </>
  );
}
