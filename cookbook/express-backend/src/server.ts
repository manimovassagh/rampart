import express from "express";
import cors from "cors";
import { rampartAuth, requireRoles } from "@rampart-auth/node";

const RAMPART_ISSUER = process.env.RAMPART_ISSUER ?? "http://localhost:8080";
const PORT = Number(process.env.PORT ?? 3001);

const app = express();

app.use(cors({ origin: "*" }));
app.use(express.json());

const auth = rampartAuth({ issuer: RAMPART_ISSUER });

// Public route — no auth required
app.get("/api/health", (_req, res) => {
  res.json({ status: "ok", issuer: RAMPART_ISSUER });
});

// Protected route — requires valid Rampart JWT
app.get("/api/profile", auth, (req, res) => {
  res.json({
    message: "Authenticated!",
    user: {
      id: req.auth!.sub,
      email: req.auth!.email,
      username: req.auth!.preferred_username,
      org_id: req.auth!.org_id,
      email_verified: req.auth!.email_verified,
      given_name: req.auth!.given_name,
      family_name: req.auth!.family_name,
      roles: req.auth!.roles ?? [],
    },
  });
});

// Protected route — returns all raw claims
app.get("/api/claims", auth, (req, res) => {
  res.json(req.auth);
});

// Role-protected route — requires "editor" role
app.get("/api/editor/dashboard", auth, requireRoles("editor"), (req, res) => {
  res.json({
    message: "Welcome, Editor!",
    user: req.auth!.preferred_username,
    roles: req.auth!.roles,
    data: {
      drafts: 3,
      published: 12,
      pending_review: 2,
    },
  });
});

// Role-protected route — requires "manager" role
app.get("/api/manager/reports", auth, requireRoles("manager"), (req, res) => {
  res.json({
    message: "Manager Reports",
    user: req.auth!.preferred_username,
    roles: req.auth!.roles,
    reports: [
      { name: "Q1 Revenue", status: "complete" },
      { name: "User Growth", status: "in_progress" },
    ],
  });
});

app.listen(PORT, () => {
  console.log(`Sample backend running on http://localhost:${PORT}`);
  console.log(`Rampart issuer: ${RAMPART_ISSUER}`);
  console.log(`\nRoutes:`);
  console.log(`  GET /api/health            — public`);
  console.log(`  GET /api/profile           — protected (any authenticated user)`);
  console.log(`  GET /api/claims            — protected (any authenticated user)`);
  console.log(`  GET /api/editor/dashboard  — protected (requires "editor" role)`);
  console.log(`  GET /api/manager/reports   — protected (requires "manager" role)`);
});
