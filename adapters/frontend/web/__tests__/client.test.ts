import { describe, it, expect, vi } from "vitest";
import http from "node:http";
import { RampartClient } from "../src/client.js";

/** Create a minimal JWT string with the given payload (no real signature). */
function makeJwt(payload: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: "RS256", typ: "JWT" })).toString("base64");
  const body = Buffer.from(JSON.stringify(payload)).toString("base64");
  return `${header}.${body}.fake-signature`;
}

function makeClient(issuer = "http://localhost:8080") {
  return new RampartClient({
    issuer,
    clientId: "test-client",
    redirectUri: "http://localhost:3000/callback",
  });
}

describe("RampartClient", () => {
  describe("isAuthenticated", () => {
    it("returns false when no tokens are set", () => {
      const client = makeClient();
      expect(client.isAuthenticated()).toBe(false);
    });

    it("returns true for a valid non-expired token", () => {
      const client = makeClient();
      client.setTokens({
        access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 }),
        refresh_token: "rt",
        token_type: "Bearer",
        expires_in: 3600,
      });
      expect(client.isAuthenticated()).toBe(true);
    });

    it("returns false for an expired token", () => {
      const client = makeClient();
      client.setTokens({
        access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) - 60 }),
        refresh_token: "rt",
        token_type: "Bearer",
        expires_in: 3600,
      });
      expect(client.isAuthenticated()).toBe(false);
    });

    it("returns false for a malformed token", () => {
      const client = makeClient();
      client.setTokens({
        access_token: "not-a-jwt",
        refresh_token: "rt",
        token_type: "Bearer",
        expires_in: 3600,
      });
      expect(client.isAuthenticated()).toBe(false);
    });

    it("returns false for a token without exp claim", () => {
      const client = makeClient();
      client.setTokens({
        access_token: makeJwt({ sub: "user-1" }),
        refresh_token: "rt",
        token_type: "Bearer",
        expires_in: 3600,
      });
      expect(client.isAuthenticated()).toBe(false);
    });
  });

  describe("token management", () => {
    it("getAccessToken returns null when no tokens set", () => {
      const client = makeClient();
      expect(client.getAccessToken()).toBeNull();
    });

    it("getAccessToken returns the access token after setTokens", () => {
      const client = makeClient();
      const token = makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 });
      client.setTokens({
        access_token: token,
        refresh_token: "rt",
        token_type: "Bearer",
        expires_in: 3600,
      });
      expect(client.getAccessToken()).toBe(token);
    });

    it("getTokens returns a copy of tokens", () => {
      const client = makeClient();
      const tokens = {
        access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 }),
        refresh_token: "rt",
        token_type: "Bearer",
        expires_in: 3600,
      };
      client.setTokens(tokens);
      const got = client.getTokens();
      expect(got).toEqual(tokens);
      expect(got).not.toBe(tokens); // should be a copy
    });

    it("calls onTokenChange when tokens change", () => {
      const onChange = vi.fn();
      const client = new RampartClient({
        issuer: "http://localhost:8080",
        clientId: "test",
        redirectUri: "http://localhost:3000/callback",
        onTokenChange: onChange,
      });

      const tokens = {
        access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 }),
        refresh_token: "rt",
        token_type: "Bearer",
        expires_in: 3600,
      };

      client.setTokens(tokens);
      expect(onChange).toHaveBeenCalledWith(tokens);

      client.setTokens(null);
      expect(onChange).toHaveBeenCalledWith(null);
      expect(onChange).toHaveBeenCalledTimes(2);
    });
  });

  describe("issuer normalization", () => {
    it("strips trailing slash from issuer", () => {
      const client = new RampartClient({
        issuer: "http://localhost:8080/",
        clientId: "test",
        redirectUri: "http://localhost:3000/callback",
      });
      expect(client.isAuthenticated()).toBe(false);
    });
  });

  describe("refresh", () => {
    it("throws when no refresh token is available", async () => {
      const client = makeClient();
      await expect(client.refresh()).rejects.toThrow("No refresh token available.");
    });

    it("refreshes tokens via /token/refresh endpoint", async () => {
      const newAccessToken = makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 });

      const server = http.createServer((req, res) => {
        let body = "";
        req.on("data", (chunk) => (body += chunk));
        req.on("end", () => {
          if (req.url === "/token/refresh" && req.method === "POST") {
            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify({
              access_token: newAccessToken,
              token_type: "Bearer",
              expires_in: 900,
            }));
          } else {
            res.writeHead(404);
            res.end();
          }
        });
      });

      await new Promise<void>((resolve) => server.listen(0, () => resolve()));
      const addr = server.address();
      const url = typeof addr === "object" && addr ? `http://127.0.0.1:${addr.port}` : "";

      try {
        const client = makeClient(url);
        client.setTokens({
          access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) + 60 }),
          refresh_token: "valid-rt",
          token_type: "Bearer",
          expires_in: 900,
        });

        const tokens = await client.refresh();
        expect(tokens.access_token).toBe(newAccessToken);
        expect(client.getAccessToken()).toBe(newAccessToken);
      } finally {
        server.close();
      }
    });
  });

  describe("authFetch", () => {
    it("retries on 401 after token refresh", async () => {
      const freshToken = makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 });
      let callCount = 0;

      const server = http.createServer((req, res) => {
        let body = "";
        req.on("data", (chunk) => (body += chunk));
        req.on("end", () => {
          if (req.url === "/token/refresh") {
            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify({
              access_token: freshToken,
              token_type: "Bearer",
              expires_in: 900,
            }));
            return;
          }

          callCount++;
          if (callCount === 1) {
            res.writeHead(401, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ error: "unauthorized", status: 401 }));
          } else {
            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ data: "secret" }));
          }
        });
      });

      await new Promise<void>((resolve) => server.listen(0, () => resolve()));
      const addr = server.address();
      const url = typeof addr === "object" && addr ? `http://127.0.0.1:${addr.port}` : "";

      try {
        const client = makeClient(url);
        client.setTokens({
          access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) + 60 }),
          refresh_token: "valid-rt",
          token_type: "Bearer",
          expires_in: 0,
        });

        const res = await client.authFetch(`${url}/api/data`);
        expect(res.status).toBe(200);

        const data = await res.json();
        expect(data).toEqual({ data: "secret" });
      } finally {
        server.close();
      }
    });
  });

  describe("logout", () => {
    it("clears tokens and calls server", async () => {
      let logoutCalled = false;

      const server = http.createServer((req, res) => {
        req.on("data", () => {});
        req.on("end", () => {
          if (req.url === "/logout" && req.method === "POST") {
            logoutCalled = true;
            res.writeHead(204);
            res.end();
          } else {
            res.writeHead(404);
            res.end();
          }
        });
      });

      await new Promise<void>((resolve) => server.listen(0, () => resolve()));
      const addr = server.address();
      const url = typeof addr === "object" && addr ? `http://127.0.0.1:${addr.port}` : "";

      try {
        const client = makeClient(url);
        client.setTokens({
          access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 }),
          refresh_token: "rt",
          token_type: "Bearer",
          expires_in: 3600,
        });

        expect(client.isAuthenticated()).toBe(true);
        await client.logout();
        expect(client.isAuthenticated()).toBe(false);
        expect(client.getAccessToken()).toBeNull();
        expect(logoutCalled).toBe(true);
      } finally {
        server.close();
      }
    });
  });

  describe("getUser", () => {
    it("fetches user profile from /me endpoint", async () => {
      const mockUser = {
        id: "user-1",
        org_id: "org-1",
        email: "jane@example.com",
        email_verified: true,
      };

      const server = http.createServer((req, res) => {
        req.on("data", () => {});
        req.on("end", () => {
          if (req.url === "/me" && req.method === "GET") {
            const auth = req.headers.authorization;
            if (!auth?.startsWith("Bearer ")) {
              res.writeHead(401, { "Content-Type": "application/json" });
              res.end(JSON.stringify({ error: "unauthorized", status: 401 }));
              return;
            }
            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify(mockUser));
          } else {
            res.writeHead(404);
            res.end();
          }
        });
      });

      await new Promise<void>((resolve) => server.listen(0, () => resolve()));
      const addr = server.address();
      const url = typeof addr === "object" && addr ? `http://127.0.0.1:${addr.port}` : "";

      try {
        const client = makeClient(url);
        client.setTokens({
          access_token: makeJwt({ exp: Math.floor(Date.now() / 1000) + 3600 }),
          refresh_token: "rt",
          token_type: "Bearer",
          expires_in: 3600,
        });

        const user = await client.getUser();
        expect(user.email).toBe("jane@example.com");
        expect(user.id).toBe("user-1");
      } finally {
        server.close();
      }
    });
  });
});
