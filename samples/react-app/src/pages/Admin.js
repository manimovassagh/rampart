import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useAuth } from "@rampart/react";
export function Admin() {
    const { user } = useAuth();
    return (_jsxs("div", { className: "space-y-6", children: [_jsx("h2", { className: "text-2xl font-bold text-gray-900", children: "Admin Panel" }), _jsxs("div", { className: "bg-white rounded-lg border border-gray-200 p-6", children: [_jsxs("p", { className: "text-gray-600 mb-4", children: ["Welcome, ", _jsx("strong", { children: user?.preferred_username }), ". You have the", " ", _jsx("code", { className: "bg-blue-100 text-blue-700 px-1.5 py-0.5 rounded text-sm", children: "admin" }), " ", "role."] }), _jsxs("div", { className: "bg-gray-50 rounded-md p-4 text-sm text-gray-500", children: ["This page is protected by", " ", _jsx("code", { className: "bg-gray-100 px-1.5 py-0.5 rounded", children: "<ProtectedRoute roles={[\"admin\"]} />" }), ". Users without the admin role see a fallback message instead."] })] })] }));
}
