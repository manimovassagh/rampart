import { useState } from "react";
import { useAuth } from "@rampart/react";

interface Endpoint {
  label: string;
  url: string;
}

const RAMPART_URL = "http://localhost:8080";

const ENDPOINTS: Endpoint[] = [
  { label: "GET /api/profile", url: "/api/profile" },
  { label: "GET /api/claims", url: "/api/claims" },
  { label: "GET /me", url: `${RAMPART_URL}/me` },
];

export function ApiTester() {
  const { authFetch } = useAuth();
  const [response, setResponse] = useState<{ status: number; body: unknown } | null>(null);
  const [loading, setLoading] = useState(false);
  const [activeEndpoint, setActiveEndpoint] = useState<string | null>(null);

  const testEndpoint = async (endpoint: Endpoint) => {
    setLoading(true);
    setActiveEndpoint(endpoint.label);
    try {
      const res = await authFetch(endpoint.url);
      const body = await res.json().catch(() => null);
      setResponse({ status: res.status, body });
    } catch (err) {
      setResponse({ status: 0, body: { error: String(err) } });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">API Tester</h3>

      <div className="flex flex-wrap gap-2 mb-4">
        {ENDPOINTS.map((ep) => (
          <button
            key={ep.url}
            onClick={() => testEndpoint(ep)}
            disabled={loading}
            className="px-3 py-1.5 text-sm rounded-md bg-gray-100 text-gray-700 hover:bg-gray-200 disabled:opacity-50 transition-colors font-mono"
          >
            {ep.label}
          </button>
        ))}
      </div>

      {response && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="text-sm text-gray-500">{activeEndpoint}</span>
            <span
              className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                response.status >= 200 && response.status < 300
                  ? "bg-green-100 text-green-800"
                  : "bg-red-100 text-red-800"
              }`}
            >
              {response.status}
            </span>
          </div>
          <pre className="bg-gray-900 text-gray-100 p-4 rounded-md text-xs overflow-auto max-h-64">
            {JSON.stringify(response.body, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}
