import { createElement, Fragment } from "react";
import type { ReactNode } from "react";
import { useAuth } from "./use-auth.js";

export interface ProtectedRouteProps {
  children: ReactNode;
  roles?: string[];
  fallback?: ReactNode;
  loadingFallback?: ReactNode;
}

export function ProtectedRoute({
  children,
  roles,
  fallback = null,
  loadingFallback = null,
}: ProtectedRouteProps) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return createElement(Fragment, null, loadingFallback);
  }

  if (!user) {
    return createElement(Fragment, null, fallback);
  }

  if (roles && roles.length > 0) {
    const userRoles = user.roles ?? [];
    const hasRole = roles.some((r) => userRoles.includes(r));
    if (!hasRole) {
      return createElement(Fragment, null, fallback);
    }
  }

  return createElement(Fragment, null, children);
}
