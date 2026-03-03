import { useState, useEffect, useCallback } from "react";
import LoginForm from "./components/LoginForm";
import RegistrationForm from "./components/RegistrationForm";
import Layout from "./components/admin/Layout";
import DashboardPage from "./components/admin/DashboardPage";
import UsersPage from "./components/admin/UsersPage";
import UserDetailPage from "./components/admin/UserDetailPage";
import UserCreatePage from "./components/admin/UserCreatePage";
import OrganizationsPage from "./components/admin/OrganizationsPage";
import OrganizationDetailPage from "./components/admin/OrganizationDetailPage";
import OrganizationCreatePage from "./components/admin/OrganizationCreatePage";
import ToastContainer from "./components/admin/Toast";
import { getStoredTokens } from "./api/auth";

type Route = "login" | "register" | "admin";

function getRoute(): Route {
  const hash = window.location.hash.replace("#/", "");
  if (hash === "register") return "register";
  if (hash.startsWith("admin")) return "admin";
  // Legacy: redirect old "dashboard" hash to admin
  if (hash === "dashboard") return "admin";
  return "login";
}

function getAdminPage(): string {
  const hash = window.location.hash.replace("#/", "");
  if (hash.startsWith("admin/")) return hash.slice(6);
  return "dashboard";
}

export default function App() {
  const [route, setRoute] = useState<Route>(getRoute);
  const [adminPage, setAdminPage] = useState(getAdminPage);

  useEffect(() => {
    function onHashChange() {
      setRoute(getRoute());
      setAdminPage(getAdminPage());
    }
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  // If user has tokens, redirect to admin
  useEffect(() => {
    if (route === "login" || route === "register") {
      const { accessToken } = getStoredTokens();
      if (accessToken) {
        navigate("admin/dashboard");
      }
    }
  }, [route]);

  function navigate(path: string) {
    window.location.hash = `#/${path}`;
  }

  const handleLogout = useCallback(() => {
    navigate("login");
  }, []);

  const handleAdminNavigate = useCallback((page: string) => {
    navigate(`admin/${page}`);
  }, []);

  // Auth pages (login/register)
  if (route !== "admin") {
    return (
      <div className="flex min-h-screen flex-col bg-gradient-to-br from-slate-50 to-slate-100">
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

        <main className="flex flex-1 items-center justify-center px-4 py-12">
          <div className="w-full max-w-md">
            {route === "login" && (
              <LoginForm
                onSuccess={() => navigate("admin/dashboard")}
                onNavigateRegister={() => navigate("register")}
              />
            )}
            {route === "register" && (
              <RegistrationForm onNavigateLogin={() => navigate("login")} />
            )}
            <p className="mt-6 text-center text-xs text-slate-400">
              Powered by Rampart Identity Server
            </p>
          </div>
        </main>
        <ToastContainer />
      </div>
    );
  }

  // Admin console
  return (
    <>
      <Layout
        currentPage={adminPage}
        onNavigate={handleAdminNavigate}
        onLogout={handleLogout}
      >
        {adminPage === "dashboard" && (
          <DashboardPage onNavigate={handleAdminNavigate} />
        )}
        {adminPage === "users" && (
          <UsersPage onNavigate={handleAdminNavigate} />
        )}
        {adminPage === "users/new" && (
          <UserCreatePage onNavigate={handleAdminNavigate} />
        )}
        {adminPage.startsWith("users/") &&
          adminPage !== "users/new" && (
            <UserDetailPage
              userId={adminPage.slice(6)}
              onNavigate={handleAdminNavigate}
            />
          )}
        {adminPage === "organizations" && (
          <OrganizationsPage onNavigate={handleAdminNavigate} />
        )}
        {adminPage === "organizations/new" && (
          <OrganizationCreatePage onNavigate={handleAdminNavigate} />
        )}
        {adminPage.startsWith("organizations/") &&
          adminPage !== "organizations/new" && (
            <OrganizationDetailPage
              orgId={adminPage.slice(14)}
              onNavigate={handleAdminNavigate}
            />
          )}
      </Layout>
      <ToastContainer />
    </>
  );
}
