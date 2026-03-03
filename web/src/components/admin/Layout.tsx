import { useState, useCallback } from "react";
import Sidebar from "./Sidebar";
import { logout } from "../../api/auth";
import { useCurrentOrg, clearOrgCache } from "../../hooks/useCurrentOrg";

interface LayoutProps {
  currentPage: string;
  onNavigate: (page: string) => void;
  onLogout: () => void;
  children: React.ReactNode;
}

export default function Layout({
  currentPage,
  onNavigate,
  onLogout,
  children,
}: LayoutProps) {
  const { org } = useCurrentOrg();
  const [collapsed, setCollapsed] = useState(false);

  const handleLogout = useCallback(async () => {
    clearOrgCache();
    await logout();
    onLogout();
  }, [onLogout]);

  return (
    <div className="flex h-screen bg-slate-50">
      <Sidebar
        collapsed={collapsed}
        onToggle={() => setCollapsed(!collapsed)}
        currentPage={currentPage}
        onNavigate={onNavigate}
      />

      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Topbar */}
        <header className="flex h-14 items-center justify-between border-b border-slate-200 bg-white px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-900">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-4 w-4 text-white"
              >
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              </svg>
            </div>
            <span className="text-sm font-bold tracking-tight text-slate-900">
              Rampart
            </span>
            {org && (
              <>
                <span className="text-slate-300">/</span>
                <span className="rounded bg-indigo-50 px-2 py-0.5 text-xs font-semibold text-indigo-700">
                  {org.display_name || org.name}
                </span>
              </>
            )}
          </div>

          <button
            onClick={handleLogout}
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50"
          >
            Sign out
          </button>
        </header>

        {/* Main content area */}
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
