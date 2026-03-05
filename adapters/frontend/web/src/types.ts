/** Configuration for the Rampart client. */
export interface RampartClientConfig {
  /** Rampart server URL (e.g. "http://localhost:8080") */
  issuer: string;
  /** OAuth 2.0 client ID registered with the Rampart server. */
  clientId: string;
  /** OAuth 2.0 redirect URI — must exactly match a registered redirect URI. */
  redirectUri: string;
  /** OAuth 2.0 scopes (default: "openid"). */
  scope?: string;
  /** Called when tokens change (login, refresh, logout). Use for persistence. */
  onTokenChange?: (tokens: RampartTokens | null) => void;
}

/** Tokens returned by login and refresh. */
export interface RampartTokens {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

/** User profile from /me or login response. */
export interface RampartUser {
  id: string;
  org_id: string;
  preferred_username?: string;
  username?: string;
  email: string;
  email_verified: boolean;
  given_name?: string;
  family_name?: string;
  roles?: string[];
  enabled?: boolean;
  created_at?: string;
  updated_at?: string;
}

/** Rampart API error response. */
export interface RampartError {
  error: string;
  error_description: string;
  status: number;
}
