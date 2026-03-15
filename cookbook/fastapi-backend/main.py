"""FastAPI sample backend for Rampart — drop-in replacement for the Express backend."""

from __future__ import annotations

import os
import sys
from dataclasses import asdict

from fastapi import Depends, FastAPI, HTTPException, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

# ---------------------------------------------------------------------------
# Allow importing the rampart adapter from the local source tree so that
# `pip install rampart-python` is not strictly required during development.
# ---------------------------------------------------------------------------
_ADAPTER_PATH = os.path.join(
    os.path.dirname(__file__), "..", "..", "adapters", "backend", "python"
)
if os.path.isdir(_ADAPTER_PATH):
    sys.path.insert(0, os.path.abspath(_ADAPTER_PATH))

from rampart import RampartClaims  # noqa: E402
from rampart.fastapi import rampart_auth, require_roles  # noqa: E402

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
RAMPART_ISSUER = os.environ.get("RAMPART_ISSUER", "http://localhost:8080")
PORT = int(os.environ.get("PORT", "3001"))

# ---------------------------------------------------------------------------
# App & middleware
# ---------------------------------------------------------------------------
app = FastAPI(title="Rampart FastAPI Sample Backend")


# ---------------------------------------------------------------------------
# Custom error handler to match Express error response format
# ---------------------------------------------------------------------------
@app.exception_handler(HTTPException)
async def custom_http_exception_handler(request: Request, exc: HTTPException):
    """Return error responses in the same format as the Express backend."""
    if exc.status_code == 401:
        return JSONResponse(
            status_code=401,
            content={
                "error": "unauthorized",
                "error_description": exc.detail or "Missing authorization header.",
                "status": 401,
            },
        )
    if exc.status_code == 403:
        return JSONResponse(
            status_code=403,
            content={
                "error": "forbidden",
                "error_description": exc.detail.replace("Missing required roles:", "Missing required role(s):") if exc.detail else "Forbidden",
                "status": 403,
            },
        )
    return JSONResponse(
        status_code=exc.status_code,
        content={"error": str(exc.status_code), "error_description": exc.detail, "status": exc.status_code},
    )


app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# ---------------------------------------------------------------------------
# Auth dependency
# ---------------------------------------------------------------------------
auth = rampart_auth(RAMPART_ISSUER)

# Role checkers
check_editor = require_roles("editor")
check_manager = require_roles("manager")

# ---------------------------------------------------------------------------
# Routes
# ---------------------------------------------------------------------------


@app.get("/api/health")
async def health():
    """Public health-check endpoint."""
    return {"status": "ok", "issuer": RAMPART_ISSUER}


@app.get("/api/profile")
async def profile(claims: RampartClaims = Depends(auth)):
    """Protected — returns the authenticated user's profile."""
    return {
        "message": "Authenticated!",
        "user": {
            "id": claims.sub,
            "email": claims.email,
            "username": claims.preferred_username,
            "org_id": claims.org_id,
            "email_verified": claims.email_verified if claims.email_verified is not None else False,
            "given_name": claims.given_name or "",
            "family_name": claims.family_name or "",
            "roles": claims.roles if claims.roles else [],
        },
    }


@app.get("/api/claims")
async def get_claims(claims: RampartClaims = Depends(auth)):
    """Protected — returns all raw JWT claims."""
    return asdict(claims)


@app.get("/api/editor/dashboard")
async def editor_dashboard(claims: RampartClaims = Depends(auth)):
    """Protected — requires the 'editor' role."""
    check_editor(claims)
    return {
        "message": "Welcome, Editor!",
        "user": claims.preferred_username,
        "roles": claims.roles,
        "data": {
            "drafts": 3,
            "published": 12,
            "pending_review": 2,
        },
    }


@app.get("/api/manager/reports")
async def manager_reports(claims: RampartClaims = Depends(auth)):
    """Protected — requires the 'manager' role."""
    check_manager(claims)
    return {
        "message": "Manager Reports",
        "user": claims.preferred_username,
        "roles": claims.roles,
        "reports": [
            {"name": "Q1 Revenue", "status": "complete"},
            {"name": "User Growth", "status": "in_progress"},
        ],
    }


# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    import uvicorn

    print(f"Sample backend running on http://localhost:{PORT}")
    print(f"Rampart issuer: {RAMPART_ISSUER}")
    print()
    print("Routes:")
    print('  GET /api/health            — public')
    print('  GET /api/profile           — protected (any authenticated user)')
    print('  GET /api/claims            — protected (any authenticated user)')
    print('  GET /api/editor/dashboard  — protected (requires "editor" role)')
    print('  GET /api/manager/reports   — protected (requires "manager" role)')

    uvicorn.run(app, host="0.0.0.0", port=PORT)
