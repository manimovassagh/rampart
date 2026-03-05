import { describe, it, expect, beforeAll, afterAll } from "vitest";
import express from "express";
import request from "supertest";
import { generateKeyPair, exportJWK, SignJWT } from "jose";
import type { Server } from "node:http";
import { rampartAuth } from "../src/middleware.js";

let server: Server;
let issuer: string;
let privateKey: CryptoKey;

beforeAll(async () => {
  const { publicKey, privateKey: privKey } = await generateKeyPair("RS256");
  privateKey = privKey;

  const jwk = await exportJWK(publicKey);
  jwk.kid = "test-key-1";
  jwk.use = "sig";
  jwk.alg = "RS256";

  const jwksApp = express();
  jwksApp.get("/.well-known/jwks.json", (_req, res) => {
    res.json({ keys: [jwk] });
  });

  await new Promise<void>((resolve) => {
    server = jwksApp.listen(0, () => resolve());
  });

  const addr = server.address();
  if (typeof addr === "object" && addr) {
    issuer = `http://127.0.0.1:${addr.port}`;
  }
});

afterAll(() => {
  server?.close();
});

function buildToken(overrides: Record<string, unknown> = {}, expOverride?: number) {
  const now = Math.floor(Date.now() / 1000);
  const jwt = new SignJWT({
    org_id: "org-uuid-1234",
    preferred_username: "jane",
    email: "jane@example.com",
    email_verified: true,
    given_name: "Jane",
    family_name: "Doe",
    ...overrides,
  })
    .setProtectedHeader({ alg: "RS256", kid: "test-key-1" })
    .setIssuedAt(now)
    .setExpirationTime(expOverride ?? now + 3600)
    .setSubject("user-uuid-5678")
    .setIssuer(issuer);

  return jwt.sign(privateKey);
}

function createApp(perRoute = false) {
  const app = express();
  const middleware = rampartAuth({ issuer });

  if (perRoute) {
    app.get("/public", (_req, res) => {
      res.json({ ok: true });
    });
    app.get("/protected", middleware, (req, res) => {
      res.json({ auth: req.auth });
    });
  } else {
    app.use(middleware);
    app.get("/protected", (req, res) => {
      res.json({ auth: req.auth });
    });
  }

  return app;
}

describe("rampartAuth middleware", () => {
  it("returns 401 when Authorization header is missing", async () => {
    const res = await request(createApp()).get("/protected");

    expect(res.status).toBe(401);
    expect(res.body.error).toBe("unauthorized");
    expect(res.body.error_description).toBe("Missing authorization header.");
  });

  it("returns 401 for non-Bearer auth scheme", async () => {
    const res = await request(createApp())
      .get("/protected")
      .set("Authorization", "Basic dXNlcjpwYXNz");

    expect(res.status).toBe(401);
    expect(res.body.error_description).toBe(
      "Invalid authorization header format."
    );
  });

  it("returns 401 for an invalid token", async () => {
    const res = await request(createApp())
      .get("/protected")
      .set("Authorization", "Bearer not.a.real.token");

    expect(res.status).toBe(401);
    expect(res.body.error_description).toBe(
      "Invalid or expired access token."
    );
  });

  it("returns 401 for an expired token", async () => {
    const pastExp = Math.floor(Date.now() / 1000) - 3600;
    const token = await buildToken({}, pastExp);

    const res = await request(createApp())
      .get("/protected")
      .set("Authorization", `Bearer ${token}`);

    expect(res.status).toBe(401);
    expect(res.body.error_description).toBe(
      "Invalid or expired access token."
    );
  });

  it("returns 401 when token issuer does not match", async () => {
    const now = Math.floor(Date.now() / 1000);
    const token = await new SignJWT({
      org_id: "org-uuid-1234",
      preferred_username: "jane",
      email: "jane@example.com",
      email_verified: true,
    })
      .setProtectedHeader({ alg: "RS256", kid: "test-key-1" })
      .setIssuedAt(now)
      .setExpirationTime(now + 3600)
      .setSubject("user-uuid-5678")
      .setIssuer("http://wrong-issuer.example.com")
      .sign(privateKey);

    const res = await request(createApp())
      .get("/protected")
      .set("Authorization", `Bearer ${token}`);

    expect(res.status).toBe(401);
  });

  it("populates req.auth with all claims on valid token", async () => {
    const token = await buildToken();

    const res = await request(createApp())
      .get("/protected")
      .set("Authorization", `Bearer ${token}`);

    expect(res.status).toBe(200);
    expect(res.body.auth).toMatchObject({
      iss: issuer,
      sub: "user-uuid-5678",
      org_id: "org-uuid-1234",
      preferred_username: "jane",
      email: "jane@example.com",
      email_verified: true,
      given_name: "Jane",
      family_name: "Doe",
    });
    expect(typeof res.body.auth.iat).toBe("number");
    expect(typeof res.body.auth.exp).toBe("number");
  });

  it("works as per-route middleware", async () => {
    const app = createApp(true);
    const token = await buildToken();

    const publicRes = await request(app).get("/public");
    expect(publicRes.status).toBe(200);
    expect(publicRes.body).toEqual({ ok: true });

    const protectedRes = await request(app)
      .get("/protected")
      .set("Authorization", `Bearer ${token}`);
    expect(protectedRes.status).toBe(200);
    expect(protectedRes.body.auth.sub).toBe("user-uuid-5678");
  });

  it("omits given_name and family_name when not in token", async () => {
    const token = await buildToken({
      given_name: undefined,
      family_name: undefined,
    });

    const res = await request(createApp())
      .get("/protected")
      .set("Authorization", `Bearer ${token}`);

    expect(res.status).toBe(200);
    expect(res.body.auth).not.toHaveProperty("given_name");
    expect(res.body.auth).not.toHaveProperty("family_name");
  });

  it("returns Content-Type application/json on 401", async () => {
    const res = await request(createApp()).get("/protected");

    expect(res.status).toBe(401);
    expect(res.headers["content-type"]).toMatch(/application\/json/);
  });
});
