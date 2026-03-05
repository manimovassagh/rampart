"""Tests for Rampart Python middleware."""

from __future__ import annotations

import json
import time
from dataclasses import asdict
from typing import Any, Dict

import jwt as pyjwt
import pytest
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa

from rampart.middleware import RampartAuth, RampartClaims

# ---------------------------------------------------------------------------
# Helpers: generate RSA keys and JWKS for testing
# ---------------------------------------------------------------------------

ISSUER = "https://auth.example.com"


def _generate_rsa_keypair():
    """Generate an RSA private key and its public key."""
    private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    return private_key


def _private_key_pem(private_key) -> bytes:
    return private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )


def _build_jwks_response(private_key) -> Dict[str, Any]:
    """Build a JWKS JSON response from an RSA private key."""
    from jwt.algorithms import RSAAlgorithm

    public_key = private_key.public_key()
    jwk_dict = json.loads(RSAAlgorithm.to_jwk(public_key))
    jwk_dict["kid"] = "test-key-1"
    jwk_dict["use"] = "sig"
    jwk_dict["alg"] = "RS256"
    return {"keys": [jwk_dict]}


def _sign_token(private_key, claims: Dict[str, Any]) -> str:
    """Sign a JWT with the given private key."""
    return pyjwt.encode(
        claims,
        _private_key_pem(private_key),
        algorithm="RS256",
        headers={"kid": "test-key-1"},
    )


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture()
def rsa_key():
    return _generate_rsa_keypair()


@pytest.fixture()
def jwks_json(rsa_key):
    return _build_jwks_response(rsa_key)


@pytest.fixture()
def valid_claims() -> Dict[str, Any]:
    now = int(time.time())
    return {
        "sub": "user-123",
        "iss": ISSUER,
        "iat": now,
        "exp": now + 3600,
        "org_id": "org-456",
        "preferred_username": "jdoe",
        "email": "jdoe@example.com",
        "email_verified": True,
        "given_name": "Jane",
        "family_name": "Doe",
        "roles": ["admin", "user"],
    }


# ---------------------------------------------------------------------------
# Unit tests for RampartClaims
# ---------------------------------------------------------------------------


class TestRampartClaims:
    def test_defaults(self):
        claims = RampartClaims(sub="u1", iss=ISSUER, iat=0, exp=0)
        assert claims.roles == []
        assert claims.org_id is None
        assert claims.email is None

    def test_full_claims(self, valid_claims):
        claims = RampartClaims(**valid_claims)
        assert claims.sub == "user-123"
        assert claims.roles == ["admin", "user"]
        assert claims.email_verified is True

    def test_asdict(self, valid_claims):
        claims = RampartClaims(**valid_claims)
        d = asdict(claims)
        assert d["sub"] == "user-123"
        assert isinstance(d["roles"], list)


# ---------------------------------------------------------------------------
# Unit tests for RampartAuth.verify_token
# ---------------------------------------------------------------------------


class TestRampartAuthVerifyToken:
    def test_verify_valid_token(self, rsa_key, jwks_json, valid_claims, responses_mock):
        """A correctly signed token with valid claims should verify successfully."""
        responses_mock.get(
            f"{ISSUER}/.well-known/jwks.json",
            json=jwks_json,
        )
        auth = RampartAuth(issuer=ISSUER)
        token = _sign_token(rsa_key, valid_claims)
        result = auth.verify_token(token)

        assert result.sub == "user-123"
        assert result.iss == ISSUER
        assert result.roles == ["admin", "user"]
        assert result.email == "jdoe@example.com"

    def test_expired_token_raises(self, rsa_key, jwks_json, valid_claims, responses_mock):
        """An expired token should raise ExpiredSignatureError."""
        responses_mock.get(f"{ISSUER}/.well-known/jwks.json", json=jwks_json)
        auth = RampartAuth(issuer=ISSUER)

        valid_claims["exp"] = int(time.time()) - 3600
        token = _sign_token(rsa_key, valid_claims)

        with pytest.raises(pyjwt.ExpiredSignatureError):
            auth.verify_token(token)

    def test_wrong_issuer_raises(self, rsa_key, jwks_json, valid_claims, responses_mock):
        """A token with a different issuer should be rejected."""
        responses_mock.get(f"{ISSUER}/.well-known/jwks.json", json=jwks_json)
        auth = RampartAuth(issuer=ISSUER)

        valid_claims["iss"] = "https://evil.example.com"
        token = _sign_token(rsa_key, valid_claims)

        with pytest.raises(pyjwt.InvalidIssuerError):
            auth.verify_token(token)

    def test_roles_as_string_normalized(self, rsa_key, jwks_json, valid_claims, responses_mock):
        """If roles is a single string, it should be wrapped in a list."""
        responses_mock.get(f"{ISSUER}/.well-known/jwks.json", json=jwks_json)
        auth = RampartAuth(issuer=ISSUER)

        valid_claims["roles"] = "admin"
        token = _sign_token(rsa_key, valid_claims)
        result = auth.verify_token(token)

        assert result.roles == ["admin"]

    def test_missing_optional_claims(self, rsa_key, jwks_json, responses_mock):
        """Tokens without optional claims should still verify."""
        responses_mock.get(f"{ISSUER}/.well-known/jwks.json", json=jwks_json)
        auth = RampartAuth(issuer=ISSUER)

        now = int(time.time())
        minimal_claims = {
            "sub": "user-minimal",
            "iss": ISSUER,
            "iat": now,
            "exp": now + 3600,
        }
        token = _sign_token(rsa_key, minimal_claims)
        result = auth.verify_token(token)

        assert result.sub == "user-minimal"
        assert result.roles == []
        assert result.email is None
        assert result.org_id is None


# ---------------------------------------------------------------------------
# Pytest fixture for `responses` library
# ---------------------------------------------------------------------------


@pytest.fixture()
def responses_mock():
    """Activate the responses library to mock HTTP requests."""
    import responses

    with responses.RequestsMock() as rsps:
        yield rsps
