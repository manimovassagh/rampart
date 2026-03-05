import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@rampart/react";

export function Callback() {
  const { handleCallback } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const exchanged = useRef(false);

  useEffect(() => {
    if (exchanged.current) return;
    exchanged.current = true;

    async function exchange() {
      try {
        await handleCallback();
        navigate("/dashboard", { replace: true });
      } catch (err: unknown) {
        const message =
          err && typeof err === "object" && "error_description" in err
            ? String((err as { error_description: string }).error_description)
            : "Authentication failed.";
        setError(message);
      }
    }

    exchange();
  }, [handleCallback, navigate]);

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="bg-white rounded-lg border border-red-200 p-6 max-w-md text-center">
          <h2 className="text-lg font-semibold text-red-700 mb-2">
            Login Failed
          </h2>
          <p className="text-gray-600 text-sm mb-4">{error}</p>
          <a
            href="/"
            className="text-blue-600 hover:text-blue-800 text-sm font-medium"
          >
            Back to Home
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-gray-500 text-lg">Completing login...</div>
    </div>
  );
}
