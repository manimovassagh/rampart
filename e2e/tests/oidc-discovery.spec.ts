import { test, expect } from "@playwright/test";

test.describe("OIDC Discovery", () => {
  test("GET /.well-known/openid-configuration returns valid metadata", async ({
    request,
  }) => {
    const resp = await request.get("/.well-known/openid-configuration");
    expect(resp.ok()).toBeTruthy();

    const body = await resp.json();
    expect(body.issuer).toBeTruthy();
    expect(body.authorization_endpoint).toContain("/oauth/authorize");
    expect(body.token_endpoint).toContain("/oauth/token");
    expect(body.jwks_uri).toContain("/.well-known/jwks.json");
    expect(body.response_types_supported).toContain("code");
    expect(body.grant_types_supported).toContain("authorization_code");
    expect(body.scopes_supported).toContain("openid");
    expect(body.id_token_signing_alg_values_supported).toContain("RS256");
    expect(body.code_challenge_methods_supported).toContain("S256");
  });

  test("GET /.well-known/jwks.json returns RSA key", async ({ request }) => {
    const resp = await request.get("/.well-known/jwks.json");
    expect(resp.ok()).toBeTruthy();

    const body = await resp.json();
    expect(body.keys).toHaveLength(1);
    expect(body.keys[0].kty).toBe("RSA");
    expect(body.keys[0].alg).toBe("RS256");
    expect(body.keys[0].use).toBe("sig");
    expect(body.keys[0].kid).toBeTruthy();
    expect(body.keys[0].n).toBeTruthy();
    expect(body.keys[0].e).toBe("AQAB");
  });
});
