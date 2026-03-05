import { createRemoteJWKSet, jwtVerify, type JWTPayload } from "jose";
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import type { RampartMiddlewareConfig, RampartClaims } from "./types.js";

const DEFAULT_COOKIE_NAME = "rampart_token";
const DEFAULT_LOGIN_PATH = "/login";

/**
 * Creates a Next.js Edge Middleware that validates Rampart JWTs.
 *
 * Checks for a token in cookies (cookieName) or the Authorization header.
 * Redirects to loginPath if no valid token is found.
 * Paths listed in publicPaths are skipped.
 */
export function withRampartAuth(config: RampartMiddlewareConfig) {
  const issuer = config.issuer.replace(/\/+$/, "");
  const publicPaths = config.publicPaths ?? [];
  const cookieName = config.cookieName ?? DEFAULT_COOKIE_NAME;
  const loginPath = config.loginPath ?? DEFAULT_LOGIN_PATH;

  const jwksUrl = new URL(issuer + "/.well-known/jwks.json");
  const JWKS = createRemoteJWKSet(jwksUrl);

  return async function middleware(request: NextRequest) {
    const { pathname } = request.nextUrl;

    // Skip auth for public paths
    if (isPublicPath(pathname, publicPaths)) {
      return NextResponse.next();
    }

    // Extract token from cookie or Authorization header
    const token = extractToken(request, cookieName);

    if (!token) {
      return redirectToLogin(request, loginPath);
    }

    try {
      const { payload } = await jwtVerify(token, JWKS, {
        issuer,
        algorithms: ["RS256"],
      });

      // Attach claims as a request header for downstream consumption
      const response = NextResponse.next();
      response.headers.set(
        "x-rampart-claims",
        JSON.stringify(mapClaims(payload))
      );
      return response;
    } catch {
      return redirectToLogin(request, loginPath);
    }
  };
}

function extractToken(request: NextRequest, cookieName: string): string | null {
  // Try cookie first
  const cookieToken = request.cookies.get(cookieName)?.value;
  if (cookieToken) {
    return cookieToken;
  }

  // Fall back to Authorization header
  const authHeader = request.headers.get("authorization");
  if (authHeader) {
    const parts = authHeader.split(" ");
    if (parts.length === 2 && parts[0].toLowerCase() === "bearer") {
      return parts[1];
    }
  }

  return null;
}

function isPublicPath(pathname: string, publicPaths: string[]): boolean {
  return publicPaths.some((publicPath) => {
    if (publicPath.endsWith("*")) {
      return pathname.startsWith(publicPath.slice(0, -1));
    }
    return pathname === publicPath;
  });
}

function redirectToLogin(request: NextRequest, loginPath: string): NextResponse {
  const loginUrl = request.nextUrl.clone();
  loginUrl.pathname = loginPath;
  loginUrl.searchParams.set("callbackUrl", request.nextUrl.pathname);
  return NextResponse.redirect(loginUrl);
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
