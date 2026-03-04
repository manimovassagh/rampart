import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { useAuth } from "@rampart/react";
export function Landing() {
    const { loginWithRedirect } = useAuth();
    const [unauthResponse, setUnauthResponse] = useState(null);
    const testUnauth = async () => {
        try {
            const res = await fetch("/api/profile");
            const body = await res.json().catch(() => null);
            setUnauthResponse({ status: res.status, body });
        }
        catch (err) {
            setUnauthResponse({ status: 0, body: { error: String(err) } });
        }
    };
    return (_jsxs("div", { className: "space-y-8", children: [_jsxs("div", { className: "text-center", children: [_jsx("h1", { className: "text-3xl font-bold text-gray-900 mb-2", children: "Rampart React Sample" }), _jsxs("p", { className: "text-gray-500", children: ["Full-stack auth demo with", " ", _jsx("code", { className: "bg-gray-100 px-1.5 py-0.5 rounded text-sm", children: "@rampart/react" })] })] }), _jsxs("div", { className: "bg-white rounded-lg border border-gray-200 p-6 text-center", children: [_jsx("h3", { className: "text-lg font-semibold text-gray-900 mb-3", children: "Get Started" }), _jsx("p", { className: "text-sm text-gray-500 mb-6", children: "Click below to sign in via the Rampart authorization server." }), _jsx("button", { onClick: () => loginWithRedirect(), className: "px-6 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 transition-colors", children: "Login with Rampart" })] }), _jsxs("div", { className: "bg-white rounded-lg border border-gray-200 p-6", children: [_jsx("h3", { className: "text-lg font-semibold text-gray-900 mb-3", children: "Test Without Auth" }), _jsx("p", { className: "text-sm text-gray-500 mb-4", children: "Try hitting a protected API endpoint without logging in to see the 401 response." }), _jsx("button", { onClick: testUnauth, className: "px-3 py-1.5 text-sm rounded-md bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors font-mono", children: "GET /api/profile (no auth)" }), unauthResponse && (_jsxs("div", { className: "mt-4", children: [_jsx("span", { className: `inline-flex items-center px-2 py-0.5 rounded text-xs font-medium mb-2 ${unauthResponse.status >= 200 && unauthResponse.status < 300
                                    ? "bg-green-100 text-green-800"
                                    : "bg-red-100 text-red-800"}`, children: unauthResponse.status }), _jsx("pre", { className: "bg-gray-900 text-gray-100 p-4 rounded-md text-xs overflow-auto", children: JSON.stringify(unauthResponse.body, null, 2) })] }))] })] }));
}
