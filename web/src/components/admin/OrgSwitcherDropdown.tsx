import { useState, useEffect, useRef } from "react";
import { useCurrentOrg } from "../../hooks/useCurrentOrg";
import { listOrgs } from "../../api/organizations";
import type { OrgResponse } from "../../types";

interface OrgSwitcherDropdownProps {
  onNavigate: (page: string) => void;
}

export default function OrgSwitcherDropdown({
  onNavigate,
}: OrgSwitcherDropdownProps) {
  const { org, switchOrg } = useCurrentOrg();
  const [open, setOpen] = useState(false);
  const [orgs, setOrgs] = useState<OrgResponse[]>([]);
  const [loading, setLoading] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    if (open) {
      document.addEventListener("mousedown", handleClick);
    }
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  const handleOpen = () => {
    setOpen(!open);
    if (!open) {
      setLoading(true);
      listOrgs(1, 50)
        .then((data) => setOrgs(data.organizations))
        .catch(() => {})
        .finally(() => setLoading(false));
    }
  };

  const handleSwitch = async (target: OrgResponse) => {
    if (target.id === org?.id) {
      setOpen(false);
      return;
    }
    await switchOrg(target.id);
    setOpen(false);
    onNavigate("dashboard");
  };

  if (!org) return null;

  const orgName = org.display_name || org.name;

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={handleOpen}
        className="flex items-center gap-1.5 rounded-lg bg-indigo-50 px-2.5 py-1 text-xs font-semibold text-indigo-700 transition-colors hover:bg-indigo-100"
      >
        <span>{orgName}</span>
        <svg
          viewBox="0 0 20 20"
          fill="currentColor"
          className={`h-3.5 w-3.5 transition-transform ${open ? "rotate-180" : ""}`}
        >
          <path
            fillRule="evenodd"
            d="M5.22 8.22a.75.75 0 011.06 0L10 11.94l3.72-3.72a.75.75 0 111.06 1.06l-4.25 4.25a.75.75 0 01-1.06 0L5.22 9.28a.75.75 0 010-1.06z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      {open && (
        <div className="absolute left-0 top-full z-50 mt-2 w-72 rounded-xl border border-slate-200 bg-white py-1 shadow-xl">
          <p className="px-3 py-2 text-[10px] font-semibold uppercase tracking-wider text-slate-400">
            Switch organization
          </p>

          {loading && (
            <div className="flex justify-center py-4">
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-slate-200 border-t-slate-600" />
            </div>
          )}

          {!loading &&
            orgs.map((o) => {
              const active = o.id === org.id;
              return (
                <button
                  key={o.id}
                  onClick={() => handleSwitch(o)}
                  className={`flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors ${
                    active
                      ? "bg-indigo-50 text-indigo-700"
                      : "text-slate-700 hover:bg-slate-50"
                  }`}
                >
                  <div
                    className={`flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-md text-xs font-bold ${
                      active
                        ? "bg-indigo-100 text-indigo-700"
                        : "bg-slate-100 text-slate-600"
                    }`}
                  >
                    {(o.display_name || o.name).charAt(0).toUpperCase()}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">
                      {o.display_name || o.name}
                    </p>
                    <p className="truncate text-[11px] text-slate-400">
                      {o.slug}
                    </p>
                  </div>
                  {active && (
                    <svg
                      viewBox="0 0 20 20"
                      fill="currentColor"
                      className="h-4 w-4 flex-shrink-0 text-indigo-600"
                    >
                      <path
                        fillRule="evenodd"
                        d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z"
                        clipRule="evenodd"
                      />
                    </svg>
                  )}
                </button>
              );
            })}

          {!loading && orgs.length > 0 && (
            <div className="mx-3 my-1 border-t border-slate-100" />
          )}

          {!loading && (
            <button
              onClick={() => {
                setOpen(false);
                onNavigate("organizations");
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-xs font-medium text-slate-500 transition-colors hover:bg-slate-50 hover:text-slate-700"
            >
              <svg
                viewBox="0 0 20 20"
                fill="currentColor"
                className="h-3.5 w-3.5"
              >
                <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" />
              </svg>
              Manage organizations
            </button>
          )}
        </div>
      )}
    </div>
  );
}
