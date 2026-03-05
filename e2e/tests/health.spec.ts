import { test, expect } from "@playwright/test";

test.describe("Health endpoints", () => {
  test("GET /healthz returns alive", async ({ request }) => {
    const resp = await request.get("/healthz");
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.status).toBe("alive");
  });

  test("GET /readyz returns ready", async ({ request }) => {
    const resp = await request.get("/readyz");
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.status).toBe("ready");
  });
});
