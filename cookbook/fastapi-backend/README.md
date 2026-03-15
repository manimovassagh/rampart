# Rampart FastAPI Sample Backend

A Python/FastAPI backend that demonstrates the [Rampart Python adapter](../../adapters/backend/python/) for JWT verification. This is a drop-in replacement for the Express sample backend, serving the same endpoints on the same port (3001) with identical JSON responses.

## Prerequisites

- Python 3.9+
- A running Rampart server (default: `http://localhost:8080`)

## Setup

```bash
# Create a virtual environment (recommended)
python3 -m venv .venv
source .venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Install the Rampart Python adapter from the local source tree
pip install -e ../../adapters/backend/python
```

## Run

```bash
# Using the built-in entrypoint
python main.py

# Or using uvicorn directly
uvicorn main:app --host 0.0.0.0 --port 3001 --reload
```

The server starts on `http://localhost:3001`.

## Configuration

| Environment Variable | Default                  | Description                  |
|----------------------|--------------------------|------------------------------|
| `RAMPART_ISSUER`     | `http://localhost:8080`  | Base URL of the Rampart server |
| `PORT`               | `3001`                   | Port to listen on            |

## Endpoints

| Method | Path                    | Auth     | Description                       |
|--------|-------------------------|----------|-----------------------------------|
| GET    | `/api/health`           | Public   | Health check                      |
| GET    | `/api/profile`          | JWT      | Authenticated user profile        |
| GET    | `/api/claims`           | JWT      | All JWT claims                    |
| GET    | `/api/editor/dashboard` | JWT+RBAC | Requires "editor" role            |
| GET    | `/api/manager/reports`  | JWT+RBAC | Requires "manager" role           |

## Switching from Express

Since this backend exposes the same routes and response shapes, you can swap it in by stopping the Express backend and starting this one. The React sample app will work without any changes.
