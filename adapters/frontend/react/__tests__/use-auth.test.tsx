import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import http from "node:http";
import { render, screen, waitFor, act } from "@testing-library/react";
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

function AuthTester() {
  const { user, isAuthenticated, isLoading, logout, getAccessToken } = useAuth();

  if (isLoading) return createElement("div", { "data-testid": "loading" }, "Loading");

  return createElement(
    "div",
    null,
    createElement("div", { "data-testid": "status" }, isAuthenticated ? "authed" : "anon"),
    createElement("div", { "data-testid": "user" }, user?.email ?? "none"),
    createElement("div", { "data-testid": "token" }, getAccessToken() ?? "no-token"),
    createElement("button", {
      "data-testid": "logout-btn",
      onClick: async () => {
        await logout();
      },
    })
  );
}

function renderWithProvider(child: React.ReactNode) {
  return render(
    createElement(
      RampartProvider,
      { issuer: baseUrl, clientId: "test", redirectUri: "http://localhost:3000/callback" },
      child
    )
  );
}

describe("useAuth", () => {
  it("throws when used outside provider", () => {
    expect(() => {
      render(createElement(AuthTester));
    }).toThrow("useAuth must be used within a <RampartProvider>");
  });

  it("starts as unauthenticated", async () => {
    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
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

    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("authed");
      expect(screen.getByTestId("user").textContent).toBe("jane@example.com");
      expect(screen.getByTestId("token").textContent).toBe("stored-token");
    });
  });

  it("logs out and clears user", async () => {
    window.localStorage.setItem(
      "rampart_tokens",
      JSON.stringify({
        access_token: "stored-token",
        refresh_token: "stored-refresh",
        token_type: "Bearer",
        expires_in: 900,
      })
    );

    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("authed");
    });

    await act(async () => {
      screen.getByTestId("logout-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
      expect(screen.getByTestId("token").textContent).toBe("no-token");
    });
  });

  it("clears localStorage on logout", async () => {
    window.localStorage.setItem(
      "rampart_tokens",
      JSON.stringify({
        access_token: "stored-token",
        refresh_token: "stored-refresh",
        token_type: "Bearer",
        expires_in: 900,
      })
    );

    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("authed");
    });

    await act(async () => {
      screen.getByTestId("logout-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
    });

    expect(window.localStorage.getItem("rampart_tokens")).toBeNull();
  });
});
