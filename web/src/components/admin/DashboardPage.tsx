import { useState, useEffect } from "react";
import { getStats } from "../../api/admin";
import { useCurrentOrg } from "../../hooks/useCurrentOrg";
import type { DashboardStats } from "../../types";

interface DashboardPageProps {
  onNavigate: (page: string) => void;
}

export default function DashboardPage({ onNavigate }: DashboardPageProps) {
  const { org } = useCurrentOrg();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getStats()
      .then(setStats)
      .catch(() => setStats(null))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-slate-200 border-t-slate-900" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-5xl">
      <h1 className="text-2xl font-bold text-slate-900">Dashboard</h1>
      <p className="mt-1 text-sm text-slate-500">
        {org
          ? `Managing ${org.display_name || org.name}`
          : "Overview of your identity server"}
      </p>

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Total Users"
          value={stats?.total_users ?? 0}
          color="bg-blue-50 text-blue-700"
        />
        <StatCard
          label="Active Sessions"
          value={stats?.active_sessions ?? 0}
          color="bg-emerald-50 text-emerald-700"
        />
        <StatCard
          label="New Users (7d)"
          value={stats?.recent_users ?? 0}
          color="bg-purple-50 text-purple-700"
        />
        <StatCard
          label="Organizations"
          value={stats?.total_organizations ?? 0}
          color="bg-amber-50 text-amber-700"
        />
      </div>

      <div className="mt-8">
        <h2 className="text-lg font-semibold text-slate-900">Quick Actions</h2>
        <div className="mt-3 flex flex-wrap gap-3">
          <button
            onClick={() => onNavigate("users")}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
          >
            View all users
          </button>
          <button
            onClick={() => onNavigate("users/new")}
            className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800"
          >
            Create user
          </button>
        </div>
      </div>
    </div>
  );
}

function StatCard({
  label,
  value,
  color,
}: {
  label: string;
  value: number;
  color: string;
}) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <p className="text-sm font-medium text-slate-500">{label}</p>
      <p className={`mt-2 text-3xl font-bold ${color} inline-block rounded-lg px-2 py-1`}>
        {value}
      </p>
    </div>
  );
}
