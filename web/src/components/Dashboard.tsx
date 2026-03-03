import { useEffect, useState } from "react";
import { getMe, logout } from "../api/auth";
import type { MeResponse } from "../types";

interface Props {
  onLogout: () => void;
}

export default function Dashboard({ onLogout }: Props) {
  const [user, setUser] = useState<MeResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    getMe().then((me) => {
      if (!me) {
        onLogout();
        return;
      }
      setUser(me);
      setLoading(false);
    });
  }, [onLogout]);

  async function handleLogout() {
    setLoggingOut(true);
    await logout();
    onLogout();
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <svg className="h-6 w-6 animate-spin text-slate-400" viewBox="0 0 24 24" fill="none">
          <circle
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="3"
            className="opacity-25"
          />
          <path
            d="M4 12a8 8 0 018-8"
            stroke="currentColor"
            strokeWidth="3"
            strokeLinecap="round"
            className="opacity-75"
          />
        </svg>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-lg">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Dashboard</h1>
          <p className="text-sm text-slate-500">
            Welcome, <span className="font-semibold text-slate-700">{user?.preferred_username}</span>
          </p>
        </div>
        <button
          onClick={handleLogout}
          disabled={loggingOut}
          className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 focus:ring-2 focus:ring-slate-200 focus:outline-none disabled:opacity-50"
        >
          {loggingOut ? "Signing out..." : "Sign out"}
        </button>
      </div>

      <div className="rounded-lg border border-slate-100 bg-slate-50 p-4">
        <h2 className="mb-3 text-sm font-medium text-slate-500 uppercase tracking-wider">
          Account Details
        </h2>
        <dl className="space-y-2 text-sm">
          <div className="flex justify-between">
            <dt className="text-slate-500">Username</dt>
            <dd className="font-medium text-slate-900">{user?.preferred_username}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-slate-500">Email</dt>
            <dd className="font-medium text-slate-900">{user?.email}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-slate-500">Email Verified</dt>
            <dd className="font-medium text-slate-900">
              {user?.email_verified ? "Yes" : "No"}
            </dd>
          </div>
          {user?.given_name && (
            <div className="flex justify-between">
              <dt className="text-slate-500">Name</dt>
              <dd className="font-medium text-slate-900">
                {user.given_name} {user.family_name}
              </dd>
            </div>
          )}
          <div className="flex justify-between">
            <dt className="text-slate-500">User ID</dt>
            <dd className="font-mono text-xs text-slate-600">{user?.id}</dd>
          </div>
        </dl>
      </div>
    </div>
  );
}
