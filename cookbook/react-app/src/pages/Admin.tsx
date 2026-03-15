import { useAuth } from "@rampart-auth/react";

export function Admin() {
  const { user } = useAuth();

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-gray-900">Admin Panel</h2>
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <p className="text-gray-600 mb-4">
          Welcome, <strong>{user?.preferred_username}</strong>. You have the{" "}
          <code className="bg-blue-100 text-blue-700 px-1.5 py-0.5 rounded text-sm">admin</code>{" "}
          role.
        </p>
        <div className="bg-gray-50 rounded-md p-4 text-sm text-gray-500">
          This page is protected by{" "}
          <code className="bg-gray-100 px-1.5 py-0.5 rounded">
            {"<ProtectedRoute roles={[\"admin\"]} />"}
          </code>
          . Users without the admin role see a fallback message instead.
        </div>
      </div>
    </div>
  );
}
