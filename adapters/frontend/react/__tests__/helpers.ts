import http from "node:http";

export const MOCK_USER = {
  id: "user-uuid-1",
  org_id: "org-uuid-1",
  preferred_username: "jane",
  email: "jane@example.com",
  email_verified: true,
  given_name: "Jane",
  family_name: "Doe",
};

export const MOCK_ADMIN = {
  ...MOCK_USER,
  id: "admin-uuid-1",
  preferred_username: "admin",
  email: "admin@example.com",
  roles: ["admin", "user"],
};

export const VALID_REFRESH_TOKEN = "valid-refresh-token";

export function createMockServer(): http.Server {
  return http.createServer((req, res) => {
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
        const user = parsed.identifier === "admin" ? MOCK_ADMIN : MOCK_USER;
        return json(200, {
          access_token: "access-1",
          refresh_token: VALID_REFRESH_TOKEN,
          token_type: "Bearer",
          expires_in: 900,
          user,
        });
      }

      if (req.url === "/register" && req.method === "POST") {
        return json(201, MOCK_USER);
      }

      if (req.url === "/token/refresh" && req.method === "POST") {
        const parsed = JSON.parse(body);
        if (parsed.refresh_token !== VALID_REFRESH_TOKEN) {
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
}

export async function startServer(server: http.Server): Promise<string> {
  await new Promise<void>((resolve) => {
    server.listen(0, () => resolve());
  });
  const addr = server.address();
  if (typeof addr === "object" && addr) {
    return `http://127.0.0.1:${addr.port}`;
  }
  throw new Error("Failed to start mock server");
}
