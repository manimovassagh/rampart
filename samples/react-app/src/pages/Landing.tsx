import { useState } from "react";
import { useAuth } from "@rampart/react";

export function Landing() {
  const { loginWithRedirect } = useAuth();
  const [unauthResponse, setUnauthResponse] = useState<{
    status: number;
    body: unknown;
  } | null>(null);

  const testUnauth = async () => {
    try {
      const res = await fetch("/api/profile");
      const body = await res.json().catch(() => null);
      setUnauthResponse({ status: res.status, body });
    } catch (err) {
      setUnauthResponse({ status: 0, body: { error: String(err) } });
    }
  };

  return (
    <div className="space-y-8">
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">
          Rampart React Sample
        </h1>
        <p className="text-gray-500">
          Full-stack auth demo with{" "}
          <code className="bg-gray-100 px-1.5 py-0.5 rounded text-sm">
            @rampart/react
          </code>
        </p>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-6 text-center">
        <h3 className="text-lg font-semibold text-gray-900 mb-3">
          Get Started
        </h3>
        <p className="text-sm text-gray-500 mb-6">
          Click below to sign in via the Rampart authorization server.
        </p>
        <button
          onClick={() => loginWithRedirect()}
          className="px-6 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 transition-colors"
        >
          Login with Rampart
        </button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-3">
          Test Without Auth
        </h3>
        <p className="text-sm text-gray-500 mb-4">
          Try hitting a protected API endpoint without logging in to see the 401
          response.
        </p>
        <button
          onClick={testUnauth}
          className="px-3 py-1.5 text-sm rounded-md bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors font-mono"
        >
          GET /api/profile (no auth)
        </button>

        {unauthResponse && (
          <div className="mt-4">
            <span
              className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium mb-2 ${
                unauthResponse.status >= 200 && unauthResponse.status < 300
                  ? "bg-green-100 text-green-800"
                  : "bg-red-100 text-red-800"
              }`}
            >
              {unauthResponse.status}
            </span>
            <pre className="bg-gray-900 text-gray-100 p-4 rounded-md text-xs overflow-auto">
              {JSON.stringify(unauthResponse.body, null, 2)}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}
