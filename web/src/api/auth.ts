import type {
  LoginRequest,
  LoginResponse,
  RefreshResponse,
  MeResponse,
  ApiErrorResponse,
} from "../types";

const TOKEN_KEY = "rampart_access_token";
const REFRESH_KEY = "rampart_refresh_token";

export function getStoredTokens() {
  return {
    accessToken: localStorage.getItem(TOKEN_KEY),
    refreshToken: localStorage.getItem(REFRESH_KEY),
  };
}

export function storeTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_KEY, refreshToken);
}

export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

export type LoginResult =
  | { ok: true; data: LoginResponse }
  | { ok: false; error: ApiErrorResponse };

export async function login(data: LoginRequest): Promise<LoginResult> {
  const res = await fetch("/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  let body: unknown;
  try {
    body = await res.json();
  } catch {
    return {
      ok: false,
      error: {
        error: "network_error",
        error_description: "Unable to reach the server. Please try again.",
        status: res.status,
      },
    };
  }

  if (res.ok) {
    const data = body as LoginResponse;
    storeTokens(data.access_token, data.refresh_token);
    return { ok: true, data };
  }

  return { ok: false, error: body as ApiErrorResponse };
}

export async function refreshToken(): Promise<RefreshResponse | null> {
  const { refreshToken } = getStoredTokens();
  if (!refreshToken) return null;

  const res = await fetch("/token/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!res.ok) {
    clearTokens();
    return null;
  }

  const data = (await res.json()) as RefreshResponse;
  localStorage.setItem(TOKEN_KEY, data.access_token);
  return data;
}

export async function logout(): Promise<void> {
  const { refreshToken } = getStoredTokens();
  if (refreshToken) {
    try {
      await fetch("/logout", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
    } catch {
      // Best-effort logout
    }
  }
  clearTokens();
}

export async function getMe(): Promise<MeResponse | null> {
  const { accessToken } = getStoredTokens();
  if (!accessToken) return null;

  const res = await fetch("/me", {
    headers: { Authorization: `Bearer ${accessToken}` },
  });

  if (res.status === 401) {
    // Try refreshing the token
    const refreshed = await refreshToken();
    if (!refreshed) return null;

    const retryRes = await fetch("/me", {
      headers: { Authorization: `Bearer ${refreshed.access_token}` },
    });
    if (!retryRes.ok) return null;
    return (await retryRes.json()) as MeResponse;
  }

  if (!res.ok) return null;
  return (await res.json()) as MeResponse;
}
