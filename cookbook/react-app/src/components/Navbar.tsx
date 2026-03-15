import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "@rampart-auth/react";

export function Navbar() {
  const { user, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate("/");
  };

  return (
    <nav className="bg-white border-b border-gray-200">
      <div className="max-w-4xl mx-auto px-4 h-14 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link to="/" className="text-lg font-bold text-gray-900">
            Rampart
          </Link>
          {isAuthenticated && (
            <>
              <Link
                to="/dashboard"
                className="text-sm text-gray-600 hover:text-gray-900"
              >
                Dashboard
              </Link>
              <Link
                to="/admin"
                className="text-sm text-gray-600 hover:text-gray-900"
              >
                Admin
              </Link>
            </>
          )}
        </div>

        <div className="flex items-center gap-4">
          {isAuthenticated ? (
            <>
              <span className="text-sm text-gray-600">
                {user?.preferred_username ?? user?.email}
              </span>
              <button
                onClick={handleLogout}
                className="text-sm px-3 py-1.5 rounded-md bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
              >
                Logout
              </button>
            </>
          ) : (
            <span className="text-sm text-gray-400">Not signed in</span>
          )}
        </div>
      </div>
    </nav>
  );
}
