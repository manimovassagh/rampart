import { useCallback, useContext } from "react";
import type { RampartUser } from "@rampart/web";
import { RampartContext } from "./context.js";

export interface UseAuthReturn {
  user: RampartUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  loginWithRedirect: () => Promise<void>;
  handleCallback: (url?: string) => Promise<void>;
  logout: () => Promise<void>;
  getAccessToken: () => string | null;
  authFetch: (url: string, init?: RequestInit) => Promise<Response>;
}

export function useAuth(): UseAuthReturn {
  const ctx = useContext(RampartContext);
  if (!ctx) {
    throw new Error("useAuth must be used within a <RampartProvider>");
  }

  const { client, user, setUser, isLoading } = ctx;

  const loginWithRedirect = useCallback(async () => {
    await client.loginWithRedirect();
  }, [client]);

  const handleCallback = useCallback(
    async (url?: string) => {
      await client.handleCallback(url);
      const me = await client.getUser();
      setUser(me);
    },
    [client, setUser]
  );

  const logout = useCallback(async () => {
    await client.logout();
    setUser(null);
  }, [client, setUser]);

  const getAccessToken = useCallback(() => {
    return client.getAccessToken();
  }, [client]);

  const authFetch = useCallback(
    (url: string, init?: RequestInit) => {
      return client.authFetch(url, init);
    },
    [client]
  );

  return {
    user,
    isAuthenticated: user !== null,
    isLoading,
    loginWithRedirect,
    handleCallback,
    logout,
    getAccessToken,
    authFetch,
  };
}
