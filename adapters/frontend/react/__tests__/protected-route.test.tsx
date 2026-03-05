import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import http from "node:http";
import { render, screen, waitFor, act } from "@testing-library/react";
import { createElement, useState } from "react";
import { RampartProvider } from "../src/context.js";
import { useAuth } from "../src/use-auth.js";
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

function LoginTrigger({ identifier }: { identifier: string }) {
  const { login, isLoading } = useAuth();
  if (isLoading) return createElement("div", { "data-testid": "loading" }, "Loading");
  return createElement("button", {
    "data-testid": "login-btn",
    onClick: () => login({ identifier, password: "pass" }),
  });
}

describe("ProtectedRoute", () => {
  it("shows fallback when not authenticated", async () => {
    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl },
        createElement(
          ProtectedRoute,
          { fallback: createElement("div", { "data-testid": "fallback" }, "Please log in") },
          createElement("div", { "data-testid": "secret" }, "Secret content")
        )
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("fallback")).toBeDefined();
      expect(screen.queryByTestId("secret")).toBeNull();
    });
  });

  it("shows children when authenticated", async () => {
    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl },
        createElement(LoginTrigger, { identifier: "jane" }),
        createElement(
          ProtectedRoute,
          { fallback: createElement("div", { "data-testid": "fallback" }, "Nope") },
          createElement("div", { "data-testid": "secret" }, "Secret content")
        )
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("login-btn")).toBeDefined();
    });

    await act(async () => {
      screen.getByTestId("login-btn").click();
    });

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

    const { container } = render(
      createElement(
        RampartProvider,
        { issuer: baseUrl },
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
    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl },
        createElement(LoginTrigger, { identifier: "jane" }),
        createElement(
          ProtectedRoute,
          {
            roles: ["admin"],
            fallback: createElement("div", { "data-testid": "no-access" }, "No access"),
          },
          createElement("div", { "data-testid": "admin-panel" }, "Admin panel")
        )
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("login-btn")).toBeDefined();
    });

    await act(async () => {
      screen.getByTestId("login-btn").click();
    });

    // jane has no roles — should see fallback
    await waitFor(() => {
      expect(screen.getByTestId("no-access")).toBeDefined();
      expect(screen.queryByTestId("admin-panel")).toBeNull();
    });
  });

  it("shows children when user has required role", async () => {
    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl },
        createElement(LoginTrigger, { identifier: "admin" }),
        createElement(
          ProtectedRoute,
          {
            roles: ["admin"],
            fallback: createElement("div", { "data-testid": "no-access" }, "No access"),
          },
          createElement("div", { "data-testid": "admin-panel" }, "Admin panel")
        )
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("login-btn")).toBeDefined();
    });

    await act(async () => {
      screen.getByTestId("login-btn").click();
    });

    // admin user has ["admin", "user"] roles — should see admin panel
    await waitFor(() => {
      expect(screen.getByTestId("admin-panel")).toBeDefined();
      expect(screen.queryByTestId("no-access")).toBeNull();
    });
  });
});
