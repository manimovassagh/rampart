import { useCallback, useContext } from "react";
import AsyncStorage from "@react-native-async-storage/async-storage";
import { Linking } from "react-native";
import type { RampartTokens, RampartUser, RampartError } from "@rampart-auth/web";
import { RampartNativeContext } from "./RampartProvider.js";
import type { UseAuthReturn } from "./types.js";

const PKCE_VERIFIER_KEY = "rampart_pkce_code_verifier";
const PKCE_STATE_KEY = "rampart_pkce_state";

/**
 * useAuth provides authentication methods for React Native apps.
 *
 * Must be used inside a `<RampartProvider>`.
 *
 * Key differences from the web adapter:
 * - Uses AsyncStorage instead of localStorage / sessionStorage
 * - Uses Linking.openURL (system browser) instead of window.location
 * - handleCallback requires the full deep link URL as a parameter
 */
export function useAuth(): UseAuthReturn {
  const ctx = useContext(RampartNativeContext);
  if (!ctx) {
    throw new Error("useAuth must be used within a <RampartProvider>");
  }

  const { issuer, clientId, redirectUri, scope, user, setUser, tokens, setTokens, isLoading } =
    ctx;

  /**
   * Open the system browser to start the OAuth PKCE authorization flow.
   * Generates code_verifier and state, stores them in AsyncStorage,
   * then opens the authorization URL in the default browser.
   */
  const loginWithRedirect = useCallback(async () => {
    const codeVerifier = generateCodeVerifier();
    const codeChallenge = await generateCodeChallenge(codeVerifier);
    const state = generateState();

    // Store PKCE params in AsyncStorage (no sessionStorage on native)
    await AsyncStorage.setItem(PKCE_VERIFIER_KEY, codeVerifier);
    await AsyncStorage.setItem(PKCE_STATE_KEY, state);

    const params = new URLSearchParams({
      response_type: "code",
      client_id: clientId,
      redirect_uri: redirectUri,
      scope,
      state,
      code_challenge: codeChallenge,
      code_challenge_method: "S256",
    });

    const authUrl = `${issuer}/oauth/authorize?${params.toString()}`;
    await Linking.openURL(authUrl);
  }, [issuer, clientId, redirectUri, scope]);

  /**
   * Handle the OAuth callback after the system browser redirects back.
   * Extracts code + state from the deep link URL, validates state,
   * and exchanges the authorization code for tokens.
   *
   * @param url - The full callback deep link URL (e.g. "myapp://auth/callback?code=...&state=...")
   */
  const handleCallback = useCallback(
    async (url: string) => {
      const callbackUrl = new URL(url);
      const code = callbackUrl.searchParams.get("code");
      const state = callbackUrl.searchParams.get("state");
      const error = callbackUrl.searchParams.get("error");

      if (error) {
        const description =
          callbackUrl.searchParams.get("error_description") ?? error;
        throw {
          error,
          error_description: description,
          status: 0,
        } as RampartError;
      }

      if (!code || !state) {
        throw {
          error: "invalid_callback",
          error_description:
            "Missing code or state parameter in callback URL.",
          status: 0,
        } as RampartError;
      }

      // Validate state
      const storedState = await AsyncStorage.getItem(PKCE_STATE_KEY);
      if (!storedState || storedState !== state) {
        await cleanupPKCEStorage();
        throw {
          error: "state_mismatch",
          error_description:
            "State parameter does not match. Possible CSRF attack.",
          status: 0,
        } as RampartError;
      }

      const codeVerifier = await AsyncStorage.getItem(PKCE_VERIFIER_KEY);
      if (!codeVerifier) {
        await cleanupPKCEStorage();
        throw {
          error: "missing_verifier",
          error_description:
            "Code verifier not found in AsyncStorage.",
          status: 0,
        } as RampartError;
      }

      // Exchange code for tokens
      const body = new URLSearchParams({
        grant_type: "authorization_code",
        code,
        client_id: clientId,
        redirect_uri: redirectUri,
        code_verifier: codeVerifier,
      });

      const res = await fetch(`${issuer}/oauth/token`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });

      await cleanupPKCEStorage();

      if (!res.ok) {
        throw await parseError(res);
      }

      const data: RampartTokens = await res.json();
      setTokens(data);

      // Fetch user profile
      const meRes = await fetch(`${issuer}/me`, {
        headers: { Authorization: `Bearer ${data.access_token}` },
      });

      if (meRes.ok) {
        const me: RampartUser = await meRes.json();
        setUser(me);
      }
    },
    [issuer, clientId, redirectUri, setTokens, setUser]
  );

  /**
   * Logout: invalidate the refresh token on the server and clear local state.
   */
  const logout = useCallback(async () => {
    if (tokens?.refresh_token) {
      await fetch(`${issuer}/logout`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: tokens.refresh_token }),
      }).catch(() => {});
    }

    setTokens(null);
    setUser(null);
  }, [issuer, tokens, setTokens, setUser]);

  /**
   * Get the current access token (or null).
   */
  const getAccessToken = useCallback((): string | null => {
    return tokens?.access_token ?? null;
  }, [tokens]);

  /**
   * Fetch with automatic Authorization header.
   * On 401, attempts one token refresh then retries.
   */
  const authFetch = useCallback(
    async (url: string, init?: RequestInit): Promise<Response> => {
      const doFetch = (accessToken: string) =>
        fetch(url, {
          ...init,
          headers: {
            ...init?.headers,
            Authorization: `Bearer ${accessToken}`,
          },
        });

      if (!tokens?.access_token) {
        return fetch(url, init);
      }

      let res = await doFetch(tokens.access_token);

      if (res.status === 401 && tokens.refresh_token) {
        try {
          // Attempt token refresh
          const refreshRes = await fetch(`${issuer}/token/refresh`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ refresh_token: tokens.refresh_token }),
          });

          if (refreshRes.ok) {
            const refreshData = await refreshRes.json();
            const updated: RampartTokens = {
              access_token: refreshData.access_token,
              refresh_token: tokens.refresh_token,
              token_type: refreshData.token_type,
              expires_in: refreshData.expires_in,
            };
            setTokens(updated);
            res = await doFetch(updated.access_token);
          } else {
            setTokens(null);
          }
        } catch {
          // Refresh failed — return the original 401
        }
      }

      return res;
    },
    [issuer, tokens, setTokens]
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

// ---------------------------------------------------------------------------
// PKCE helpers (same algorithm as @rampart-auth/web but using react-native
// compatible crypto — expo-crypto or react-native-get-random-values polyfill
// provides the Web Crypto API on native).
// ---------------------------------------------------------------------------

function generateCodeVerifier(): string {
  const array = new Uint8Array(48);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

async function generateCodeChallenge(verifier: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return base64UrlEncode(new Uint8Array(digest));
}

function generateState(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

function base64UrlEncode(buffer: Uint8Array): string {
  let binary = "";
  for (const byte of buffer) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

async function cleanupPKCEStorage(): Promise<void> {
  await AsyncStorage.multiRemove([PKCE_VERIFIER_KEY, PKCE_STATE_KEY]).catch(
    () => {}
  );
}

async function parseError(res: Response): Promise<RampartError> {
  try {
    return await res.json();
  } catch {
    return {
      error: "unknown_error",
      error_description: `HTTP ${res.status}: ${res.statusText}`,
      status: res.status,
    };
  }
}
