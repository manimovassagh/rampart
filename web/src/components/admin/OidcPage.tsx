import { useState, useEffect, useCallback } from "react";

interface EndpointInfo {
  label: string;
  url: string;
  description: string;
}

export default function OidcPage() {
  const [discovery, setDiscovery] = useState<Record<string, unknown> | null>(null);
  const [jwks, setJwks] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"endpoints" | "discovery" | "jwks">("endpoints");
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      fetch("/.well-known/openid-configuration").then((r) => r.json()),
      fetch("/.well-known/jwks.json").then((r) => r.json()),
    ])
      .then(([disc, keys]) => {
        setDiscovery(disc as Record<string, unknown>);
        setJwks(keys as Record<string, unknown>);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const copyToClipboard = useCallback((text: string, label: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(label);
      setTimeout(() => setCopied(null), 2000);
    });
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-slate-200 border-t-slate-900" />
      </div>
    );
  }

  const issuer = (discovery?.issuer as string) ?? "";
  const baseUrl = issuer || window.location.origin;

  const endpoints: EndpointInfo[] = [
    {
      label: "Issuer",
      url: issuer,
      description: "The issuer identifier — use this to configure SDKs and adapters",
    },
    {
      label: "Discovery",
      url: `${baseUrl}/.well-known/openid-configuration`,
      description: "OIDC Discovery endpoint — auto-configure clients from this URL",
    },
    {
      label: "JWKS",
      url: (discovery?.jwks_uri as string) ?? `${baseUrl}/.well-known/jwks.json`,
      description: "JSON Web Key Set — public keys for verifying tokens",
    },
    {
      label: "Token Endpoint",
      url: (discovery?.token_endpoint as string) ?? "",
      description: "Where clients exchange credentials for tokens",
    },
    {
      label: "UserInfo Endpoint",
      url: (discovery?.userinfo_endpoint as string) ?? "",
      description: "Returns claims about the authenticated user",
    },
  ];

  const tabs = [
    { id: "endpoints" as const, label: "Endpoints" },
    { id: "discovery" as const, label: "Discovery JSON" },
    { id: "jwks" as const, label: "JWKS JSON" },
  ];

  return (
    <div className="mx-auto max-w-4xl">
      <h1 className="text-2xl font-bold text-slate-900">OIDC Configuration</h1>
      <p className="mt-1 text-sm text-slate-500">
        OpenID Connect endpoints and public keys for integrating with Rampart
      </p>

      {/* Tabs */}
      <div className="mt-6 flex gap-1 rounded-lg bg-slate-100 p-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? "bg-white text-slate-900 shadow-sm"
                : "text-slate-600 hover:text-slate-900"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Endpoints tab */}
      {activeTab === "endpoints" && (
        <div className="mt-4 space-y-3">
          {endpoints.map((ep) => (
            <div
              key={ep.label}
              className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm"
            >
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-semibold text-slate-900">{ep.label}</p>
                  <p className="mt-0.5 text-xs text-slate-400">{ep.description}</p>
                  <div className="mt-2 flex items-center gap-2">
                    <code className="block min-w-0 flex-1 truncate rounded bg-slate-50 px-3 py-1.5 text-xs text-slate-700">
                      {ep.url}
                    </code>
                    <button
                      onClick={() => copyToClipboard(ep.url, ep.label)}
                      className={`flex-shrink-0 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors ${
                        copied === ep.label
                          ? "border-emerald-300 bg-emerald-50 text-emerald-700"
                          : "border-slate-300 text-slate-600 hover:bg-slate-50"
                      }`}
                    >
                      {copied === ep.label ? "Copied!" : "Copy"}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Discovery JSON tab */}
      {activeTab === "discovery" && (
        <div className="mt-4">
          <div className="flex items-center justify-between rounded-t-xl border border-b-0 border-slate-200 bg-slate-50 px-4 py-2.5">
            <p className="text-xs font-semibold text-slate-600">
              /.well-known/openid-configuration
            </p>
            <button
              onClick={() =>
                copyToClipboard(JSON.stringify(discovery, null, 2), "discovery")
              }
              className={`rounded-md border px-3 py-1 text-xs font-medium transition-colors ${
                copied === "discovery"
                  ? "border-emerald-300 bg-emerald-50 text-emerald-700"
                  : "border-slate-300 text-slate-600 hover:bg-slate-50"
              }`}
            >
              {copied === "discovery" ? "Copied!" : "Copy JSON"}
            </button>
          </div>
          <pre className="overflow-x-auto rounded-b-xl border border-slate-200 bg-white p-4 text-xs leading-relaxed text-slate-700">
            {JSON.stringify(discovery, null, 2)}
          </pre>
        </div>
      )}

      {/* JWKS JSON tab */}
      {activeTab === "jwks" && (
        <div className="mt-4">
          <div className="flex items-center justify-between rounded-t-xl border border-b-0 border-slate-200 bg-slate-50 px-4 py-2.5">
            <p className="text-xs font-semibold text-slate-600">
              /.well-known/jwks.json
            </p>
            <button
              onClick={() =>
                copyToClipboard(JSON.stringify(jwks, null, 2), "jwks")
              }
              className={`rounded-md border px-3 py-1 text-xs font-medium transition-colors ${
                copied === "jwks"
                  ? "border-emerald-300 bg-emerald-50 text-emerald-700"
                  : "border-slate-300 text-slate-600 hover:bg-slate-50"
              }`}
            >
              {copied === "jwks" ? "Copied!" : "Copy JSON"}
            </button>
          </div>
          <pre className="overflow-x-auto rounded-b-xl border border-slate-200 bg-white p-4 text-xs leading-relaxed text-slate-700">
            {JSON.stringify(jwks, null, 2)}
          </pre>
        </div>
      )}

      {/* Quick integration guide */}
      <div className="mt-6 rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 className="text-sm font-semibold text-slate-900">Quick Integration</h2>
        <p className="mt-1 text-xs text-slate-500">
          Use the Discovery URL to auto-configure any OIDC client library:
        </p>
        <div className="mt-3 rounded-lg bg-slate-900 p-4">
          <code className="block text-xs leading-relaxed text-emerald-400">
            <span className="text-slate-500"># Fetch OIDC configuration</span>
            {"\n"}curl {baseUrl}/.well-known/openid-configuration
            {"\n\n"}
            <span className="text-slate-500"># Fetch public keys</span>
            {"\n"}curl {baseUrl}/.well-known/jwks.json
          </code>
        </div>
      </div>
    </div>
  );
}
