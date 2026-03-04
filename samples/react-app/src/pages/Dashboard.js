import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useAuth } from "@rampart/react";
import { UserCard } from "../components/UserCard";
import { ApiTester } from "../components/ApiTester";
export function Dashboard() {
    const { user } = useAuth();
    if (!user)
        return null;
    return (_jsxs("div", { className: "space-y-6", children: [_jsx("h2", { className: "text-2xl font-bold text-gray-900", children: "Dashboard" }), _jsx(UserCard, { user: user }), _jsx(ApiTester, {})] }));
}
