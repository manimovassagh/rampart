---
name: rampart-python-setup
description: Add Rampart authentication to a Python app (FastAPI or Flask). Sets up JWT verification middleware, dependency injection for auth claims, and route protection. Use when securing a Python backend with Rampart.
argument-hint: [issuer-url]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Add Rampart Authentication to a Python App

Set up JWT-based authentication with a Rampart IAM server in FastAPI or Flask.

## What This Skill Does

1. Installs `PyJWT` and `cryptography` for JWT verification
2. Creates auth middleware that validates Rampart JWTs
3. Fetches JWKS from Rampart for key verification
4. Provides typed user claims as a dependency/decorator
5. Returns Rampart-compatible 401 JSON errors

## Step-by-Step

### 1. Detect the framework

Check if the project uses FastAPI or Flask. Look for `from fastapi import` or `from flask import` in imports. Proceed with the matching framework below.

---

## FastAPI Setup

### 2. Install dependencies

```bash
pip install PyJWT[crypto] httpx
# or add to requirements.txt / pyproject.toml
```

### 3. Create the auth module

Create `auth/rampart.py`:

```python
from typing import Optional
from dataclasses import dataclass
import httpx
import jwt
from jwt import PyJWKClient
from fastapi import Request, HTTPException, Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials

ISSUER = "$ARGUMENTS" or "http://localhost:8080"
JWKS_URL = f"{ISSUER}/.well-known/jwks.json"

_jwk_client = PyJWKClient(JWKS_URL)
_bearer = HTTPBearer()


@dataclass
class RampartUser:
    sub: str
    email: str
    preferred_username: str
    org_id: str
    email_verified: bool = False
    given_name: Optional[str] = None
    family_name: Optional[str] = None


def _verify_token(token: str) -> dict:
    signing_key = _jwk_client.get_signing_key_from_jwt(token)
    return jwt.decode(
        token,
        signing_key.key,
        algorithms=["RS256"],
        issuer=ISSUER,
        options={"require": ["sub", "iss", "exp", "iat"]},
    )


async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(_bearer),
) -> RampartUser:
    try:
        claims = _verify_token(credentials.credentials)
    except jwt.ExpiredSignatureError:
        raise HTTPException(status_code=401, detail="Token expired.")
    except jwt.InvalidTokenError:
        raise HTTPException(status_code=401, detail="Invalid or expired access token.")

    return RampartUser(
        sub=claims["sub"],
        email=claims.get("email", ""),
        preferred_username=claims.get("preferred_username", ""),
        org_id=claims.get("org_id", ""),
        email_verified=claims.get("email_verified", False),
        given_name=claims.get("given_name"),
        family_name=claims.get("family_name"),
    )
```

### 4. Protect FastAPI routes

```python
from fastapi import FastAPI, Depends
from auth.rampart import get_current_user, RampartUser

app = FastAPI()

@app.get("/api/profile")
async def profile(user: RampartUser = Depends(get_current_user)):
    return {
        "id": user.sub,
        "email": user.email,
        "username": user.preferred_username,
        "org": user.org_id,
    }

@app.get("/api/admin")
async def admin_only(user: RampartUser = Depends(get_current_user)):
    # Add role checks here
    return {"message": f"Hello admin {user.preferred_username}"}
```

---

## Flask Setup

### 2. Install dependencies

```bash
pip install PyJWT[crypto] requests
```

### 3. Create the auth module

Create `auth/rampart.py`:

```python
from functools import wraps
from dataclasses import dataclass
from typing import Optional
import jwt
from jwt import PyJWKClient
from flask import request, jsonify, g

ISSUER = "$ARGUMENTS" or "http://localhost:8080"
JWKS_URL = f"{ISSUER}/.well-known/jwks.json"

_jwk_client = PyJWKClient(JWKS_URL)


@dataclass
class RampartUser:
    sub: str
    email: str
    preferred_username: str
    org_id: str
    email_verified: bool = False
    given_name: Optional[str] = None
    family_name: Optional[str] = None


def require_auth(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        auth_header = request.headers.get("Authorization", "")
        if not auth_header.startswith("Bearer "):
            return jsonify(error="unauthorized", error_description="Missing authorization header.", status=401), 401

        token = auth_header[7:]
        try:
            signing_key = _jwk_client.get_signing_key_from_jwt(token)
            claims = jwt.decode(token, signing_key.key, algorithms=["RS256"], issuer=ISSUER)
        except jwt.ExpiredSignatureError:
            return jsonify(error="unauthorized", error_description="Token expired.", status=401), 401
        except jwt.InvalidTokenError:
            return jsonify(error="unauthorized", error_description="Invalid or expired access token.", status=401), 401

        g.user = RampartUser(
            sub=claims["sub"],
            email=claims.get("email", ""),
            preferred_username=claims.get("preferred_username", ""),
            org_id=claims.get("org_id", ""),
            email_verified=claims.get("email_verified", False),
            given_name=claims.get("given_name"),
            family_name=claims.get("family_name"),
        )
        return f(*args, **kwargs)
    return decorated
```

### 4. Protect Flask routes

```python
from flask import Flask, jsonify, g
from auth.rampart import require_auth

app = Flask(__name__)

@app.get("/api/profile")
@require_auth
def profile():
    return jsonify(id=g.user.sub, email=g.user.email, username=g.user.preferred_username)
```

---

## Environment Variables

```bash
RAMPART_ISSUER=http://localhost:8080
```

If `$ARGUMENTS` is empty, use `os.environ.get("RAMPART_ISSUER", "http://localhost:8080")` in the code.

## Checklist

- [ ] `PyJWT[crypto]` installed
- [ ] Auth module created with JWKS verification
- [ ] Protected routes use dependency injection (FastAPI) or decorator (Flask)
- [ ] 401 errors return Rampart-compatible JSON
- [ ] Issuer URL configured via environment variable
