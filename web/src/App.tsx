import { useState, useEffect, useCallback } from "react";
import LoginForm from "./components/LoginForm";
import RegistrationForm from "./components/RegistrationForm";
import Dashboard from "./components/Dashboard";
import { getStoredTokens } from "./api/auth";

type Route = "login" | "register" | "dashboard";

function getRoute(): Route {
  const hash = window.location.hash.replace("#/", "");
  if (hash === "register") return "register";
  if (hash === "dashboard") return "dashboard";
  return "login";
}

export default function App() {
  const [route, setRoute] = useState<Route>(getRoute);

  useEffect(() => {
    function onHashChange() {
      setRoute(getRoute());
    }
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  // If user has tokens, redirect to dashboard
  useEffect(() => {
    if (route === "login" || route === "register") {
      const { accessToken } = getStoredTokens();
      if (accessToken) {
        navigate("dashboard");
      }
    }
  }, [route]);

  function navigate(r: Route) {
    window.location.hash = `#/${r}`;
  }

  const handleLogout = useCallback(() => {
    navigate("login");
  }, []);

  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-br from-slate-50 to-slate-100">
      {/* Header bar */}
      <header className="border-b border-slate-200 bg-white px-6 py-4 shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-5 w-5 text-white"
            >
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
          </div>
          <span className="text-xl font-bold tracking-tight text-slate-900">
            Rampart
          </span>
        </div>
      </header>

      {/* Main content */}
      <main className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="w-full max-w-md">
          {route === "login" && (
            <LoginForm
              onSuccess={() => navigate("dashboard")}
              onNavigateRegister={() => navigate("register")}
            />
          )}
          {route === "register" && (
            <RegistrationForm onNavigateLogin={() => navigate("login")} />
          )}
          {route === "dashboard" && (
            <Dashboard onLogout={handleLogout} />
          )}
          <p className="mt-6 text-center text-xs text-slate-400">
            Powered by Rampart Identity Server
          </p>
        </div>
      </main>
    </div>
  );
}
