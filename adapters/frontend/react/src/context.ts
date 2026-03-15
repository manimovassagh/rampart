import {
  createContext,
  createElement,
  useEffect,
  useRef,
  useState,
} from "react";
import type { ReactNode } from "react";
import { RampartClient } from "@rampart-auth/web";
import type { RampartUser, RampartTokens } from "@rampart-auth/web";

export interface RampartContextValue {
  client: RampartClient;
  user: RampartUser | null;
  setUser: (user: RampartUser | null) => void;
  isLoading: boolean;
}

export const RampartContext = createContext<RampartContextValue | null>(null);

export interface RampartProviderProps {
  issuer: string;
  clientId: string;
  redirectUri: string;
  scope?: string;
  persist?: boolean;
  children: ReactNode;
}

const STORAGE_KEY = "rampart_tokens";

export function RampartProvider({
  issuer,
  clientId,
  redirectUri,
  scope,
  // WARNING: Enabling persist stores tokens in localStorage, which is vulnerable
  // to XSS attacks. Only enable this if you understand the security implications.
  persist = false,
  children,
}: RampartProviderProps) {
  const [user, setUser] = useState<RampartUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const clientRef = useRef<RampartClient | null>(null);
  if (!clientRef.current) {
    clientRef.current = new RampartClient({
      issuer,
      clientId,
      redirectUri,
      scope,
      onTokenChange: persist
        ? (tokens: RampartTokens | null) => {
            if (tokens) {
              window.localStorage.setItem(STORAGE_KEY, JSON.stringify(tokens));
            } else {
              window.localStorage.removeItem(STORAGE_KEY);
            }
          }
        : undefined,
    });
  }

  const client = clientRef.current;

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      if (persist) {
        try {
          const stored = window.localStorage.getItem(STORAGE_KEY);
          if (stored) {
            client.setTokens(JSON.parse(stored));
            const me = await client.getUser();
            if (!cancelled) setUser(me);
          }
        } catch {
          client.setTokens(null);
        }
      }
      if (!cancelled) setIsLoading(false);
    }

    bootstrap();
    return () => {
      cancelled = true;
    };
  }, [client, persist]);

  return createElement(
    RampartContext.Provider,
    { value: { client, user, setUser, isLoading } },
    children
  );
}
