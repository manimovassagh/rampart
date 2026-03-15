import type { RampartTokens, RampartUser } from "@rampart-auth/web";

/** Configuration for the React Native Rampart provider. */
export interface RampartNativeProviderProps {
  /** Rampart server URL (e.g. "https://auth.example.com") */
  issuer: string;
  /** OAuth 2.0 client ID registered with the Rampart server. */
  clientId: string;
  /** Deep link URI that handles the OAuth callback (e.g. "myapp://auth/callback"). */
  redirectUri: string;
  /** OAuth 2.0 scopes (default: "openid"). */
  scope?: string;
  /** Persist tokens to AsyncStorage (default: true). */
  persist?: boolean;
  /** React children. */
  children: React.ReactNode;
}

/** Shape of the internal context value. */
export interface RampartNativeContextValue {
  issuer: string;
  clientId: string;
  redirectUri: string;
  scope: string;
  user: RampartUser | null;
  setUser: (user: RampartUser | null) => void;
  tokens: RampartTokens | null;
  setTokens: (tokens: RampartTokens | null) => void;
  isLoading: boolean;
}

/** Return type of the useAuth hook. */
export interface UseAuthReturn {
  /** Current authenticated user, or null. */
  user: RampartUser | null;
  /** Whether a user is currently authenticated. */
  isAuthenticated: boolean;
  /** Whether the provider is still loading persisted tokens. */
  isLoading: boolean;
  /** Open the system browser to start the OAuth PKCE login flow. */
  loginWithRedirect: () => Promise<void>;
  /** Exchange the authorization code from the callback URL for tokens. */
  handleCallback: (url: string) => Promise<void>;
  /** Logout and clear all tokens. */
  logout: () => Promise<void>;
  /** Get the current access token (or null). */
  getAccessToken: () => string | null;
  /** Fetch with automatic Authorization header and token refresh on 401. */
  authFetch: (url: string, init?: RequestInit) => Promise<Response>;
}
