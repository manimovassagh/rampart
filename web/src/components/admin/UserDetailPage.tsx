import { useState, useEffect, useCallback } from "react";
import {
  getUser,
  updateUser,
  deleteUser,
  resetPassword,
  listSessions,
  revokeSessions,
} from "../../api/admin";
import type {
  AdminUserResponse,
  SessionResponse,
  UpdateUserRequest,
} from "../../types";
import Modal from "./Modal";
import { toast } from "./Toast";

interface UserDetailPageProps {
  userId: string;
  onNavigate: (page: string) => void;
}

export default function UserDetailPage({
  userId,
  onNavigate,
}: UserDetailPageProps) {
  const [user, setUser] = useState<AdminUserResponse | null>(null);
  const [sessions, setSessions] = useState<SessionResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  // Form state
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [givenName, setGivenName] = useState("");
  const [familyName, setFamilyName] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [emailVerified, setEmailVerified] = useState(false);

  // Password reset
  const [newPassword, setNewPassword] = useState("");

  // Modals
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showRevokeModal, setShowRevokeModal] = useState(false);

  const fetchData = useCallback(() => {
    setLoading(true);
    Promise.all([getUser(userId), listSessions(userId)])
      .then(([u, s]) => {
        setUser(u);
        setSessions(s);
        setUsername(u.username);
        setEmail(u.email);
        setGivenName(u.given_name ?? "");
        setFamilyName(u.family_name ?? "");
        setEnabled(u.enabled);
        setEmailVerified(u.email_verified);
      })
      .catch(() => toast("error", "Failed to load user"))
      .finally(() => setLoading(false));
  }, [userId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const req: UpdateUserRequest = {
        username,
        email,
        given_name: givenName,
        family_name: familyName,
        enabled,
        email_verified: emailVerified,
      };
      const updated = await updateUser(userId, req);
      setUser(updated);
      toast("success", "User updated");
    } catch (e) {
      toast("error", (e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    try {
      await deleteUser(userId);
      toast("success", "User deleted");
      onNavigate("users");
    } catch (e) {
      toast("error", (e as Error).message);
    }
    setShowDeleteModal(false);
  };

  const handleResetPassword = async () => {
    if (!newPassword) return;
    try {
      await resetPassword(userId, { password: newPassword });
      setNewPassword("");
      toast("success", "Password reset");
    } catch (e) {
      toast("error", (e as Error).message);
    }
  };

  const handleRevokeSessions = async () => {
    try {
      await revokeSessions(userId);
      setSessions([]);
      toast("success", "All sessions revoked");
    } catch (e) {
      toast("error", (e as Error).message);
    }
    setShowRevokeModal(false);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-slate-200 border-t-slate-900" />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="text-center py-20 text-slate-500">User not found</div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl">
      <button
        onClick={() => onNavigate("users")}
        className="mb-4 text-sm text-slate-500 hover:text-slate-700"
      >
        &larr; Back to users
      </button>

      <h1 className="text-2xl font-bold text-slate-900">{user.username}</h1>
      <p className="mt-1 text-sm text-slate-500">{user.email}</p>

      {/* Profile Card */}
      <div className="mt-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Profile</h2>
        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <Field label="Username" value={username} onChange={setUsername} />
          <Field label="Email" value={email} onChange={setEmail} />
          <Field label="First name" value={givenName} onChange={setGivenName} />
          <Field
            label="Last name"
            value={familyName}
            onChange={setFamilyName}
          />
        </div>
        <div className="mt-4 flex flex-wrap gap-4">
          <Toggle label="Enabled" checked={enabled} onChange={setEnabled} />
          <Toggle
            label="Email verified"
            checked={emailVerified}
            onChange={setEmailVerified}
          />
        </div>
        <div className="mt-6">
          <button
            onClick={handleSave}
            disabled={saving}
            className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 disabled:opacity-50"
          >
            {saving ? "Saving..." : "Save changes"}
          </button>
        </div>
      </div>

      {/* Sessions Card */}
      <div className="mt-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">
            Sessions ({sessions.length})
          </h2>
          {sessions.length > 0 && (
            <button
              onClick={() => setShowRevokeModal(true)}
              className="text-sm font-medium text-red-600 hover:text-red-700"
            >
              Revoke all
            </button>
          )}
        </div>
        {sessions.length === 0 ? (
          <p className="mt-3 text-sm text-slate-400">No active sessions</p>
        ) : (
          <div className="mt-3 divide-y divide-slate-100">
            {sessions.map((s) => (
              <div key={s.id} className="flex items-center justify-between py-2">
                <div>
                  <p className="text-sm font-mono text-slate-700">
                    {s.id.slice(0, 8)}...
                  </p>
                  <p className="text-xs text-slate-400">
                    Created {new Date(s.created_at).toLocaleString()}
                  </p>
                </div>
                <span className="text-xs text-slate-400">
                  Expires {new Date(s.expires_at).toLocaleString()}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Password Reset Card */}
      <div className="mt-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Reset Password</h2>
        <div className="mt-3 flex gap-3">
          <input
            type="password"
            placeholder="New password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            className="flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
          />
          <button
            onClick={handleResetPassword}
            disabled={!newPassword}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
          >
            Reset
          </button>
        </div>
      </div>

      {/* Danger Zone */}
      <div className="mt-6 rounded-xl border border-red-200 bg-red-50 p-6">
        <h2 className="text-lg font-semibold text-red-900">Danger Zone</h2>
        <p className="mt-1 text-sm text-red-700">
          Deleting this user is permanent and cannot be undone.
        </p>
        <button
          onClick={() => setShowDeleteModal(true)}
          className="mt-4 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
        >
          Delete user
        </button>
      </div>

      {/* Modals */}
      <Modal
        open={showDeleteModal}
        title="Delete user"
        message={`Are you sure you want to delete ${user.username}? This action cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={handleDelete}
        onCancel={() => setShowDeleteModal(false)}
      />
      <Modal
        open={showRevokeModal}
        title="Revoke all sessions"
        message={`This will sign ${user.username} out of all devices immediately.`}
        confirmLabel="Revoke all"
        confirmVariant="danger"
        onConfirm={handleRevokeSessions}
        onCancel={() => setShowRevokeModal(false)}
      />
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
      />
    </div>
  );
}

function Toggle({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 text-sm text-slate-700">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 rounded border-slate-300 text-slate-900 focus:ring-slate-500"
      />
      {label}
    </label>
  );
}
