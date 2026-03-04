import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@rampart/react";
export function Callback() {
    const { handleCallback } = useAuth();
    const navigate = useNavigate();
    const [error, setError] = useState(null);
    useEffect(() => {
        let cancelled = false;
        async function exchange() {
            try {
                await handleCallback();
                if (!cancelled)
                    navigate("/dashboard", { replace: true });
            }
            catch (err) {
                if (!cancelled) {
                    const message = err && typeof err === "object" && "error_description" in err
                        ? String(err.error_description)
                        : "Authentication failed.";
                    setError(message);
                }
            }
        }
        exchange();
        return () => {
            cancelled = true;
        };
    }, [handleCallback, navigate]);
    if (error) {
        return (_jsx("div", { className: "min-h-screen flex items-center justify-center bg-gray-50", children: _jsxs("div", { className: "bg-white rounded-lg border border-red-200 p-6 max-w-md text-center", children: [_jsx("h2", { className: "text-lg font-semibold text-red-700 mb-2", children: "Login Failed" }), _jsx("p", { className: "text-gray-600 text-sm mb-4", children: error }), _jsx("a", { href: "/", className: "text-blue-600 hover:text-blue-800 text-sm font-medium", children: "Back to Home" })] }) }));
    }
    return (_jsx("div", { className: "min-h-screen flex items-center justify-center bg-gray-50", children: _jsx("div", { className: "text-gray-500 text-lg", children: "Completing login..." }) }));
}
