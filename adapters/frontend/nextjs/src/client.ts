"use client";

import { useCallback, useEffect, useState } from "react";
import type { RampartClaims } from "./types.js";

// Re-export everything from @rampart/react for convenience
export {
  RampartProvider,
  RampartContext,
  useAuth,
  ProtectedRoute,
} from "@rampart/react";

export type {
  RampartProviderProps,
  RampartContextValue,
  UseAuthReturn,
  ProtectedRouteProps,
} from "@rampart/react";

const DEFAULT_SESSION_ENDPOINT = "/api/auth/session";

interface UseRampartSessionOptions {
  /** API endpoint that returns the session. Default: "/api/auth/session" */
  sessionEndpoint?: string;
}

interface RampartSession {
  claims: RampartClaims | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  refresh: () => Promise<void>;
}

/**
 * Client-side hook that reads the current Rampart session from a server endpoint.
 *
 * Create a Route Handler at /api/auth/session that uses getServerAuth()
 * to return the current claims:
 *
 * ```ts
 * // app/api/auth/session/route.ts
 * import { cookies } from "next/headers";
 * import { getServerAuth } from "@rampart/nextjs/server";
 *
 * export async function GET() {
 *   const auth = await getServerAuth(await cookies(), process.env.RAMPART_ISSUER!);
 *   if (!auth) return Response.json({ claims: null });
 *   return Response.json({ claims: auth.claims });
 * }
 * ```
 */
export function useRampartSession(
  options?: UseRampartSessionOptions
): RampartSession {
  const endpoint = options?.sessionEndpoint ?? DEFAULT_SESSION_ENDPOINT;
  const [claims, setClaims] = useState<RampartClaims | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchSession = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await fetch(endpoint, { credentials: "same-origin" });
      if (res.ok) {
        const data = await res.json();
        setClaims(data.claims ?? null);
      } else {
        setClaims(null);
      }
    } catch {
      setClaims(null);
    } finally {
      setIsLoading(false);
    }
  }, [endpoint]);

  useEffect(() => {
    fetchSession();
  }, [fetchSession]);

  return {
    claims,
    isLoading,
    isAuthenticated: claims !== null,
    refresh: fetchSession,
  };
}
