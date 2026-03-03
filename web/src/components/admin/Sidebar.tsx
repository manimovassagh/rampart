import { useCurrentOrg } from "../../hooks/useCurrentOrg";

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
];

export default function Sidebar({
  collapsed,
  onToggle,
  currentPage,
  onNavigate,
}: SidebarProps) {
  const { org } = useCurrentOrg();

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

      {/* Current organization badge */}
      {org && (
        <div className="border-t border-slate-200 p-2">
          <button
            onClick={() => onNavigate(`organizations/${org.id}`)}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 transition-colors hover:bg-indigo-50"
            title={`Managing: ${org.display_name || org.name}`}
          >
            <div className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-md bg-indigo-100 text-xs font-bold text-indigo-700">
              {(org.display_name || org.name).charAt(0).toUpperCase()}
            </div>
            {!collapsed && (
              <div className="min-w-0 text-left">
                <p className="truncate text-xs font-semibold text-slate-900">
                  {org.display_name || org.name}
                </p>
                <p className="truncate text-[10px] text-slate-400">
                  {org.slug}
                </p>
              </div>
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
