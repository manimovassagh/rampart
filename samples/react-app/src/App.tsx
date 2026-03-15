import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth, ProtectedRoute } from "@rampart-auth/react";
import { Navbar } from "./components/Navbar";
import { Landing } from "./pages/Landing";
import { Dashboard } from "./pages/Dashboard";
import { Admin } from "./pages/Admin";
import { Callback } from "./pages/Callback";

export function App() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-gray-500 text-lg">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <main className="max-w-4xl mx-auto px-4 py-8">
        <Routes>
          <Route
            path="/"
            element={
              isAuthenticated ? <Navigate to="/dashboard" replace /> : <Landing />
            }
          />
          <Route path="/callback" element={<Callback />} />
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute fallback={<Navigate to="/" replace />}>
                <Dashboard />
              </ProtectedRoute>
            }
          />
          <Route
            path="/admin"
            element={
              <ProtectedRoute
                roles={["admin"]}
                fallback={
                  <div className="text-center py-16">
                    <h2 className="text-2xl font-bold text-gray-800 mb-2">
                      Insufficient Permissions
                    </h2>
                    <p className="text-gray-500">
                      You need the <code className="bg-gray-100 px-2 py-0.5 rounded text-sm">admin</code> role to access this page.
                    </p>
                  </div>
                }
              >
                <Admin />
              </ProtectedRoute>
            }
          />
        </Routes>
      </main>
    </div>
  );
}
