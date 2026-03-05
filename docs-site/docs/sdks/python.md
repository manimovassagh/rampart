---
sidebar_position: 6
title: Python
description: Integrate Rampart authentication into Python applications with FastAPI and Flask adapters.
---

# Python Adapter

The `rampart-python` adapter provides authentication middleware for Python web applications. It supports FastAPI (via dependency injection) and Flask (via decorators), handling JWKS-based JWT verification, claims extraction, and role-based access control.

## Installation

```bash
pip install rampart-python
```

```bash
poetry add rampart-python
```

The package depends on `PyJWT[crypto]` for JWT verification and `httpx` for JWKS fetching.

## FastAPI

### Quick Start

```python
from fastapi import FastAPI, Depends
from rampart import RampartAuth, Claims

app = FastAPI()

auth = RampartAuth(
    issuer_url="https://auth.example.com",
    audience="my-api",
)

@app.get("/health")
async def health():
    return {"status": "ok"}

@app.get("/api/profile")
async def profile(claims: Claims = Depends(auth.require_auth)):
    return {
        "user_id": claims.sub,
        "email": claims.email,
        "roles": claims.roles,
    }
```

### Configuration

```python
auth = RampartAuth(
    # Required
    issuer_url="https://auth.example.com",
    audience="my-api",

    # Optional
    realm="default",                     # Organization/realm
    clock_tolerance=5,                   # Seconds of clock skew tolerance
    jwks_cache_ttl=600,                  # JWKS cache TTL in seconds
    required_claims=["email"],           # Claims that must be present
)
```

#### Configuration from Environment

```python
from rampart import RampartAuth

auth = RampartAuth.from_env()
```

Reads `RAMPART_URL`, `RAMPART_CLIENT_ID`, and `RAMPART_REALM` from the environment.

### Dependency Injection

The adapter provides FastAPI dependencies for common auth patterns:

#### `require_auth`

Verifies the bearer token and returns the decoded claims. Raises `HTTPException(401)` if the token is missing or invalid.

```python
from rampart import Claims

@app.get("/api/data")
async def get_data(claims: Claims = Depends(auth.require_auth)):
    return {"user": claims.sub}
```

#### `require_roles(*roles)`

Requires the user to have all specified roles. Raises `HTTPException(403)` if any role is missing.

```python
@app.delete("/api/admin/users/{user_id}")
async def delete_user(
    user_id: str,
    claims: Claims = Depends(auth.require_roles("admin")),
):
    return {"deleted": user_id}
```

#### `require_scopes(*scopes)`

Requires the token to include all specified scopes.

```python
@app.post("/api/emails/send")
async def send_email(
    claims: Claims = Depends(auth.require_scopes("email:send")),
):
    return {"sent": True}
```

#### `optional_auth`

Verifies the token if present but allows unauthenticated requests. Returns `None` if no token is provided.

```python
from rampart import Claims
from typing import Optional

@app.get("/api/feed")
async def get_feed(claims: Optional[Claims] = Depends(auth.optional_auth)):
    if claims:
        return {"feed": "personalized", "user": claims.sub}
    return {"feed": "public"}
```

### Claims Object

```python
class Claims:
    sub: str                  # User ID
    email: str | None         # Email address
    name: str | None          # Display name
    roles: list[str]          # Assigned roles
    scope: str                # Space-separated scopes
    org_id: str | None        # Organization ID
    iss: str                  # Issuer URL
    aud: str | list[str]      # Audience
    exp: int                  # Expiration timestamp
    iat: int                  # Issued-at timestamp

    def has_role(self, role: str) -> bool: ...
    def has_any_role(self, *roles: str) -> bool: ...
    def has_scope(self, scope: str) -> bool: ...
    def has_all_scopes(self, *scopes: str) -> bool: ...
```

### Full FastAPI Example

```python
import os
from fastapi import FastAPI, Depends, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from rampart import RampartAuth, Claims

app = FastAPI(title="Task API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

auth = RampartAuth(
    issuer_url=os.environ.get("RAMPART_URL", "https://auth.example.com"),
    audience=os.environ.get("RAMPART_CLIENT_ID", "task-api"),
)

# In-memory task store (replace with a real database)
tasks: dict[str, list[dict]] = {}


class TaskCreate(BaseModel):
    title: str


# Public endpoint
@app.get("/health")
async def health():
    return {"status": "ok"}


# List tasks for the authenticated user
@app.get("/api/tasks")
async def list_tasks(claims: Claims = Depends(auth.require_auth)):
    user_tasks = tasks.get(claims.sub, [])
    return {"tasks": user_tasks}


# Create a task
@app.post("/api/tasks", status_code=201)
async def create_task(
    body: TaskCreate,
    claims: Claims = Depends(auth.require_scopes("tasks:write")),
):
    if claims.sub not in tasks:
        tasks[claims.sub] = []

    task = {
        "id": str(len(tasks[claims.sub]) + 1),
        "title": body.title,
        "assignee": claims.sub,
    }
    tasks[claims.sub].append(task)
    return task


# Admin-only: view all tasks
@app.get("/api/admin/tasks")
async def admin_list_tasks(
    claims: Claims = Depends(auth.require_roles("admin")),
):
    all_tasks = [task for user_tasks in tasks.values() for task in user_tasks]
    return {"tasks": all_tasks, "total": len(all_tasks)}


# Admin-only: view stats
@app.get("/api/admin/stats")
async def admin_stats(
    claims: Claims = Depends(auth.require_roles("admin")),
):
    return {
        "total_users": len(tasks),
        "total_tasks": sum(len(t) for t in tasks.values()),
    }
```

Run with:

```bash
RAMPART_URL=https://auth.example.com RAMPART_CLIENT_ID=task-api \
  uvicorn main:app --reload --port 8000
```

## Flask

### Quick Start

```python
from flask import Flask, jsonify
from rampart.flask import RampartAuth, require_auth, require_roles, current_claims

app = Flask(__name__)

auth = RampartAuth(app, {
    "issuer_url": "https://auth.example.com",
    "audience": "my-api",
})

@app.route("/health")
def health():
    return jsonify(status="ok")

@app.route("/api/profile")
@require_auth
def profile():
    return jsonify(
        user_id=current_claims.sub,
        email=current_claims.email,
        roles=current_claims.roles,
    )
```

### Configuration

```python
app.config["RAMPART_ISSUER_URL"] = "https://auth.example.com"
app.config["RAMPART_AUDIENCE"] = "my-api"
app.config["RAMPART_REALM"] = "default"
app.config["RAMPART_CLOCK_TOLERANCE"] = 5
app.config["RAMPART_JWKS_CACHE_TTL"] = 600

auth = RampartAuth(app)
```

### Decorators

#### `@require_auth`

Verifies the bearer token. Returns 401 if missing or invalid. Access the claims via `current_claims`.

```python
from rampart.flask import require_auth, current_claims

@app.route("/api/data")
@require_auth
def get_data():
    return jsonify(user=current_claims.sub)
```

#### `@require_roles(*roles)`

Requires all specified roles.

```python
from rampart.flask import require_roles, current_claims

@app.route("/api/admin/users/<user_id>", methods=["DELETE"])
@require_roles("admin")
def delete_user(user_id):
    return jsonify(deleted=user_id)
```

#### `@require_scopes(*scopes)`

Requires all specified scopes.

```python
from rampart.flask import require_scopes

@app.route("/api/emails/send", methods=["POST"])
@require_scopes("email:send")
def send_email():
    return jsonify(sent=True)
```

#### `@optional_auth`

Verifies the token if present. `current_claims` is `None` when unauthenticated.

```python
from rampart.flask import optional_auth, current_claims

@app.route("/api/feed")
@optional_auth
def get_feed():
    if current_claims:
        return jsonify(feed="personalized", user=current_claims.sub)
    return jsonify(feed="public")
```

### Full Flask Example

```python
import os
from flask import Flask, jsonify, request
from rampart.flask import (
    RampartAuth,
    require_auth,
    require_roles,
    current_claims,
)

app = Flask(__name__)
app.config["RAMPART_ISSUER_URL"] = os.environ.get(
    "RAMPART_URL", "https://auth.example.com"
)
app.config["RAMPART_AUDIENCE"] = os.environ.get("RAMPART_CLIENT_ID", "task-api")

auth = RampartAuth(app)

# In-memory task store
tasks: dict[str, list[dict]] = {}


@app.route("/health")
def health():
    return jsonify(status="ok")


@app.route("/api/tasks")
@require_auth
def list_tasks():
    user_tasks = tasks.get(current_claims.sub, [])
    return jsonify(tasks=user_tasks)


@app.route("/api/tasks", methods=["POST"])
@require_auth
def create_task():
    data = request.get_json()
    if current_claims.sub not in tasks:
        tasks[current_claims.sub] = []

    task = {
        "id": str(len(tasks[current_claims.sub]) + 1),
        "title": data["title"],
        "assignee": current_claims.sub,
    }
    tasks[current_claims.sub].append(task)
    return jsonify(task), 201


@app.route("/api/admin/stats")
@require_roles("admin")
def admin_stats():
    return jsonify(
        total_users=len(tasks),
        total_tasks=sum(len(t) for t in tasks.values()),
    )


if __name__ == "__main__":
    app.run(port=8000, debug=True)
```

Run with:

```bash
RAMPART_URL=https://auth.example.com RAMPART_CLIENT_ID=task-api \
  flask run --port 8000
```

## Token Verification Details

Both the FastAPI and Flask adapters perform identical verification:

1. Extract the token from the `Authorization: Bearer <token>` header
2. Fetch JWKS from `{issuer_url}/.well-known/jwks.json` (cached in memory)
3. Decode and verify the JWT signature using the matching key
4. Validate `iss`, `aud`, `exp`, and `iat` claims
5. Apply clock tolerance for `exp` and `nbf`
6. Verify any `required_claims` are present
7. Return the decoded `Claims` object

Supported algorithms: RS256, RS384, RS512, ES256, ES384, ES512.

## Custom Error Handling

### FastAPI

```python
from rampart import RampartAuthError, TokenExpiredError

@app.exception_handler(RampartAuthError)
async def auth_error_handler(request, exc):
    if isinstance(exc, TokenExpiredError):
        return JSONResponse(
            status_code=401,
            content={"error": "token_expired", "message": "Session expired."},
        )
    return JSONResponse(
        status_code=401,
        content={"error": "unauthorized", "message": "Invalid token."},
    )
```

### Flask

```python
from rampart.flask import RampartAuthError, TokenExpiredError

@app.errorhandler(RampartAuthError)
def handle_auth_error(error):
    if isinstance(error, TokenExpiredError):
        return jsonify(error="token_expired", message="Session expired."), 401
    return jsonify(error="unauthorized", message="Invalid token."), 401
```

## Client Credentials Flow

For service-to-service authentication:

```python
from rampart import RampartClient

client = RampartClient(
    issuer_url="https://auth.example.com",
    client_id="my-service",
    client_secret=os.environ["RAMPART_CLIENT_SECRET"],
)

# Get a token
token = await client.client_credentials_token(scopes=["users:read"])

# Use the token
import httpx
async with httpx.AsyncClient() as http:
    resp = await http.get(
        "https://api.internal/users",
        headers={"Authorization": f"Bearer {token.access_token}"},
    )
```

The client caches tokens and refreshes them before expiry.
