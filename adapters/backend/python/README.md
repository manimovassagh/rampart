# Rampart Python Middleware

[![PyPI version](https://img.shields.io/pypi/v/rampart-python.svg)](https://pypi.org/project/rampart-python/)
[![Python versions](https://img.shields.io/pypi/pyversions/rampart-python.svg)](https://pypi.org/project/rampart-python/)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/manimovassagh/rampart/blob/main/adapters/backend/python/LICENSE)

JWT verification middleware for [Rampart](https://github.com/manimovassagh/rampart) IAM server. Supports **FastAPI** and **Flask** out of the box, or use the core library directly with any framework.

## Features

- **JWT verification** -- validates RS256-signed tokens against Rampart's JWKS endpoint
- **FastAPI integration** -- dependency-injection-based auth via `Depends()`
- **Flask integration** -- decorator-based auth with `@rampart_auth`
- **Role-based access control (RBAC)** -- enforce required roles with a single function call
- **JWKS caching** -- automatic key caching with configurable TTL to minimize network calls
- **Typed claims** -- verified tokens return a `RampartClaims` dataclass with full type hints
- **Framework-agnostic core** -- use `RampartAuth` directly for custom integrations
- **Consistent error responses** -- `401`/`403` JSON errors across both frameworks

## Requirements

- Python 3.9 or later
- A running [Rampart](https://github.com/manimovassagh/rampart) IAM server (for JWKS/token verification)

## Installation

```bash
# Core (PyJWT + cryptography)
pip install rampart-python

# With FastAPI support
pip install "rampart-python[fastapi]"

# With Flask support
pip install "rampart-python[flask]"
```

## Quick Start

### FastAPI

```python
from fastapi import Depends, FastAPI
from rampart import RampartClaims
from rampart.fastapi import rampart_auth

app = FastAPI()
auth = rampart_auth("https://auth.example.com")

@app.get("/me")
async def me(claims: RampartClaims = Depends(auth)):
    return {
        "user_id": claims.sub,
        "email": claims.email,
        "roles": claims.roles,
    }
```

### Flask

```python
from flask import Flask, g
from rampart.flask import rampart_auth

app = Flask(__name__)

@app.route("/me")
@rampart_auth("https://auth.example.com")
def me():
    return {
        "user_id": g.auth.sub,
        "email": g.auth.email,
        "roles": g.auth.roles,
    }
```

## Role-Based Access Control

### FastAPI

```python
from rampart.fastapi import rampart_auth, require_roles

auth = rampart_auth("https://auth.example.com")
check_admin = require_roles("admin")

@app.get("/admin/users")
async def list_users(claims: RampartClaims = Depends(auth)):
    check_admin(claims)  # Raises 403 if "admin" role is missing
    return {"users": ["..."]}
```

### Flask

```python
from rampart.flask import rampart_auth, require_roles

@app.route("/admin/users")
@rampart_auth("https://auth.example.com")
@require_roles("admin")
def list_users():
    return {"users": ["..."]}
```

## Direct Usage (No Framework)

```python
from rampart import RampartAuth

auth = RampartAuth(issuer="https://auth.example.com")
claims = auth.verify_token(raw_jwt_string)

print(claims.sub)       # "user-123"
print(claims.email)     # "user@example.com"
print(claims.roles)     # ["admin", "user"]
print(claims.org_id)    # "org-456"
```

## Configuration Options

```python
RampartAuth(
    issuer="https://auth.example.com",  # Required: Rampart server URL
    audience="my-api",                   # Optional: expected audience claim
    jwks_cache_ttl=300,                  # JWKS cache lifetime in seconds (default: 300)
    algorithms=["RS256"],                # Allowed JWT algorithms (default: ["RS256"])
)
```

## Claims

Verified tokens return a `RampartClaims` dataclass:

| Field                | Type           | Description                    |
|----------------------|----------------|--------------------------------|
| `sub`                | `str`          | Subject (user ID)              |
| `iss`                | `str`          | Issuer URL                     |
| `iat`                | `int`          | Issued-at timestamp            |
| `exp`                | `int`          | Expiration timestamp           |
| `org_id`             | `str | None`   | Organization / tenant ID       |
| `preferred_username` | `str | None`   | Username                       |
| `email`              | `str | None`   | Email address                  |
| `email_verified`     | `bool | None`  | Whether email is verified      |
| `given_name`         | `str | None`   | First name                     |
| `family_name`        | `str | None`   | Last name                      |
| `roles`              | `list[str]`    | Assigned roles                 |

## Error Responses

On authentication failure, both FastAPI and Flask return a `401` JSON response:

```json
{
  "detail": "Token has expired"
}
```

On authorization failure (missing roles), returns `403`:

```json
{
  "detail": "Missing required roles: admin, editor"
}
```

**401 error messages:**

- `"Missing or invalid Authorization header"` -- no `Authorization: Bearer` header (Flask)
- `"Token has expired"` -- the JWT expiration (`exp`) has passed
- `"Invalid token: <reason>"` -- signature, issuer, or other validation failed
**403 error messages:**

- `"Missing required roles: <role1>, <role2>"` -- the token lacks one or more required roles

> **Note:** FastAPI uses `HTTPException` which returns `{"detail": "..."}`.
> Flask returns the same shape via `jsonify({"detail": "..."})` for consistency.

## Running Tests

```bash
pip install -e ".[dev]"
pytest tests/
```

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/manimovassagh/rampart/blob/main/adapters/backend/python/LICENSE) file for details.

## Links

- [Rampart IAM Server](https://github.com/manimovassagh/rampart)
- [PyPI Package](https://pypi.org/project/rampart-python/)
- [Issue Tracker](https://github.com/manimovassagh/rampart/issues)
