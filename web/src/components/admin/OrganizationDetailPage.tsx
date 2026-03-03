import { useState, useEffect } from "react";
import {
  getOrg,
  updateOrg,
  deleteOrg,
  getOrgSettings,
  updateOrgSettings,
} from "../../api/organizations";
import type {
  OrgResponse,
  OrgSettingsResponse,
  UpdateOrgSettingsRequest,
} from "../../types";
import Modal from "./Modal";

interface OrganizationDetailPageProps {
  orgId: string;
  onNavigate: (page: string) => void;
}

type Tab = "general" | "password" | "mfa" | "sessions" | "branding" | "danger";

export default function OrganizationDetailPage({
  orgId,
  onNavigate,
}: OrganizationDetailPageProps) {
  const [org, setOrg] = useState<OrgResponse | null>(null);
  const [settings, setSettings] = useState<OrgSettingsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<Tab>("general");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    setLoading(true);
    Promise.all([getOrg(orgId), getOrgSettings(orgId)])
      .then(([o, s]) => {
        setOrg(o);
        setSettings(s);
      })
      .catch(() => setError("Failed to load organization."))
      .finally(() => setLoading(false));
  }, [orgId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-slate-200 border-t-slate-900" />
      </div>
    );
  }

  if (!org) {
    return (
      <div className="py-20 text-center text-slate-500">
        Organization not found.
      </div>
    );
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: "general", label: "General" },
    { id: "password", label: "Password Policy" },
    { id: "mfa", label: "MFA" },
    { id: "sessions", label: "Sessions" },
    { id: "branding", label: "Branding" },
    { id: "danger", label: "Danger Zone" },
  ];

  return (
    <div className="mx-auto max-w-4xl">
      <div className="flex items-center gap-3">
        <button
          onClick={() => onNavigate("organizations")}
          className="rounded-md p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-700"
        >
          <svg viewBox="0 0 20 20" fill="currentColor" className="h-5 w-5">
            <path
              fillRule="evenodd"
              d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z"
              clipRule="evenodd"
            />
          </svg>
        </button>
        <div>
          <h1 className="text-2xl font-bold text-slate-900">
            {org.display_name || org.name}
          </h1>
          <p className="text-sm text-slate-500">
            {org.slug} &middot; {org.user_count} users
          </p>
        </div>
      </div>

      {error && (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}
      {success && (
        <div className="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">
          {success}
        </div>
      )}

      <div className="mt-6 flex gap-1 border-b border-slate-200">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => {
              setTab(t.id);
              setError("");
              setSuccess("");
            }}
            className={`px-4 py-2 text-sm font-medium transition-colors ${
              tab === t.id
                ? "border-b-2 border-slate-900 text-slate-900"
                : "text-slate-500 hover:text-slate-700"
            } ${t.id === "danger" ? "ml-auto text-red-600 hover:text-red-700" : ""}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="mt-6">
        {tab === "general" && (
          <GeneralTab
            org={org}
            onSave={(updated) => {
              setOrg(updated);
              setSuccess("Organization updated.");
              setTimeout(() => setSuccess(""), 3000);
            }}
            onError={setError}
          />
        )}
        {tab === "password" && settings && (
          <PasswordTab
            orgId={orgId}
            settings={settings}
            onSave={(s) => {
              setSettings(s);
              setSuccess("Password policy updated.");
              setTimeout(() => setSuccess(""), 3000);
            }}
            onError={setError}
          />
        )}
        {tab === "mfa" && settings && (
          <MFATab
            orgId={orgId}
            settings={settings}
            onSave={(s) => {
              setSettings(s);
              setSuccess("MFA settings updated.");
              setTimeout(() => setSuccess(""), 3000);
            }}
            onError={setError}
          />
        )}
        {tab === "sessions" && settings && (
          <SessionsTab
            orgId={orgId}
            settings={settings}
            onSave={(s) => {
              setSettings(s);
              setSuccess("Session settings updated.");
              setTimeout(() => setSuccess(""), 3000);
            }}
            onError={setError}
          />
        )}
        {tab === "branding" && settings && (
          <BrandingTab
            orgId={orgId}
            settings={settings}
            onSave={(s) => {
              setSettings(s);
              setSuccess("Branding updated.");
              setTimeout(() => setSuccess(""), 3000);
            }}
            onError={setError}
          />
        )}
        {tab === "danger" && (
          <DangerTab
            org={org}
            onDelete={() => onNavigate("organizations")}
            onError={setError}
          />
        )}
      </div>
    </div>
  );
}

// --- General Tab ---

function GeneralTab({
  org,
  onSave,
  onError,
}: {
  org: OrgResponse;
  onSave: (o: OrgResponse) => void;
  onError: (e: string) => void;
}) {
  const [name, setName] = useState(org.name);
  const [displayName, setDisplayName] = useState(org.display_name);
  const [enabled, setEnabled] = useState(org.enabled);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    onError("");
    try {
      const updated = await updateOrg(org.id, {
        name,
        display_name: displayName,
        enabled,
      });
      onSave(updated);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Failed to update.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-700">Name</label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="mt-1 block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-slate-700">
          Slug (read-only)
        </label>
        <input
          type="text"
          value={org.slug}
          disabled
          className="mt-1 block w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-500"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-slate-700">
          Display Name
        </label>
        <input
          type="text"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          className="mt-1 block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        />
      </div>
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id="org-enabled"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
          className="h-4 w-4 rounded border-slate-300"
        />
        <label htmlFor="org-enabled" className="text-sm text-slate-700">
          Enabled
        </label>
      </div>
      <button
        onClick={handleSave}
        disabled={saving}
        className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 disabled:opacity-50"
      >
        {saving ? "Saving..." : "Save"}
      </button>
    </div>
  );
}

// --- Password Policy Tab ---

function PasswordTab({
  orgId,
  settings,
  onSave,
  onError,
}: {
  orgId: string;
  settings: OrgSettingsResponse;
  onSave: (s: OrgSettingsResponse) => void;
  onError: (e: string) => void;
}) {
  const [minLength, setMinLength] = useState(settings.password_min_length);
  const [requireUpper, setRequireUpper] = useState(settings.password_require_uppercase);
  const [requireLower, setRequireLower] = useState(settings.password_require_lowercase);
  const [requireNumbers, setRequireNumbers] = useState(settings.password_require_numbers);
  const [requireSymbols, setRequireSymbols] = useState(settings.password_require_symbols);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    onError("");
    try {
      const req: UpdateOrgSettingsRequest = {
        ...settingsToRequest(settings),
        password_min_length: minLength,
        password_require_uppercase: requireUpper,
        password_require_lowercase: requireLower,
        password_require_numbers: requireNumbers,
        password_require_symbols: requireSymbols,
      };
      const updated = await updateOrgSettings(orgId, req);
      onSave(updated);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Failed to update.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-700">
          Minimum Password Length
        </label>
        <input
          type="number"
          min={1}
          max={128}
          value={minLength}
          onChange={(e) => setMinLength(Number(e.target.value))}
          className="mt-1 block w-32 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        />
      </div>
      <Checkbox label="Require uppercase letters" checked={requireUpper} onChange={setRequireUpper} />
      <Checkbox label="Require lowercase letters" checked={requireLower} onChange={setRequireLower} />
      <Checkbox label="Require numbers" checked={requireNumbers} onChange={setRequireNumbers} />
      <Checkbox label="Require special characters" checked={requireSymbols} onChange={setRequireSymbols} />
      <button
        onClick={handleSave}
        disabled={saving}
        className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 disabled:opacity-50"
      >
        {saving ? "Saving..." : "Save Password Policy"}
      </button>
    </div>
  );
}

// --- MFA Tab ---

function MFATab({
  orgId,
  settings,
  onSave,
  onError,
}: {
  orgId: string;
  settings: OrgSettingsResponse;
  onSave: (s: OrgSettingsResponse) => void;
  onError: (e: string) => void;
}) {
  const [mfa, setMfa] = useState(settings.mfa_enforcement);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    onError("");
    try {
      const req: UpdateOrgSettingsRequest = {
        ...settingsToRequest(settings),
        mfa_enforcement: mfa,
      };
      const updated = await updateOrgSettings(orgId, req);
      onSave(updated);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Failed to update.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-700">
          MFA Enforcement
        </label>
        <select
          value={mfa}
          onChange={(e) => setMfa(e.target.value)}
          className="mt-1 block w-64 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        >
          <option value="off">Off</option>
          <option value="optional">Optional</option>
          <option value="required">Required</option>
        </select>
        <p className="mt-1 text-xs text-slate-500">
          {mfa === "off" && "MFA is disabled for all users in this organization."}
          {mfa === "optional" && "Users can choose to enable MFA on their account."}
          {mfa === "required" && "All users must set up MFA to access their account."}
        </p>
      </div>
      <button
        onClick={handleSave}
        disabled={saving}
        className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 disabled:opacity-50"
      >
        {saving ? "Saving..." : "Save MFA Settings"}
      </button>
    </div>
  );
}

// --- Sessions Tab ---

function SessionsTab({
  orgId,
  settings,
  onSave,
  onError,
}: {
  orgId: string;
  settings: OrgSettingsResponse;
  onSave: (s: OrgSettingsResponse) => void;
  onError: (e: string) => void;
}) {
  const [accessTTL, setAccessTTL] = useState(settings.access_token_ttl_seconds / 60);
  const [refreshTTL, setRefreshTTL] = useState(settings.refresh_token_ttl_seconds / 86400);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    onError("");
    try {
      const req: UpdateOrgSettingsRequest = {
        ...settingsToRequest(settings),
        access_token_ttl_seconds: accessTTL * 60,
        refresh_token_ttl_seconds: refreshTTL * 86400,
      };
      const updated = await updateOrgSettings(orgId, req);
      onSave(updated);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Failed to update.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-700">
          Access Token TTL (minutes)
        </label>
        <input
          type="number"
          min={1}
          value={accessTTL}
          onChange={(e) => setAccessTTL(Number(e.target.value))}
          className="mt-1 block w-32 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-slate-700">
          Refresh Token TTL (days)
        </label>
        <input
          type="number"
          min={1}
          value={refreshTTL}
          onChange={(e) => setRefreshTTL(Number(e.target.value))}
          className="mt-1 block w-32 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        />
      </div>
      <button
        onClick={handleSave}
        disabled={saving}
        className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 disabled:opacity-50"
      >
        {saving ? "Saving..." : "Save Session Settings"}
      </button>
    </div>
  );
}

// --- Branding Tab ---

function BrandingTab({
  orgId,
  settings,
  onSave,
  onError,
}: {
  orgId: string;
  settings: OrgSettingsResponse;
  onSave: (s: OrgSettingsResponse) => void;
  onError: (e: string) => void;
}) {
  const [logoUrl, setLogoUrl] = useState(settings.logo_url ?? "");
  const [primaryColor, setPrimaryColor] = useState(settings.primary_color ?? "");
  const [bgColor, setBgColor] = useState(settings.background_color ?? "");
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    onError("");
    try {
      const req: UpdateOrgSettingsRequest = {
        ...settingsToRequest(settings),
        logo_url: logoUrl,
        primary_color: primaryColor,
        background_color: bgColor,
      };
      const updated = await updateOrgSettings(orgId, req);
      onSave(updated);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Failed to update.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-700">Logo URL</label>
        <input
          type="url"
          value={logoUrl}
          onChange={(e) => setLogoUrl(e.target.value)}
          placeholder="https://example.com/logo.png"
          className="mt-1 block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
        />
      </div>
      <div className="flex gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-700">
            Primary Color
          </label>
          <div className="mt-1 flex items-center gap-2">
            <input
              type="color"
              value={primaryColor || "#0f172a"}
              onChange={(e) => setPrimaryColor(e.target.value)}
              className="h-9 w-9 cursor-pointer rounded border border-slate-300"
            />
            <input
              type="text"
              value={primaryColor}
              onChange={(e) => setPrimaryColor(e.target.value)}
              placeholder="#0f172a"
              className="block w-28 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
            />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700">
            Background Color
          </label>
          <div className="mt-1 flex items-center gap-2">
            <input
              type="color"
              value={bgColor || "#f8fafc"}
              onChange={(e) => setBgColor(e.target.value)}
              className="h-9 w-9 cursor-pointer rounded border border-slate-300"
            />
            <input
              type="text"
              value={bgColor}
              onChange={(e) => setBgColor(e.target.value)}
              placeholder="#f8fafc"
              className="block w-28 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none focus:ring-1 focus:ring-slate-500"
            />
          </div>
        </div>
      </div>
      <button
        onClick={handleSave}
        disabled={saving}
        className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 disabled:opacity-50"
      >
        {saving ? "Saving..." : "Save Branding"}
      </button>
    </div>
  );
}

// --- Danger Zone Tab ---

function DangerTab({
  org,
  onDelete,
  onError,
}: {
  org: OrgResponse;
  onDelete: () => void;
  onError: (e: string) => void;
}) {
  const [showModal, setShowModal] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const isDefault = org.slug === "default";

  async function handleDelete() {
    setDeleting(true);
    onError("");
    try {
      await deleteOrg(org.id);
      onDelete();
    } catch (err) {
      onError(err instanceof Error ? err.message : "Failed to delete.");
      setShowModal(false);
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="rounded-xl border border-red-200 bg-red-50 p-6">
      <h3 className="text-lg font-semibold text-red-900">Delete Organization</h3>
      <p className="mt-1 text-sm text-red-700">
        This will permanently delete the organization and all associated users and settings.
        This action cannot be undone.
      </p>
      {isDefault && (
        <p className="mt-2 text-sm font-medium text-red-800">
          The default organization cannot be deleted.
        </p>
      )}
      <button
        onClick={() => setShowModal(true)}
        disabled={isDefault}
        className="mt-4 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
      >
        Delete Organization
      </button>

      <Modal
        open={showModal}
        title="Delete Organization"
        message={`Are you sure you want to delete "${org.display_name || org.name}"? All users in this organization will be permanently removed.`}
        confirmLabel={deleting ? "Deleting..." : "Delete"}
        confirmVariant="danger"
        onConfirm={handleDelete}
        onCancel={() => setShowModal(false)}
      />
    </div>
  );
}

// --- Helpers ---

function Checkbox({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 rounded border-slate-300"
      />
      <span className="text-sm text-slate-700">{label}</span>
    </div>
  );
}

function settingsToRequest(s: OrgSettingsResponse): UpdateOrgSettingsRequest {
  return {
    password_min_length: s.password_min_length,
    password_require_uppercase: s.password_require_uppercase,
    password_require_lowercase: s.password_require_lowercase,
    password_require_numbers: s.password_require_numbers,
    password_require_symbols: s.password_require_symbols,
    mfa_enforcement: s.mfa_enforcement,
    access_token_ttl_seconds: s.access_token_ttl_seconds,
    refresh_token_ttl_seconds: s.refresh_token_ttl_seconds,
    logo_url: s.logo_url ?? "",
    primary_color: s.primary_color ?? "",
    background_color: s.background_color ?? "",
  };
}
