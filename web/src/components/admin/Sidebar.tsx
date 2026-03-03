import { useState, useEffect, useRef } from "react";
import { useCurrentOrg } from "../../hooks/useCurrentOrg";
import { listOrgs } from "../../api/organizations";
import type { OrgResponse } from "../../types";

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
  currentPage: string;
  onNavigate: (page: string) => void;
}

const navItems = [
  { id: "dashboard", label: "Dashboard", icon: ChartIcon },
  { id: "users", label: "Users", icon: UsersIcon },
  { id: "organizations", label: "Organizations", icon: OrgIcon },
  { id: "oidc", label: "OIDC", icon: KeyIcon },
];

export default function Sidebar({
  collapsed,
  onToggle,
  currentPage,
  onNavigate,
}: SidebarProps) {
  const { org, switchOrg } = useCurrentOrg();
  const [showSwitcher, setShowSwitcher] = useState(false);
  const [orgs, setOrgs] = useState<OrgResponse[]>([]);
  const switcherRef = useRef<HTMLDivElement>(null);

  // Close switcher on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (switcherRef.current && !switcherRef.current.contains(e.target as Node)) {
        setShowSwitcher(false);
      }
    }
    if (showSwitcher) {
      document.addEventListener("mousedown", handleClick);
    }
    return () => document.removeEventListener("mousedown", handleClick);
  }, [showSwitcher]);

  const handleOpenSwitcher = () => {
    if (collapsed) return;
    setShowSwitcher(!showSwitcher);
    if (!showSwitcher) {
      listOrgs(1, 50).then((data) => setOrgs(data.organizations)).catch(() => {});
    }
  };

  const handleSwitch = async (target: OrgResponse) => {
    if (target.id === org?.id) {
      setShowSwitcher(false);
      return;
    }
    await switchOrg(target.id);
    setShowSwitcher(false);
    // Navigate to dashboard to refresh data in the new org context
    onNavigate("dashboard");
  };

  return (
    <aside
      className={`flex flex-col border-r border-slate-200 bg-white transition-all duration-200 ${
        collapsed ? "w-16" : "w-56"
      }`}
    >
      <div className="flex h-14 items-center justify-between border-b border-slate-200 px-3">
        {!collapsed && (
          <span className="text-sm font-bold tracking-tight text-slate-900">
            Admin
          </span>
        )}
        <button
          onClick={onToggle}
          className="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700"
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          <svg
            viewBox="0 0 20 20"
            fill="currentColor"
            className="h-4 w-4"
          >
            {collapsed ? (
              <path
                fillRule="evenodd"
                d="M3 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                clipRule="evenodd"
              />
            ) : (
              <path
                fillRule="evenodd"
                d="M3 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 5a1 1 0 011-1h6a1 1 0 110 2H4a1 1 0 01-1-1zm0 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                clipRule="evenodd"
              />
            )}
          </svg>
        </button>
      </div>

      <nav className="flex-1 p-2">
        {navItems.map((item) => {
          const active = currentPage.startsWith(item.id);
          return (
            <button
              key={item.id}
              onClick={() => onNavigate(item.id)}
              className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                active
                  ? "bg-slate-100 text-slate-900"
                  : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
              }`}
            >
              <item.icon className="h-4 w-4 flex-shrink-0" />
              {!collapsed && <span>{item.label}</span>}
            </button>
          );
        })}
      </nav>

      {/* Organization switcher */}
      {org && (
        <div className="relative border-t border-slate-200 p-2" ref={switcherRef}>
          {/* Switcher dropdown — opens upward */}
          {showSwitcher && !collapsed && (
            <div className="absolute bottom-full left-2 right-2 mb-1 rounded-lg border border-slate-200 bg-white py-1 shadow-lg">
              <p className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400">
                Switch organization
              </p>
              {orgs.length === 0 && (
                <div className="flex justify-center py-3">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-slate-200 border-t-slate-600" />
                </div>
              )}
              {orgs.map((o) => (
                <button
                  key={o.id}
                  onClick={() => handleSwitch(o)}
                  className={`flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors ${
                    o.id === org.id
                      ? "bg-indigo-50 text-indigo-700"
                      : "text-slate-700 hover:bg-slate-50"
                  }`}
                >
                  <div className={`flex h-6 w-6 flex-shrink-0 items-center justify-center rounded text-xs font-bold ${
                    o.id === org.id
                      ? "bg-indigo-100 text-indigo-700"
                      : "bg-slate-100 text-slate-600"
                  }`}>
                    {(o.display_name || o.name).charAt(0).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-xs font-medium">
                      {o.display_name || o.name}
                    </p>
                    <p className="truncate text-[10px] text-slate-400">{o.slug}</p>
                  </div>
                  {o.id === org.id && (
                    <svg viewBox="0 0 20 20" fill="currentColor" className="ml-auto h-4 w-4 flex-shrink-0 text-indigo-600">
                      <path fillRule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clipRule="evenodd" />
                    </svg>
                  )}
                </button>
              ))}
            </div>
          )}

          <button
            onClick={handleOpenSwitcher}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 transition-colors hover:bg-indigo-50"
            title={collapsed ? (org.display_name || org.name) : `Switch organization`}
          >
            <div className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-md bg-indigo-100 text-xs font-bold text-indigo-700">
              {(org.display_name || org.name).charAt(0).toUpperCase()}
            </div>
            {!collapsed && (
              <>
                <div className="min-w-0 text-left">
                  <p className="truncate text-xs font-semibold text-slate-900">
                    {org.display_name || org.name}
                  </p>
                  <p className="truncate text-[10px] text-slate-400">
                    {org.slug}
                  </p>
                </div>
                <svg viewBox="0 0 20 20" fill="currentColor" className="ml-auto h-3.5 w-3.5 flex-shrink-0 text-slate-400">
                  <path fillRule="evenodd" d="M5.22 8.22a.75.75 0 011.06 0L10 11.94l3.72-3.72a.75.75 0 111.06 1.06l-4.25 4.25a.75.75 0 01-1.06 0L5.22 9.28a.75.75 0 010-1.06z" clipRule="evenodd" />
                </svg>
              </>
            )}
          </button>
        </div>
      )}
    </aside>
  );
}

function ChartIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 20 20" fill="currentColor" className={className}>
      <path d="M2 11a1 1 0 011-1h2a1 1 0 011 1v5a1 1 0 01-1 1H3a1 1 0 01-1-1v-5zm6-4a1 1 0 011-1h2a1 1 0 011 1v9a1 1 0 01-1 1H9a1 1 0 01-1-1V7zm6-3a1 1 0 011-1h2a1 1 0 011 1v12a1 1 0 01-1 1h-2a1 1 0 01-1-1V4z" />
    </svg>
  );
}

function UsersIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 20 20" fill="currentColor" className={className}>
      <path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zm8 0a3 3 0 11-6 0 3 3 0 016 0zm-4.07 11c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z" />
    </svg>
  );
}

function OrgIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 20 20" fill="currentColor" className={className}>
      <path
        fillRule="evenodd"
        d="M4 4a2 2 0 012-2h8a2 2 0 012 2v12a1 1 0 110 2h-3a1 1 0 01-1-1v-2a1 1 0 00-1-1H9a1 1 0 00-1 1v2a1 1 0 01-1 1H4a1 1 0 110-2V4zm3 1h2v2H7V5zm2 4H7v2h2V9zm2-4h2v2h-2V5zm2 4h-2v2h2V9z"
        clipRule="evenodd"
      />
    </svg>
  );
}

function KeyIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 20 20" fill="currentColor" className={className}>
      <path fillRule="evenodd" d="M8 7a5 5 0 113.61 4.804l-1.903 1.903A1 1 0 019 14H8v1a1 1 0 01-1 1H6v1a1 1 0 01-1 1H3a1 1 0 01-1-1v-2a1 1 0 01.293-.707L8.196 8.39A5.002 5.002 0 018 7zm5-3a.75.75 0 000 1.5A1.5 1.5 0 0114.5 7 .75.75 0 0016 7a3 3 0 00-3-3z" clipRule="evenodd" />
    </svg>
  );
}
