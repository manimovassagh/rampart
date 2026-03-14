import { createRemoteJWKSet, jwtVerify, type JWTPayload } from "jose";
import type { RampartClaims, ServerAuth } from "./types.js";

const DEFAULT_COOKIE_NAME = "rampart_token";

/**
 * Validate a JWT token against the Rampart issuer's JWKS.
 * Returns the decoded claims or null if validation fails.
 */
export async function validateToken(
  token: string,
  issuer: string
): Promise<RampartClaims | null> {
  const cleanIssuer = issuer.replace(/\/+$/, "");
  const jwksUrl = new URL(cleanIssuer + "/.well-known/jwks.json");
  const JWKS = createRemoteJWKSet(jwksUrl);

  try {
    const { payload } = await jwtVerify(token, JWKS, {
      issuer: cleanIssuer,
      algorithms: ["RS256"],
    });

    return mapClaims(payload);
  } catch {
    return null;
  }
}

/**
 * Read auth state from cookies in a Server Component or Route Handler.
 *
 * Usage in App Router:
 * ```ts
 * import { cookies } from "next/headers";
 * import { getServerAuth } from "@rampart-auth/nextjs/server";
 *
 * export default async function Page() {
 *   const auth = await getServerAuth(await cookies(), "https://auth.example.com");
 *   if (!auth) redirect("/login");
 *   // auth.claims.sub, auth.claims.email, etc.
 * }
 * ```
 */
export async function getServerAuth(
  cookies: { get(name: string): { value: string } | undefined },
  issuer: string,
  cookieName: string = DEFAULT_COOKIE_NAME
): Promise<ServerAuth | null> {
  const tokenCookie = cookies.get(cookieName);
  if (!tokenCookie) {
    return null;
  }

  const token = tokenCookie.value;
  const claims = await validateToken(token, issuer);
  if (!claims) {
    return null;
  }

  return { claims, token };
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
