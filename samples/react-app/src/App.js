import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth, ProtectedRoute } from "@rampart/react";
import { Navbar } from "./components/Navbar";
import { Landing } from "./pages/Landing";
import { Dashboard } from "./pages/Dashboard";
import { Admin } from "./pages/Admin";
import { Callback } from "./pages/Callback";
export function App() {
    const { isAuthenticated, isLoading } = useAuth();
    if (isLoading) {
        return (_jsx("div", { className: "min-h-screen flex items-center justify-center bg-gray-50", children: _jsx("div", { className: "text-gray-500 text-lg", children: "Loading..." }) }));
    }
    return (_jsxs("div", { className: "min-h-screen bg-gray-50", children: [_jsx(Navbar, {}), _jsx("main", { className: "max-w-4xl mx-auto px-4 py-8", children: _jsxs(Routes, { children: [_jsx(Route, { path: "/", element: isAuthenticated ? _jsx(Navigate, { to: "/dashboard", replace: true }) : _jsx(Landing, {}) }), _jsx(Route, { path: "/callback", element: _jsx(Callback, {}) }), _jsx(Route, { path: "/dashboard", element: _jsx(ProtectedRoute, { fallback: _jsx(Navigate, { to: "/", replace: true }), children: _jsx(Dashboard, {}) }) }), _jsx(Route, { path: "/admin", element: _jsx(ProtectedRoute, { roles: ["admin"], fallback: _jsxs("div", { className: "text-center py-16", children: [_jsx("h2", { className: "text-2xl font-bold text-gray-800 mb-2", children: "Insufficient Permissions" }), _jsxs("p", { className: "text-gray-500", children: ["You need the ", _jsx("code", { className: "bg-gray-100 px-2 py-0.5 rounded text-sm", children: "admin" }), " role to access this page."] })] }), children: _jsx(Admin, {}) }) })] }) })] }));
}
