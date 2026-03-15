import { useAuth } from "@rampart-auth/react";
import { UserCard } from "../components/UserCard";
import { ApiTester } from "../components/ApiTester";

export function Dashboard() {
  const { user } = useAuth();

  if (!user) return null;

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-gray-900">Dashboard</h2>
      <UserCard user={user} />
      <ApiTester />
    </div>
  );
}
