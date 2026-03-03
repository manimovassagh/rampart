import { createRemoteJWKSet, jwtVerify, type JWTPayload } from "jose";
import type { Request, Response, NextFunction } from "express";
import type { RampartClaims, RampartConfig } from "./types.js";

export function rampartAuth(config: RampartConfig) {
  const issuer = config.issuer.replace(/\/+$/, "");
  const jwksUrl = new URL(issuer + "/.well-known/jwks.json");
  const JWKS = createRemoteJWKSet(jwksUrl);

  return async (req: Request, res: Response, next: NextFunction) => {
    const header = req.headers.authorization;

    if (!header) {
      sendUnauthorized(res, "Missing authorization header.");
      return;
    }

    const parts = header.split(" ");
    if (parts.length !== 2 || parts[0].toLowerCase() !== "bearer") {
      sendUnauthorized(res, "Invalid authorization header format.");
      return;
    }

    try {
      const { payload } = await jwtVerify(parts[1], JWKS, {
        issuer,
        algorithms: ["RS256"],
      });

      req.auth = mapClaims(payload);
      next();
    } catch {
      sendUnauthorized(res, "Invalid or expired access token.");
    }
  };
}

function mapClaims(payload: JWTPayload): RampartClaims {
  const claims: RampartClaims = {
    iss: payload.iss as string,
    sub: payload.sub as string,
    iat: payload.iat as number,
    exp: payload.exp as number,
    org_id: payload.org_id as string,
    preferred_username: payload.preferred_username as string,
    email: payload.email as string,
    email_verified: payload.email_verified as boolean,
  };

  if (payload.given_name) {
    claims.given_name = payload.given_name as string;
  }
  if (payload.family_name) {
    claims.family_name = payload.family_name as string;
  }
  if (Array.isArray(payload.roles)) {
    claims.roles = payload.roles as string[];
  }

  return claims;
}

function sendUnauthorized(res: Response, description: string): void {
  res.status(401).json({
    error: "unauthorized",
    error_description: description,
    status: 401,
  });
}
