---
sidebar_position: 4
title: CLI Tool
description: Rampart CLI reference — installation, authentication, user management, token inspection, and all available commands.
---

# CLI Tool

The Rampart CLI (`rampart-cli`) provides command-line access to authentication, user management, and token operations. It is useful for development workflows, scripting, and CI/CD pipelines.

## Installation

### Build from Source

```bash
git clone https://github.com/manimovassagh/rampart.git
cd rampart
make build-cli
```

The binary is output to `./bin/rampart-cli`. Move it to a directory on your PATH:

```bash
sudo mv ./bin/rampart-cli /usr/local/bin/
```

### Verify Installation

```bash
rampart-cli version
```

```
rampart-cli v1.0.0 (go1.22, linux/amd64)
```

## Configuration

Set the Rampart server URL before using the CLI:

```bash
export RAMPART_SERVER=http://localhost:8080
```

Alternatively, pass it with every command using the `--server` flag:

```bash
rampart-cli --server http://localhost:8080 <command>
```

## Commands

### `login`

Authenticate with the Rampart server. Stores the token locally for subsequent commands.

```bash
rampart-cli login --server http://localhost:8080
```

You will be prompted for your email and password:

```
Email: admin@example.com
Password: ********
Login successful. Token stored at ~/.rampart/token.json
```

For non-interactive use (CI/CD):

```bash
rampart-cli login --email admin@example.com --password "$RAMPART_PASSWORD"
```

### `logout`

Clear the stored authentication token:

```bash
rampart-cli logout
```

```
Token cleared. You are now logged out.
```

### `status`

Check connectivity to the Rampart server and authentication status:

```bash
rampart-cli status
```

```
Server:        http://localhost:8080
Status:        healthy
Authenticated: yes
User:          admin@example.com
Token Expires: 2026-03-05T11:00:00Z
```

### `whoami`

Display the currently authenticated user's profile:

```bash
rampart-cli whoami
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "admin@example.com",
  "first_name": "Admin",
  "last_name": "User",
  "roles": ["admin"],
  "organization": "default"
}
```

### `users list`

List all users (requires admin role):

```bash
rampart-cli users list
```

```
ID                                    EMAIL                  NAME          ROLES
550e8400-e29b-41d4-a716-446655440000  admin@example.com      Admin User    admin
660e8400-e29b-41d4-a716-446655440001  jane@example.com       Jane Smith    user
```

Use `--format json` for JSON output:

```bash
rampart-cli users list --format json
```

### `users create`

Create a new user (requires admin role):

```bash
rampart-cli users create \
  --email jane@example.com \
  --password "SecureP@ss123!" \
  --first-name Jane \
  --last-name Smith \
  --role user
```

```
User created: 660e8400-e29b-41d4-a716-446655440001
```

### `users get`

Retrieve details for a specific user by ID or email:

```bash
rampart-cli users get --email jane@example.com
```

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "email": "jane@example.com",
  "first_name": "Jane",
  "last_name": "Smith",
  "roles": ["user"],
  "organization": "default",
  "created_at": "2026-03-05T10:30:00Z",
  "last_login": "2026-03-05T10:45:00Z"
}
```

### `token`

Display and inspect the current access token:

```bash
rampart-cli token
```

```
Access Token (decoded):
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "iss": "http://localhost:8080",
  "aud": "rampart",
  "exp": 1741176000,
  "iat": 1741172400,
  "email": "admin@example.com",
  "roles": ["admin"],
  "org": "default"
}

Expires: 2026-03-05T11:00:00Z (valid for 42m remaining)
```

To output only the raw token string (useful for piping):

```bash
rampart-cli token --raw
```

```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

### `version`

Print the CLI version, Go version, and platform:

```bash
rampart-cli version
```

```
rampart-cli v1.0.0 (go1.22, linux/amd64)
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--server` | Rampart server URL (overrides `RAMPART_SERVER` env var) |
| `--format` | Output format: `text` (default), `json` |
| `--verbose` | Enable verbose output for debugging |
| `--help` | Show help for any command |

## Token Storage

The CLI stores authentication tokens at `~/.rampart/token.json`. This file is created with `0600` permissions (readable only by the current user). The token file contains:

```json
{
  "access_token": "eyJ...",
  "refresh_token": "dGh...",
  "expires_at": "2026-03-05T11:00:00Z",
  "server": "http://localhost:8080"
}
```

The CLI automatically refreshes expired access tokens using the stored refresh token. If the refresh token is also expired, you will be prompted to log in again.
