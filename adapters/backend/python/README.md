# Rampart Python Middleware

JWT verification middleware for [Rampart](https://github.com/manimovassagh/rampart) IAM server. Supports **FastAPI** and **Flask**.

## Installation

```bash
# Core (PyJWT + cryptography)
pip install rampart-python

# With FastAPI support
pip install rampart-python[fastapi]

# With Flask support
pip install rampart-python[flask]
```

## FastAPI

### Basic Authentication

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

### Role-Based Access Control

```python
from rampart.fastapi import rampart_auth, require_roles_from_claims

auth = rampart_auth("https://auth.example.com")
check_admin = require_roles_from_claims("admin")

@app.get("/admin/users")
async def list_users(claims: RampartClaims = Depends(auth)):
    check_admin(claims)  # Raises 403 if "admin" role is missing
    return {"users": ["..."]}
```

## Flask

### Basic Authentication

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

### Role-Based Access Control

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

## Configuration Options

```python
RampartAuth(
    issuer="https://auth.example.com",  # Required: Rampart server URL
    audience="my-api",                   # Optional: expected audience claim
    jwks_cache_ttl=300,                  # JWKS cache lifetime in seconds (default: 300)
    algorithms=["RS256"],                # Allowed JWT algorithms (default: ["RS256"])
)
```

## Running Tests

```bash
pip install -e ".[dev]"
pytest tests/
```
