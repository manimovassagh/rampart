/** Standard OIDC/Rampart JWT claims. */
export interface RampartClaims {
  iss: string;
  sub: string;
  iat: number;
  exp: number;
  org_id: string;
  preferred_username: string;
  email: string;
  email_verified: boolean;
  given_name?: string;
  family_name?: string;
  roles?: string[];
}

/** Configuration for the Rampart Next.js middleware. */
export interface RampartMiddlewareConfig {
  /** Rampart server URL (e.g. "https://auth.example.com") */
  issuer: string;
  /** Paths that do not require authentication (supports glob-like prefix matching). */
  publicPaths?: string[];
  /** Cookie name that holds the access token. Default: "rampart_token" */
  cookieName?: string;
  /** Path to redirect unauthenticated users. Default: "/login" */
  loginPath?: string;
}

/** Result of server-side auth resolution. */
export interface ServerAuth {
  claims: RampartClaims;
  token: string;
}
