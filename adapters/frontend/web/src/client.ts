import type {
  RampartClientConfig,
  RampartTokens,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RampartUser,
  RampartError,
} from "./types.js";

export class RampartClient {
  private issuer: string;
  private tokens: RampartTokens | null = null;
  private onTokenChange?: (tokens: RampartTokens | null) => void;

  constructor(config: RampartClientConfig) {
    this.issuer = config.issuer.replace(/\/+$/, "");
    this.onTokenChange = config.onTokenChange;
  }

  /** Register a new user. Returns the created user profile. */
  async register(req: RegisterRequest): Promise<RampartUser> {
    const res = await fetch(`${this.issuer}/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      throw await this.parseError(res);
    }

    return res.json();
  }

  /** Login and store tokens. Returns login response with user profile. */
  async login(req: LoginRequest): Promise<LoginResponse> {
    const res = await fetch(`${this.issuer}/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      throw await this.parseError(res);
    }

    const data: LoginResponse = await res.json();
    this.setTokens({
      access_token: data.access_token,
      refresh_token: data.refresh_token,
      token_type: data.token_type,
      expires_in: data.expires_in,
    });

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

  /** Check if the client has tokens (does NOT verify expiry). */
  isAuthenticated(): boolean {
    return this.tokens?.access_token != null;
  }

  /** Restore tokens from external storage (e.g. localStorage). */
  setTokens(tokens: RampartTokens | null): void {
    this.tokens = tokens;
    this.onTokenChange?.(tokens);
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
