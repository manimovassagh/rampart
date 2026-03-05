"""Flask integration for Rampart JWT verification."""

from __future__ import annotations

import functools
from typing import Any, Callable, Optional, TypeVar

import jwt as pyjwt
from flask import g, jsonify, request

from rampart.middleware import RampartAuth, RampartClaims

F = TypeVar("F", bound=Callable[..., Any])


def rampart_auth(issuer: str, audience: Optional[str] = None) -> Callable[[F], F]:
    """Flask decorator that verifies Rampart JWT tokens.

    On success, sets ``flask.g.auth`` to the verified ``RampartClaims``.

    Usage::

        from rampart.flask import rampart_auth

        @app.route("/protected")
        @rampart_auth("https://auth.example.com")
        def protected():
            return {"user": g.auth.sub}

    Args:
        issuer: Base URL of the Rampart server.
        audience: Expected audience claim (optional).

    Returns:
        A decorator for Flask view functions.
    """
    verifier = RampartAuth(issuer=issuer, audience=audience)

    def decorator(f: F) -> F:
        @functools.wraps(f)
        def wrapper(*args: Any, **kwargs: Any) -> Any:
            auth_header = request.headers.get("Authorization", "")
            if not auth_header.startswith("Bearer "):
                return jsonify({"detail": "Missing or invalid Authorization header"}), 401

            token = auth_header[7:]  # Strip "Bearer "

            try:
                claims = verifier.verify_token(token)
            except pyjwt.ExpiredSignatureError:
                return jsonify({"detail": "Token has expired"}), 401
            except pyjwt.InvalidTokenError as exc:
                return jsonify({"detail": f"Invalid token: {exc}"}), 401

            g.auth = claims
            return f(*args, **kwargs)

        return wrapper  # type: ignore[return-value]

    return decorator


def require_roles(*roles: str) -> Callable[[F], F]:
    """Flask decorator that checks the user has ALL specified roles.

    Must be applied after ``rampart_auth`` so that ``g.auth`` is set.

    Usage::

        @app.route("/admin")
        @rampart_auth("https://auth.example.com")
        @require_roles("admin")
        def admin_only():
            return {"admin": g.auth.sub}

    Args:
        *roles: Role names that the token must contain.

    Returns:
        A decorator for Flask view functions.
    """
    required = set(roles)

    def decorator(f: F) -> F:
        @functools.wraps(f)
        def wrapper(*args: Any, **kwargs: Any) -> Any:
            claims: Optional[RampartClaims] = getattr(g, "auth", None)
            if claims is None:
                return jsonify({"detail": "Authentication required before role check"}), 401

            user_roles = set(claims.roles)
            missing = required - user_roles
            if missing:
                return jsonify(
                    {"detail": f"Missing required roles: {', '.join(sorted(missing))}"}
                ), 403

            return f(*args, **kwargs)

        return wrapper  # type: ignore[return-value]

    return decorator
