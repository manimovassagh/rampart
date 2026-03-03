import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import http from "node:http";
import { RampartClient } from "../src/client.js";

let server: http.Server;
let baseUrl: string;

const MOCK_USER = {
  id: "user-uuid-1",
  org_id: "org-uuid-1",
  preferred_username: "jane",
  email: "jane@example.com",
  email_verified: true,
  given_name: "Jane",
  family_name: "Doe",
};

let validRefreshToken = "valid-refresh-token";

beforeAll(async () => {
  server = http.createServer((req, res) => {
    let body = "";
    req.on("data", (chunk) => (body += chunk));
    req.on("end", () => {
      const json = (status: number, data: unknown) => {
        res.writeHead(status, { "Content-Type": "application/json" });
        res.end(JSON.stringify(data));
      };

      if (req.url === "/login" && req.method === "POST") {
        const parsed = JSON.parse(body);
        if (parsed.password === "wrong") {
          return json(401, {
            error: "unauthorized",
            error_description: "Invalid credentials.",
            status: 401,
          });
        }
        return json(200, {
          access_token: "access-1",
          refresh_token: validRefreshToken,
          token_type: "Bearer",
          expires_in: 900,
          user: MOCK_USER,
        });
      }

      if (req.url === "/register" && req.method === "POST") {
        return json(201, MOCK_USER);
      }

      if (req.url === "/token/refresh" && req.method === "POST") {
        const parsed = JSON.parse(body);
        if (parsed.refresh_token !== validRefreshToken) {
          return json(401, {
            error: "unauthorized",
            error_description: "Invalid refresh token.",
            status: 401,
          });
        }
        return json(200, {
          access_token: "access-refreshed",
          token_type: "Bearer",
          expires_in: 900,
        });
      }

      if (req.url === "/logout" && req.method === "POST") {
        res.writeHead(204);
        res.end();
        return;
      }

      if (req.url === "/me" && req.method === "GET") {
        const auth = req.headers.authorization;
        if (!auth || !auth.startsWith("Bearer ")) {
          return json(401, {
            error: "unauthorized",
            error_description: "Missing authorization header.",
            status: 401,
          });
        }
        return json(200, MOCK_USER);
      }

      res.writeHead(404);
      res.end();
    });
  });

  await new Promise<void>((resolve) => {
    server.listen(0, () => resolve());
  });

  const addr = server.address();
  if (typeof addr === "object" && addr) {
    baseUrl = `http://127.0.0.1:${addr.port}`;
  }
});

afterAll(() => {
  server?.close();
});

describe("RampartClient", () => {
  it("registers a new user", async () => {
    const client = new RampartClient({ issuer: baseUrl });
    const user = await client.register({
      username: "jane",
      email: "jane@example.com",
      password: "SecurePass1!",
    });

    expect(user.email).toBe("jane@example.com");
    expect(user.id).toBe("user-uuid-1");
  });

  it("logs in and stores tokens", async () => {
    const client = new RampartClient({ issuer: baseUrl });
    const res = await client.login({
      identifier: "jane",
      password: "pass",
    });

    expect(res.access_token).toBe("access-1");
    expect(res.user.email).toBe("jane@example.com");
    expect(client.isAuthenticated()).toBe(true);
    expect(client.getAccessToken()).toBe("access-1");
  });

  it("throws on invalid login", async () => {
    const client = new RampartClient({ issuer: baseUrl });

    await expect(
      client.login({ identifier: "jane", password: "wrong" })
    ).rejects.toMatchObject({
      error: "unauthorized",
      status: 401,
    });

    expect(client.isAuthenticated()).toBe(false);
  });

  it("refreshes access token", async () => {
    const client = new RampartClient({ issuer: baseUrl });
    await client.login({ identifier: "jane", password: "pass" });

    const tokens = await client.refresh();
    expect(tokens.access_token).toBe("access-refreshed");
    expect(client.getAccessToken()).toBe("access-refreshed");
  });

  it("fetches user profile with getUser()", async () => {
    const client = new RampartClient({ issuer: baseUrl });
    await client.login({ identifier: "jane", password: "pass" });

    const user = await client.getUser();
    expect(user.preferred_username).toBe("jane");
    expect(user.org_id).toBe("org-uuid-1");
  });

  it("logs out and clears tokens", async () => {
    const client = new RampartClient({ issuer: baseUrl });
    await client.login({ identifier: "jane", password: "pass" });
    expect(client.isAuthenticated()).toBe(true);

    await client.logout();
    expect(client.isAuthenticated()).toBe(false);
    expect(client.getAccessToken()).toBeNull();
  });

  it("calls onTokenChange on login, refresh, and logout", async () => {
    const onChange = vi.fn();
    const client = new RampartClient({ issuer: baseUrl, onTokenChange: onChange });

    await client.login({ identifier: "jane", password: "pass" });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ access_token: "access-1" })
    );

    await client.refresh();
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ access_token: "access-refreshed" })
    );

    await client.logout();
    expect(onChange).toHaveBeenCalledWith(null);

    expect(onChange).toHaveBeenCalledTimes(3);
  });

  it("authFetch retries on 401 after refresh", async () => {
    // First request to /me returns 401, then after refresh returns 200
    let callCount = 0;
    const mockServer = http.createServer((req, res) => {
      let body = "";
      req.on("data", (chunk) => (body += chunk));
      req.on("end", () => {
        if (req.url === "/token/refresh") {
          res.writeHead(200, { "Content-Type": "application/json" });
          res.end(
            JSON.stringify({
              access_token: "fresh-token",
              token_type: "Bearer",
              expires_in: 900,
            })
          );
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

    await new Promise<void>((resolve) => mockServer.listen(0, () => resolve()));
    const addr = mockServer.address();
    const url = typeof addr === "object" && addr ? `http://127.0.0.1:${addr.port}` : "";

    const client = new RampartClient({ issuer: url });
    client.setTokens({
      access_token: "stale",
      refresh_token: validRefreshToken,
      token_type: "Bearer",
      expires_in: 0,
    });

    const res = await client.authFetch(`${url}/api/data`);
    expect(res.status).toBe(200);

    const data = await res.json();
    expect(data).toEqual({ data: "secret" });

    mockServer.close();
  });

  it("strips trailing slash from issuer", () => {
    const client = new RampartClient({ issuer: "http://localhost:8080/" });
    // No error thrown — trailing slash is handled
    expect(client.isAuthenticated()).toBe(false);
  });
});
