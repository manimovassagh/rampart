/** Configuration for the Rampart client. */
export interface RampartClientConfig {
  /** Rampart server URL (e.g. "http://localhost:8080") */
  issuer: string;
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

/** Login request body. */
export interface LoginRequest {
  identifier: string;
  password: string;
  org_slug?: string;
}

/** Register request body. */
export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  given_name?: string;
  family_name?: string;
  org_slug?: string;
}

/** Login response from Rampart. */
export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user: RampartUser;
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
