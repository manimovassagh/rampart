import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import http from "node:http";
import { render, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { RampartProvider } from "../src/context.js";
import { useAuth } from "../src/use-auth.js";
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

function StatusDisplay() {
  const { user, isLoading, isAuthenticated } = useAuth();
  if (isLoading) return createElement("div", { "data-testid": "loading" }, "Loading");
  if (isAuthenticated) {
    return createElement("div", { "data-testid": "user" }, user!.email);
  }
  return createElement("div", { "data-testid": "anon" }, "Not authenticated");
}

describe("RampartProvider", () => {
  it("shows loading then resolves to unauthenticated", async () => {
    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl, clientId: "test", redirectUri: "http://localhost:3000/callback" },
        createElement(StatusDisplay)
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("anon")).toBeDefined();
    });
  });

  it("restores tokens from localStorage and fetches user", async () => {
    window.localStorage.setItem(
      "rampart_tokens",
      JSON.stringify({
        access_token: "stored-token",
        refresh_token: "stored-refresh",
        token_type: "Bearer",
        expires_in: 900,
      })
    );

    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl, clientId: "test", redirectUri: "http://localhost:3000/callback" },
        createElement(StatusDisplay)
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("user").textContent).toBe("jane@example.com");
    });
  });

  it("clears invalid tokens and shows unauthenticated", async () => {
    window.localStorage.setItem("rampart_tokens", "invalid-json{{{");

    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl, clientId: "test", redirectUri: "http://localhost:3000/callback" },
        createElement(StatusDisplay)
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("anon")).toBeDefined();
    });
  });

  it("skips localStorage when persist is false", async () => {
    window.localStorage.setItem(
      "rampart_tokens",
      JSON.stringify({
        access_token: "stored-token",
        refresh_token: "stored-refresh",
        token_type: "Bearer",
        expires_in: 900,
      })
    );

    render(
      createElement(
        RampartProvider,
        { issuer: baseUrl, clientId: "test", redirectUri: "http://localhost:3000/callback", persist: false },
        createElement(StatusDisplay)
      )
    );

    await waitFor(() => {
      expect(screen.getByTestId("anon")).toBeDefined();
    });
  });
});
