import { getStoredTokens, refreshToken } from "./auth";
import type {
  DashboardStats,
  ListUsersResponse,
  AdminUserResponse,
  CreateUserRequest,
  UpdateUserRequest,
  ResetPasswordRequest,
  SessionResponse,
} from "../types";

async function authFetch(url: string, init?: RequestInit): Promise<Response> {
  const { accessToken } = getStoredTokens();
  const headers: Record<string, string> = {
    ...((init?.headers as Record<string, string>) ?? {}),
    Authorization: `Bearer ${accessToken}`,
  };

  let res = await fetch(url, { ...init, headers });

  if (res.status === 401) {
    const refreshed = await refreshToken();
    if (!refreshed) throw new Error("Session expired");
    headers.Authorization = `Bearer ${refreshed.access_token}`;
    res = await fetch(url, { ...init, headers });
  }

  return res;
}

async function authJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await authFetch(url, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { error_description?: string }).error_description ??
        `Request failed: ${res.status}`,
    );
  }
  return (await res.json()) as T;
}

export async function getStats(): Promise<DashboardStats> {
  return authJSON<DashboardStats>("/api/v1/admin/stats");
}

export async function listUsers(
  page = 1,
  limit = 20,
  search = "",
  status = "",
): Promise<ListUsersResponse> {
  const params = new URLSearchParams({
    page: String(page),
    limit: String(limit),
  });
  if (search) params.set("search", search);
  if (status) params.set("status", status);
  return authJSON<ListUsersResponse>(`/api/v1/admin/users?${params}`);
}

export async function getUser(id: string): Promise<AdminUserResponse> {
  return authJSON<AdminUserResponse>(`/api/v1/admin/users/${id}`);
}

export async function createUser(
  data: CreateUserRequest,
): Promise<AdminUserResponse> {
  return authJSON<AdminUserResponse>("/api/v1/admin/users", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateUser(
  id: string,
  data: UpdateUserRequest,
): Promise<AdminUserResponse> {
  return authJSON<AdminUserResponse>(`/api/v1/admin/users/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteUser(id: string): Promise<void> {
  const res = await authFetch(`/api/v1/admin/users/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { error_description?: string }).error_description ??
        `Delete failed: ${res.status}`,
    );
  }
}

export async function resetPassword(
  id: string,
  data: ResetPasswordRequest,
): Promise<void> {
  const res = await authFetch(`/api/v1/admin/users/${id}/reset-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { error_description?: string }).error_description ??
        `Reset failed: ${res.status}`,
    );
  }
}

export async function listSessions(id: string): Promise<SessionResponse[]> {
  return authJSON<SessionResponse[]>(`/api/v1/admin/users/${id}/sessions`);
}

export async function revokeSessions(id: string): Promise<void> {
  const res = await authFetch(`/api/v1/admin/users/${id}/sessions`, {
    method: "DELETE",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { error_description?: string }).error_description ??
        `Revoke failed: ${res.status}`,
    );
  }
}
