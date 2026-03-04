import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "@rampart/react";
export function Navbar() {
    const { user, isAuthenticated, logout } = useAuth();
    const navigate = useNavigate();
    const handleLogout = async () => {
        await logout();
        navigate("/");
    };
    return (_jsx("nav", { className: "bg-white border-b border-gray-200", children: _jsxs("div", { className: "max-w-4xl mx-auto px-4 h-14 flex items-center justify-between", children: [_jsxs("div", { className: "flex items-center gap-6", children: [_jsx(Link, { to: "/", className: "text-lg font-bold text-gray-900", children: "Rampart" }), isAuthenticated && (_jsxs(_Fragment, { children: [_jsx(Link, { to: "/dashboard", className: "text-sm text-gray-600 hover:text-gray-900", children: "Dashboard" }), _jsx(Link, { to: "/admin", className: "text-sm text-gray-600 hover:text-gray-900", children: "Admin" })] }))] }), _jsx("div", { className: "flex items-center gap-4", children: isAuthenticated ? (_jsxs(_Fragment, { children: [_jsx("span", { className: "text-sm text-gray-600", children: user?.preferred_username ?? user?.email }), _jsx("button", { onClick: handleLogout, className: "text-sm px-3 py-1.5 rounded-md bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors", children: "Logout" })] })) : (_jsx("span", { className: "text-sm text-gray-400", children: "Not signed in" })) })] }) }));
}
