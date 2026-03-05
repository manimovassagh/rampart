export interface RampartConfig {
  issuer: string;
}

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

export interface RampartError {
  error: string;
  error_description: string;
  status: number;
}

declare global {
  namespace Express {
    interface Request {
      auth?: RampartClaims;
    }
  }
}
