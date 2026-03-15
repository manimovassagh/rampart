import React, { createContext, useEffect, useRef, useState } from "react";
import AsyncStorage from "@react-native-async-storage/async-storage";
import type { RampartTokens, RampartUser } from "@rampart-auth/web";
import type {
  RampartNativeContextValue,
  RampartNativeProviderProps,
} from "./types.js";

const STORAGE_KEY = "rampart_tokens";

export const RampartNativeContext =
  createContext<RampartNativeContextValue | null>(null);

/**
 * RampartProvider wraps your React Native app and manages authentication state.
 *
 * It persists tokens to AsyncStorage (instead of localStorage) and provides
 * context for the useAuth hook.
 *
 * ```tsx
 * <RampartProvider
 *   issuer="https://auth.example.com"
 *   clientId="my-mobile-app"
 *   redirectUri="myapp://auth/callback"
 * >
 *   <App />
 * </RampartProvider>
 * ```
 */
export function RampartProvider({
  issuer,
  clientId,
  redirectUri,
  scope,
  persist = true,
  children,
}: RampartNativeProviderProps) {
  const [user, setUser] = useState<RampartUser | null>(null);
  const [tokens, setTokensState] = useState<RampartTokens | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const normalizedIssuer = useRef(issuer.replace(/\/+$/, "")).current;
  const resolvedScope = useRef(scope ?? "openid").current;

  const setTokens = (newTokens: RampartTokens | null) => {
    setTokensState(newTokens);
    if (persist) {
      if (newTokens) {
        AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(newTokens)).catch(
          () => {}
        );
      } else {
        AsyncStorage.removeItem(STORAGE_KEY).catch(() => {});
      }
    }
  };

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      if (persist) {
        try {
          const stored = await AsyncStorage.getItem(STORAGE_KEY);
          if (stored && !cancelled) {
            const parsed: RampartTokens = JSON.parse(stored);
            setTokensState(parsed);

            // Attempt to fetch the user profile
            const res = await fetch(`${normalizedIssuer}/me`, {
              headers: {
                Authorization: `Bearer ${parsed.access_token}`,
              },
            });

            if (res.ok && !cancelled) {
              const me: RampartUser = await res.json();
              setUser(me);
            } else if (!cancelled) {
              // Token may be expired; keep tokens for potential refresh
            }
          }
        } catch {
          // Stored tokens are invalid — clear them
          if (!cancelled) {
            setTokensState(null);
            await AsyncStorage.removeItem(STORAGE_KEY).catch(() => {});
          }
        }
      }
      if (!cancelled) setIsLoading(false);
    }

    bootstrap();
    return () => {
      cancelled = true;
    };
  }, [normalizedIssuer, persist]);

  return (
    <RampartNativeContext.Provider
      value={{
        issuer: normalizedIssuer,
        clientId,
        redirectUri,
        scope: resolvedScope,
        user,
        setUser,
        tokens,
        setTokens,
        isLoading,
      }}
    >
      {children}
    </RampartNativeContext.Provider>
  );
}
