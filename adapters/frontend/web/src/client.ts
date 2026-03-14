import type {
  RampartClientConfig,
  RampartTokens,
  RampartUser,
  RampartError,
} from "./types.js";

const STORAGE_PREFIX = "rampart_pkce_";

export class RampartClient {
  private issuer: string;
  private clientId: string;
  private redirectUri: string;
  private scope: string;
  private tokens: RampartTokens | null = null;
  private onTokenChange?: (tokens: RampartTokens | null) => void;

  constructor(config: RampartClientConfig) {
    this.issuer = config.issuer.replace(/\/+$/, "");
    this.clientId = config.clientId;
    this.redirectUri = config.redirectUri;
    this.scope = config.scope ?? "openid";
    this.onTokenChange = config.onTokenChange;
  }

  /**
   * Redirect the browser to the Rampart authorization endpoint.
   * Generates PKCE code_verifier + code_challenge and stores them in sessionStorage.
   */
  async loginWithRedirect(): Promise<void> {
    const codeVerifier = generateCodeVerifier();
    const codeChallenge = await generateCodeChallenge(codeVerifier);
    const state = generateState();

    // Store in sessionStorage (per-tab, cleared on tab close)
    sessionStorage.setItem(STORAGE_PREFIX + "code_verifier", codeVerifier);
    sessionStorage.setItem(STORAGE_PREFIX + "state", state);

    const params = new URLSearchParams({
      response_type: "code",
      client_id: this.clientId,
      redirect_uri: this.redirectUri,
      scope: this.scope,
      state,
      code_challenge: codeChallenge,
      code_challenge_method: "S256",
    });

    window.location.href = `${this.issuer}/oauth/authorize?${params.toString()}`;
  }

  /**
   * Handle the OAuth callback after redirect.
   * Extracts code + state from the URL, validates state, exchanges code for tokens.
   */
  async handleCallback(url?: string): Promise<RampartTokens> {
    const callbackUrl = new URL(url ?? window.location.href);
    const code = callbackUrl.searchParams.get("code");
    const state = callbackUrl.searchParams.get("state");
    const error = callbackUrl.searchParams.get("error");

    if (error) {
      const description = callbackUrl.searchParams.get("error_description") ?? error;
      throw { error, error_description: description, status: 0 } as RampartError;
    }

    if (!code || !state) {
      throw {
        error: "invalid_callback",
        error_description: "Missing code or state parameter in callback URL.",
        status: 0,
      } as RampartError;
    }

    // Validate state matches what we stored
    const storedState = sessionStorage.getItem(STORAGE_PREFIX + "state");
    if (!storedState || storedState !== state) {
      this.cleanupPKCEStorage();
      throw {
        error: "state_mismatch",
        error_description: "State parameter does not match. Possible CSRF attack.",
        status: 0,
      } as RampartError;
    }

    const codeVerifier = sessionStorage.getItem(STORAGE_PREFIX + "code_verifier");
    if (!codeVerifier) {
      this.cleanupPKCEStorage();
      throw {
        error: "missing_verifier",
        error_description: "Code verifier not found in session storage.",
        status: 0,
      } as RampartError;
    }

    // Exchange code for tokens
    const body = new URLSearchParams({
      grant_type: "authorization_code",
      code,
      client_id: this.clientId,
      redirect_uri: this.redirectUri,
      code_verifier: codeVerifier,
    });

    const res = await fetch(`${this.issuer}/oauth/token`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body.toString(),
    });

    this.cleanupPKCEStorage();

    if (!res.ok) {
      throw await this.parseError(res);
    }

    const data: RampartTokens = await res.json();
    this.setTokens(data);
    return data;
  }

  /** Refresh the access token using the stored refresh token. */
  async refresh(): Promise<RampartTokens> {
    if (!this.tokens?.refresh_token) {
      throw new Error("No refresh token available.");
    }

    const res = await fetch(`${this.issuer}/token/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: this.tokens.refresh_token }),
    });

    if (!res.ok) {
      this.setTokens(null);
      throw await this.parseError(res);
    }

    const data = await res.json();
    const updated: RampartTokens = {
      access_token: data.access_token,
      refresh_token: this.tokens.refresh_token,
      token_type: data.token_type,
      expires_in: data.expires_in,
    };
    this.setTokens(updated);

    return updated;
  }

  /** Logout — invalidates the refresh token on the server. */
  async logout(): Promise<void> {
    if (this.tokens?.refresh_token) {
      await fetch(`${this.issuer}/logout`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: this.tokens.refresh_token }),
      }).catch(() => {});
    }

    this.setTokens(null);
  }

  /** Fetch the current user profile from /me. */
  async getUser(): Promise<RampartUser> {
    const res = await this.authFetch(`${this.issuer}/me`);

    if (!res.ok) {
      throw await this.parseError(res);
    }

    return res.json();
  }

  /**
   * Fetch with automatic Authorization header.
   * On 401, attempts one token refresh then retries.
   */
  async authFetch(url: string, init?: RequestInit): Promise<Response> {
    const doFetch = () =>
      fetch(url, {
        ...init,
        headers: {
          ...init?.headers,
          Authorization: `Bearer ${this.tokens?.access_token}`,
        },
      });

    let res = await doFetch();

    if (res.status === 401 && this.tokens?.refresh_token) {
      try {
        await this.refresh();
        res = await doFetch();
      } catch {
        // refresh failed — return the original 401
      }
    }

    return res;
  }

  /** Get the current access token (or null). */
  getAccessToken(): string | null {
    return this.tokens?.access_token ?? null;
  }

  /** Get all current tokens (or null). */
  getTokens(): RampartTokens | null {
    return this.tokens ? { ...this.tokens } : null;
  }

  /** Check if the client has a non-expired access token. */
  isAuthenticated(): boolean {
    if (!this.tokens?.access_token) return false;
    try {
      const parts = this.tokens.access_token.split(".");
      if (parts.length !== 3) return false;
      const payload = JSON.parse(atob(parts[1]));
      if (typeof payload.exp !== "number") return false;
      return payload.exp * 1000 > Date.now();
    } catch {
      return false;
    }
  }

  /** Restore tokens from external storage (e.g. localStorage). */
  setTokens(tokens: RampartTokens | null): void {
    this.tokens = tokens;
    this.onTokenChange?.(tokens);
  }

  private cleanupPKCEStorage(): void {
    sessionStorage.removeItem(STORAGE_PREFIX + "code_verifier");
    sessionStorage.removeItem(STORAGE_PREFIX + "state");
  }

  private async parseError(res: Response): Promise<RampartError> {
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
}

/** Generate a random 64-character code verifier using Web Crypto. */
function generateCodeVerifier(): string {
  const array = new Uint8Array(48);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

/** Compute S256 code challenge: BASE64URL(SHA256(verifier)). */
async function generateCodeChallenge(verifier: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return base64UrlEncode(new Uint8Array(digest));
}

/** Generate a random state parameter for CSRF protection. */
function generateState(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

/** URL-safe base64 encoding without padding. */
function base64UrlEncode(buffer: Uint8Array): string {
  let binary = "";
  for (const byte of buffer) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}
