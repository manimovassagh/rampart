import { useState, useEffect, useRef, useCallback } from "react";
import { listUsers } from "../../api/admin";
import type { AdminUserResponse, ListUsersResponse } from "../../types";

interface UsersPageProps {
  onNavigate: (page: string) => void;
}

export default function UsersPage({ onNavigate }: UsersPageProps) {
  const [data, setData] = useState<ListUsersResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();

  const fetchUsers = useCallback(
    (p: number, s: string, st: string) => {
      setLoading(true);
      listUsers(p, 20, s, st)
        .then(setData)
        .catch(() => setData(null))
        .finally(() => setLoading(false));
    },
    [],
  );

  useEffect(() => {
    fetchUsers(page, search, status);
  }, [page, status, fetchUsers]);

  // Debounced search
  const handleSearchChange = (value: string) => {
    setSearch(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setPage(1);
      fetchUsers(1, value, status);
    }, 300);
  };

  const totalPages = data ? Math.ceil(data.total / data.limit) : 0;

  return (
    <div className="mx-auto max-w-5xl">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Users</h1>
          <p className="mt-1 text-sm text-slate-500">
            {data ? `${data.total} total users` : "Loading..."}
          </p>
        </div>
        <button
          onClick={() => onNavigate("users/new")}
          className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800"
        >
          Create user
        </button>
      </div>

      {/* Search + Filter */}
      <div className="mt-4 flex gap-3">
        <input
          type="text"
          placeholder="Search users..."
          value={search}
          onChange={(e) => handleSearchChange(e.target.value)}
          className="flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        />
        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            setPage(1);
          }}
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700 focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        >
          <option value="">All statuses</option>
          <option value="enabled">Enabled</option>
          <option value="disabled">Disabled</option>
        </select>
      </div>

      {/* Table */}
      <div className="mt-4 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-slate-900" />
          </div>
        ) : (
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50">
                <th className="px-4 py-3 font-medium text-slate-600">User</th>
                <th className="px-4 py-3 font-medium text-slate-600">Email</th>
                <th className="px-4 py-3 font-medium text-slate-600">Status</th>
                <th className="px-4 py-3 font-medium text-slate-600">Sessions</th>
                <th className="px-4 py-3 font-medium text-slate-600">Created</th>
              </tr>
            </thead>
            <tbody>
              {data?.users.map((user) => (
                <UserRow
                  key={user.id}
                  user={user}
                  onClick={() => onNavigate(`users/${user.id}`)}
                />
              ))}
              {data?.users.length === 0 && (
                <tr>
                  <td
                    colSpan={5}
                    className="px-4 py-8 text-center text-slate-400"
                  >
                    No users found
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <p className="text-sm text-slate-500">
            Page {page} of {totalPages}
          </p>
          <div className="flex gap-2">
            <button
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
            >
              Previous
            </button>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function UserRow({
  user,
  onClick,
}: {
  user: AdminUserResponse;
  onClick: () => void;
}) {
  return (
    <tr
      onClick={onClick}
      className="cursor-pointer border-b border-slate-100 transition-colors hover:bg-slate-50"
    >
      <td className="px-4 py-3">
        <div className="font-medium text-slate-900">{user.username}</div>
        {(user.given_name || user.family_name) && (
          <div className="text-xs text-slate-500">
            {[user.given_name, user.family_name].filter(Boolean).join(" ")}
          </div>
        )}
      </td>
      <td className="px-4 py-3 text-slate-600">{user.email}</td>
      <td className="px-4 py-3">
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
            user.enabled
              ? "bg-emerald-50 text-emerald-700"
              : "bg-red-50 text-red-700"
          }`}
        >
          {user.enabled ? "Enabled" : "Disabled"}
        </span>
      </td>
      <td className="px-4 py-3 text-slate-600">{user.session_count}</td>
      <td className="px-4 py-3 text-slate-500 text-xs">
        {new Date(user.created_at).toLocaleDateString()}
      </td>
    </tr>
  );
}
