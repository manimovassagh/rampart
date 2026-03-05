import { test, expect } from "@playwright/test";

const testUser = {
  username: `pw${Date.now()}`,
  email: `pw${Date.now()}@e2e.test`,
  password: "E2eSecureP@ss1",
  given_name: "PW",
  family_name: "Test",
};

let accessToken = "";
let refreshToken = "";

test.describe.serial("Authentication flow", () => {
  test("register a new user via API", async ({ request }) => {
    const resp = await request.post("/register", { data: testUser });
    expect(resp.ok()).toBeTruthy();

    const body = await resp.json();
    expect(body.id).toBeTruthy();
    expect(body.username).toBe(testUser.username);
    expect(body.email).toBe(testUser.email);
    expect(body.enabled).toBe(true);
  });

  test("login returns tokens", async ({ request }) => {
    const resp = await request.post("/login", {
      data: {
        identifier: testUser.email,
        password: testUser.password,
      },
    });
    expect(resp.ok()).toBeTruthy();

    const body = await resp.json();
    expect(body.access_token).toBeTruthy();
    expect(body.refresh_token).toBeTruthy();
    expect(body.token_type).toBe("Bearer");
    expect(body.expires_in).toBeGreaterThan(0);
    expect(body.user.email).toBe(testUser.email);

    accessToken = body.access_token;
    refreshToken = body.refresh_token;
  });

  test("GET /me returns user identity", async ({ request }) => {
    const resp = await request.get("/me", {
      headers: { Authorization: `Bearer ${accessToken}` },
    });
    expect(resp.ok()).toBeTruthy();

    const body = await resp.json();
    expect(body.preferred_username).toBe(testUser.username);
    expect(body.email).toBe(testUser.email);
    expect(body.id).toBeTruthy();
    expect(body.org_id).toBeTruthy();
  });

  test("token refresh returns new access token", async ({ request }) => {
    const resp = await request.post("/token/refresh", {
      data: { refresh_token: refreshToken },
    });
    expect(resp.ok()).toBeTruthy();

    const body = await resp.json();
    expect(body.access_token).toBeTruthy();
    expect(body.token_type).toBe("Bearer");
    expect(body.expires_in).toBeGreaterThan(0);
  });

  test("invalid credentials returns 401", async ({ request }) => {
    const resp = await request.post("/login", {
      data: {
        identifier: testUser.email,
        password: "wrong-password",
      },
    });
    expect(resp.status()).toBe(401);
  });

  test("GET /me without token returns 401", async ({ request }) => {
    const resp = await request.get("/me");
    expect(resp.status()).toBe(401);
  });
});
