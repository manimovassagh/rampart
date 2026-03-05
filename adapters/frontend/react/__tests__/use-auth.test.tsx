import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import http from "node:http";
import { render, screen, waitFor, act } from "@testing-library/react";
import { createElement, useState } from "react";
import { RampartProvider } from "../src/context.js";
import { useAuth } from "../src/use-auth.js";
import { createMockServer, startServer, MOCK_USER } from "./helpers.js";

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
  const { user, isAuthenticated, isLoading, login, register, logout, getAccessToken } =
    useAuth();
  const [result, setResult] = useState<string>("");
  const [error, setError] = useState<string>("");

  if (isLoading) return createElement("div", { "data-testid": "loading" }, "Loading");

  return createElement(
    "div",
    null,
    createElement("div", { "data-testid": "status" }, isAuthenticated ? "authed" : "anon"),
    createElement("div", { "data-testid": "user" }, user?.email ?? "none"),
    createElement("div", { "data-testid": "token" }, getAccessToken() ?? "no-token"),
    createElement("div", { "data-testid": "result" }, result),
    createElement("div", { "data-testid": "error" }, error),
    createElement("button", {
      "data-testid": "login-btn",
      onClick: async () => {
        try {
          await login({ identifier: "jane", password: "pass" });
          setResult("login-ok");
        } catch (e: unknown) {
          setError((e as { error_description: string }).error_description);
        }
      },
    }),
    createElement("button", {
      "data-testid": "login-fail-btn",
      onClick: async () => {
        try {
          await login({ identifier: "jane", password: "wrong" });
        } catch (e: unknown) {
          setError((e as { error_description: string }).error_description);
        }
      },
    }),
    createElement("button", {
      "data-testid": "register-btn",
      onClick: async () => {
        try {
          const u = await register({
            username: "jane",
            email: "jane@example.com",
            password: "SecurePass1!",
          });
          setResult(u.email);
        } catch (e: unknown) {
          setError((e as { error: string }).error);
        }
      },
    }),
    createElement("button", {
      "data-testid": "logout-btn",
      onClick: async () => {
        await logout();
        setResult("logged-out");
      },
    })
  );
}

function renderWithProvider(child: React.ReactNode) {
  return render(
    createElement(RampartProvider, { issuer: baseUrl }, child)
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

  it("logs in and updates user state", async () => {
    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
    });

    await act(async () => {
      screen.getByTestId("login-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("authed");
      expect(screen.getByTestId("user").textContent).toBe("jane@example.com");
      expect(screen.getByTestId("token").textContent).toBe("access-1");
      expect(screen.getByTestId("result").textContent).toBe("login-ok");
    });
  });

  it("throws on invalid login", async () => {
    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
    });

    await act(async () => {
      screen.getByTestId("login-fail-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("error").textContent).toBe("Invalid credentials.");
      expect(screen.getByTestId("status").textContent).toBe("anon");
    });
  });

  it("registers a user", async () => {
    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
    });

    await act(async () => {
      screen.getByTestId("register-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("result").textContent).toBe("jane@example.com");
    });
  });

  it("logs out and clears user", async () => {
    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
    });

    await act(async () => {
      screen.getByTestId("login-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("authed");
    });

    await act(async () => {
      screen.getByTestId("logout-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
      expect(screen.getByTestId("token").textContent).toBe("no-token");
      expect(screen.getByTestId("result").textContent).toBe("logged-out");
    });
  });

  it("persists tokens to localStorage on login", async () => {
    renderWithProvider(createElement(AuthTester));

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("anon");
    });

    await act(async () => {
      screen.getByTestId("login-btn").click();
    });

    await waitFor(() => {
      expect(screen.getByTestId("status").textContent).toBe("authed");
    });

    const stored = window.localStorage.getItem("rampart_tokens");
    expect(stored).toBeTruthy();
    const tokens = JSON.parse(stored!);
    expect(tokens.access_token).toBe("access-1");
  });
});
