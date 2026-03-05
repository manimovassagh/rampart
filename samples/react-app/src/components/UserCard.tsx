import type { RampartUser } from "@rampart/react";

interface UserCardProps {
  user: RampartUser;
}

export function UserCard({ user }: UserCardProps) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <div className="flex items-start justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">
            {user.given_name && user.family_name
              ? `${user.given_name} ${user.family_name}`
              : user.preferred_username ?? user.email}
          </h3>
          <p className="text-sm text-gray-500">@{user.preferred_username ?? user.username}</p>
        </div>
        <span
          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
            user.email_verified
              ? "bg-green-100 text-green-800"
              : "bg-yellow-100 text-yellow-800"
          }`}
        >
          {user.email_verified ? "Verified" : "Unverified"}
        </span>
      </div>

      <dl className="space-y-2 text-sm">
        <div className="flex justify-between">
          <dt className="text-gray-500">Email</dt>
          <dd className="text-gray-900">{user.email}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-gray-500">Org ID</dt>
          <dd className="text-gray-900 font-mono text-xs">{user.org_id}</dd>
        </div>
        {user.roles && user.roles.length > 0 && (
          <div className="flex justify-between">
            <dt className="text-gray-500">Roles</dt>
            <dd className="flex gap-1">
              {user.roles.map((role) => (
                <span
                  key={role}
                  className="bg-blue-100 text-blue-700 px-2 py-0.5 rounded text-xs font-medium"
                >
                  {role}
                </span>
              ))}
            </dd>
          </div>
        )}
      </dl>
    </div>
  );
}
