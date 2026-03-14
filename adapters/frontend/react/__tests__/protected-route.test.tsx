import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import http from "node:http";
import { render, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { RampartProvider } from "../src/context.js";
import { ProtectedRoute } from "../src/protected-route.js";
import { createMockServer, startServer } from "./helpers.js";

let server: http.Server;
let baseUrl: string;

beforeAll(async () => {
  server = createMockServer();
  baseUrl = await startServer(server);
});

afterAll(() => {
  server?.close();
});

afterEach(() => {
  window.localStorage.clear();
});

function renderWithProvider(child: React.ReactNode) {
  return render(
    createElement(
      RampartProvider,
      { issuer: baseUrl, clientId: "test", redirectUri: "http://localhost:3000/callback" },
      child
    )
  );
}

describe("ProtectedRoute", () => {
  it("shows fallback when not authenticated", async () => {
    renderWithProvider(
      createElement(
        ProtectedRoute,
        { fallback: createElement("div", { "data-testid": "fallback" }, "Please log in") },
        createElement("div", { "data-testid": "secret" }, "Secret content")
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("fallback")).toBeDefined();
      expect(screen.queryByTestId("secret")).toBeNull();
    });
  });

  it("shows children when authenticated via localStorage", async () => {
    window.localStorage.setItem(
      "rampart_tokens",
      JSON.stringify({
        access_token: "valid-token",
        refresh_token: "valid-refresh",
        token_type: "Bearer",
        expires_in: 900,
      })
    );

    renderWithProvider(
      createElement(
        ProtectedRoute,
        { fallback: createElement("div", { "data-testid": "fallback" }, "Nope") },
        createElement("div", { "data-testid": "secret" }, "Secret content")
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("secret")).toBeDefined();
      expect(screen.queryByTestId("fallback")).toBeNull();
    });
  });

  it("shows loadingFallback while provider is loading", async () => {
    window.localStorage.setItem(
      "rampart_tokens",
      JSON.stringify({
        access_token: "tok",
        refresh_token: "ref",
        token_type: "Bearer",
        expires_in: 900,
      })
    );

    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl, clientId: "test", redirectUri: "http://localhost:3000/callback" },
        createElement(
          ProtectedRoute,
          {
            loadingFallback: createElement("div", { "data-testid": "spinner" }, "Loading..."),
            fallback: createElement("div", { "data-testid": "fallback" }, "Nope"),
          },
          createElement("div", { "data-testid": "secret" }, "Secret")
        )
      )
    );

    // During loading, spinner should appear
    expect(screen.getByTestId("spinner")).toBeDefined();

    // After bootstrap, content should show (mock server returns user for valid token)
    await waitFor(() => {
      expect(screen.getByTestId("secret")).toBeDefined();
    });
  });

  it("shows fallback when user lacks required role", async () => {
    // jane has no roles
    window.localStorage.setItem(
      "rampart_tokens",
      JSON.stringify({
        access_token: "valid-token",
        refresh_token: "valid-refresh",
        token_type: "Bearer",
        expires_in: 900,
      })
    );

    renderWithProvider(
      createElement(
        ProtectedRoute,
        {
          roles: ["admin"],
          fallback: createElement("div", { "data-testid": "no-access" }, "No access"),
        },
        createElement("div", { "data-testid": "admin-panel" }, "Admin panel")
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("no-access")).toBeDefined();
      expect(screen.queryByTestId("admin-panel")).toBeNull();
    });
  });

});
