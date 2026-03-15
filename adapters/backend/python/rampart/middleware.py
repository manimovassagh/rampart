"""Core JWT verification logic for Rampart."""

from __future__ import annotations

import threading
import time
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional

import jwt
import requests
from jwt import PyJWKClient


@dataclass
class RampartClaims:
    """Verified JWT claims from a Rampart token."""

    sub: str
    iss: str
    iat: int
    exp: int
    org_id: Optional[str] = None
    preferred_username: Optional[str] = None
    email: Optional[str] = None
    email_verified: Optional[bool] = None
    given_name: Optional[str] = None
    family_name: Optional[str] = None
    roles: List[str] = field(default_factory=list)


class RampartAuth:
    """JWT verification client that fetches JWKS from a Rampart server.

    Args:
        issuer: Base URL of the Rampart server (e.g. "https://auth.example.com").
        audience: Expected audience claim. If None, audience validation is skipped.
        jwks_cache_ttl: How long (in seconds) to cache the JWKS keys. Default: 300.
        algorithms: Allowed signing algorithms. Default: ["RS256"].
    """

    def __init__(
        self,
        issuer: str,
        audience: Optional[str] = None,
        jwks_cache_ttl: int = 300,
        algorithms: Optional[List[str]] = None,
    ) -> None:
        self.issuer = issuer.rstrip("/")
        self.audience = audience
        self.algorithms = algorithms or ["RS256"]
        self._jwks_cache_ttl = jwks_cache_ttl
        self._jwks_uri = f"{self.issuer}/.well-known/jwks.json"

        self._jwk_client = PyJWKClient(self._jwks_uri, cache_jwk_set=True, lifespan=jwks_cache_ttl)
        self._lock = threading.Lock()

    def verify_token(self, token: str) -> RampartClaims:
        """Verify a JWT token and return parsed claims.

        Args:
            token: The raw JWT string (without "Bearer " prefix).

        Returns:
            RampartClaims with the verified token's claims.

        Raises:
            jwt.ExpiredSignatureError: If the token has expired.
            jwt.InvalidTokenError: If the token is invalid for any reason.
        """
        signing_key = self._jwk_client.get_signing_key_from_jwt(token)

        decode_options: Dict[str, Any] = {}
        if self.audience is None:
            decode_options["verify_aud"] = False

        payload = jwt.decode(
            token,
            signing_key.key,
            algorithms=self.algorithms,
            issuer=self.issuer,
            audience=self.audience,
            options=decode_options,
        )

        roles = payload.get("roles", [])
        if isinstance(roles, str):
            roles = [roles]

        sub = payload.get("sub", "")
        if not sub:
            raise jwt.InvalidTokenError("Token is missing required 'sub' claim")

        return RampartClaims(
            sub=sub,
            iss=payload["iss"],
            iat=payload.get("iat", 0),
            exp=payload.get("exp", 0),
            org_id=payload.get("org_id"),
            preferred_username=payload.get("preferred_username"),
            email=payload.get("email"),
            email_verified=payload.get("email_verified"),
            given_name=payload.get("given_name"),
            family_name=payload.get("family_name"),
            roles=roles,
        )
