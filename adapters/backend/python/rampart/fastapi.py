"""FastAPI integration for Rampart JWT verification."""

from __future__ import annotations

from typing import Callable, Optional

import jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from rampart.middleware import RampartAuth, RampartClaims

_bearer_scheme = HTTPBearer(auto_error=True)


def rampart_auth(
    issuer: str,
    audience: Optional[str] = None,
) -> Callable[..., RampartClaims]:
    """Create a FastAPI dependency that verifies Rampart JWT tokens.

    Usage::

        auth = rampart_auth("https://auth.example.com")

        @app.get("/protected")
        def protected(claims: RampartClaims = Depends(auth)):
            return {"user": claims.sub}

    Args:
        issuer: Base URL of the Rampart server.
        audience: Expected audience claim (optional).

    Returns:
        A FastAPI dependency that returns RampartClaims.
    """
    verifier = RampartAuth(issuer=issuer, audience=audience)

    async def _dependency(
        credentials: HTTPAuthorizationCredentials = Depends(_bearer_scheme),
    ) -> RampartClaims:
        token = credentials.credentials
        try:
            return verifier.verify_token(token)
        except jwt.ExpiredSignatureError:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Token has expired",
                headers={"WWW-Authenticate": "Bearer"},
            )
        except jwt.InvalidTokenError as exc:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Invalid token: {exc}",
                headers={"WWW-Authenticate": "Bearer"},
            )

    return _dependency


def require_roles(*roles: str) -> Callable[..., None]:
    """Create a role checker that takes RampartClaims directly as a parameter.

    Usage::

        auth = rampart_auth("https://auth.example.com")
        check_admin = require_roles("admin")

        @app.get("/admin")
        def admin_only(claims: RampartClaims = Depends(auth)):
            check_admin(claims)
            return {"admin": claims.sub}
    """
    required = set(roles)

    def _check(claims: RampartClaims) -> None:
        user_roles = set(claims.roles)
        missing = required - user_roles
        if missing:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail=f"Missing required roles: {', '.join(sorted(missing))}",
            )

    return _check
