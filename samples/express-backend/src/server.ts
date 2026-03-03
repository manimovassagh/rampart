import express from "express";
import cors from "cors";
import { rampartAuth } from "@rampart/node";

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
    },
  });
});

// Protected route — returns all raw claims
app.get("/api/claims", auth, (req, res) => {
  res.json(req.auth);
});

app.listen(PORT, () => {
  console.log(`Sample backend running on http://localhost:${PORT}`);
  console.log(`Rampart issuer: ${RAMPART_ISSUER}`);
  console.log(`\nRoutes:`);
  console.log(`  GET /api/health   — public`);
  console.log(`  GET /api/profile  — protected (requires Bearer token)`);
  console.log(`  GET /api/claims   — protected (raw JWT claims)`);
});
