require("dotenv").config();

const express = require("express");
const cors = require("cors");
const jwt = require("jsonwebtoken");
const jwksRsa = require("jwks-rsa");

const app = express();

const PORT = process.env.PORT || 3001;
const RAMPART_URL = process.env.RAMPART_URL || "http://localhost:8080";
const RAMPART_ISSUER = process.env.RAMPART_ISSUER || "http://localhost:8080";

// ---------------------------------------------------------------------------
// CORS - allow the React frontend on localhost:5173
// ---------------------------------------------------------------------------
app.use(
  cors({
    origin: "http://localhost:5173",
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    allowedHeaders: ["Content-Type", "Authorization"],
  })
);

app.use(express.json());

// ---------------------------------------------------------------------------
// JWKS client - fetches signing keys from Rampart with caching & rate limiting
// ---------------------------------------------------------------------------
const jwksClient = jwksRsa({
  jwksUri: `${RAMPART_URL}/.well-known/jwks.json`,
  cache: true,
  cacheMaxEntries: 5,
  cacheMaxAge: 600000, // 10 minutes
  rateLimit: true,
  jwksRequestsPerMinute: 10,
});

/**
 * Retrieve the signing key that matches the JWT's "kid" header.
 */
function getSigningKey(header, callback) {
  jwksClient.getSigningKey(header.kid, (err, key) => {
    if (err) {
      return callback(err);
    }
    const signingKey = key.getPublicKey();
    callback(null, signingKey);
  });
}

// ---------------------------------------------------------------------------
// Middleware: JWT validation
// ---------------------------------------------------------------------------
function requireAuth(req, res, next) {
  const authHeader = req.headers.authorization;

  if (!authHeader || !authHeader.startsWith("Bearer ")) {
    return res.status(401).json({
      error: "Unauthorized",
      message: "Missing or malformed Authorization header. Expected: Bearer <token>",
    });
  }

  const token = authHeader.split(" ")[1];

  jwt.verify(
    token,
    getSigningKey,
    {
      issuer: RAMPART_ISSUER,
      algorithms: ["RS256"],
    },
    (err, decoded) => {
      if (err) {
        const message =
          err.name === "TokenExpiredError"
            ? "Token has expired"
            : "Invalid token";

        return res.status(401).json({
          error: "Unauthorized",
          message,
        });
      }

      // Attach user claims to the request object
      req.user = {
        sub: decoded.sub,
        email: decoded.email,
        roles: decoded.roles || [],
        org_id: decoded.org_id,
      };

      next();
    }
  );
}

// ---------------------------------------------------------------------------
// Middleware: Role checking (must be used after requireAuth)
// ---------------------------------------------------------------------------
function requireRole(role) {
  return (req, res, next) => {
    if (!req.user || !Array.isArray(req.user.roles) || !req.user.roles.includes(role)) {
      return res.status(403).json({
        error: "Forbidden",
        message: `This endpoint requires the "${role}" role`,
      });
    }
    next();
  };
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

// Health check - no auth required
app.get("/api/health", (_req, res) => {
  res.json({ status: "ok" });
});

// User profile - requires valid JWT
app.get("/api/user/profile", requireAuth, (req, res) => {
  res.json({
    sub: req.user.sub,
    email: req.user.email,
    roles: req.user.roles,
    org_id: req.user.org_id,
  });
});

// Admin dashboard data - requires valid JWT + "admin" role
app.get("/api/admin/data", requireAuth, requireRole("admin"), (_req, res) => {
  res.json({
    userCount: 142,
    activeSessionCount: 37,
    recentEvents: [
      { type: "user.registered", email: "alice@example.com", timestamp: "2026-03-07T10:23:00Z" },
      { type: "user.login", email: "bob@example.com", timestamp: "2026-03-07T10:18:00Z" },
      { type: "user.password_changed", email: "carol@example.com", timestamp: "2026-03-07T09:45:00Z" },
      { type: "user.role_updated", email: "dave@example.com", timestamp: "2026-03-07T09:30:00Z" },
    ],
    systemHealth: {
      database: "healthy",
      cache: "healthy",
      authProvider: "healthy",
    },
  });
});

// ---------------------------------------------------------------------------
// Start server
// ---------------------------------------------------------------------------
app.listen(PORT, () => {
  console.log(`Rampart example API running on http://localhost:${PORT}`);
  console.log(`JWKS endpoint: ${RAMPART_URL}/.well-known/jwks.json`);
  console.log(`Expected issuer: ${RAMPART_ISSUER}`);
});
