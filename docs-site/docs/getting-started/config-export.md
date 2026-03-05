---
sidebar_label: Config Export & Import
title: Config Export & Import
---

# Config Export & Import

Rampart lets you export your entire organization configuration as a single JSON file and import it into another instance. This is the recommended workflow for promoting config from staging to production, seeding new environments, or backing up your setup.

## Exporting Config

Export all settings, roles, groups, and OAuth clients for an organization:

```bash
curl -s https://rampart.example.com/api/v1/admin/organizations/{org_id}/export \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o org-export.json
```

The response is a complete JSON snapshot:

```json
{
  "organization": {
    "name": "acme",
    "slug": "acme",
    "display_name": "Acme Corporation"
  },
  "settings": {
    "password_min_length": 12,
    "password_require_uppercase": true,
    "password_require_lowercase": true,
    "password_require_numbers": true,
    "password_require_symbols": false,
    "mfa_enforcement": "optional",
    "access_token_ttl_seconds": 900,
    "refresh_token_ttl_seconds": 86400,
    "self_registration_enabled": true,
    "email_verification_required": true,
    "forgot_password_enabled": true,
    "remember_me_enabled": true,
    "login_page_title": "Acme Corp Sign In",
    "login_page_message": "Welcome back."
  },
  "roles": [
    {"name": "admin", "description": "Full access to all resources"},
    {"name": "editor", "description": "Can edit content"},
    {"name": "viewer", "description": "Read-only access"}
  ],
  "groups": [
    {
      "name": "engineering",
      "description": "Engineering team",
      "roles": ["admin", "editor"]
    },
    {
      "name": "support",
      "description": "Customer support",
      "roles": ["viewer"]
    }
  ],
  "clients": [
    {
      "client_id": "acme-web-app",
      "name": "Acme Web Application",
      "description": "Main customer-facing SPA",
      "client_type": "public",
      "redirect_uris": [
        "https://app.acme.com/callback",
        "http://localhost:3000/callback"
      ],
      "enabled": true
    },
    {
      "client_id": "acme-api",
      "name": "Acme Backend API",
      "description": "Server-to-server API client",
      "client_type": "confidential",
      "redirect_uris": [],
      "enabled": true
    }
  ]
}
```

**Security note:** Client secrets are never included in the export. After importing, confidential clients will need their secrets regenerated.

## Importing Config

Import a previously exported JSON file into a target organization:

```bash
curl -X POST https://rampart.example.com/api/v1/admin/organizations/{org_id}/import \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @org-export.json
```

The import is **additive by default**: existing resources that match by name or client ID are updated, and new ones are created. Nothing is deleted.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/admin/organizations/{org_id}/export` | Export full org config as JSON |
| `POST` | `/api/v1/admin/organizations/{org_id}/import` | Import org config from JSON |

Both endpoints require an admin token with the `admin` role in the target organization.

## CLI Usage

The Rampart CLI wraps the export/import API:

```bash
# Export
rampart config export --org acme --output acme-config.json

# Import into a different instance
rampart config import --org acme --input acme-config.json --server https://prod.example.com
```

## Docker Usage

You can seed an organization on first startup by mounting an export file:

```bash
docker run -d \
  -e RAMPART_DB_URL=postgres://... \
  -v ./acme-config.json:/etc/rampart/seed.json \
  -e RAMPART_SEED_FILE=/etc/rampart/seed.json \
  -p 8080:8080 \
  rampart/rampart:latest
```

The seed file is applied once when the organization does not yet exist. Subsequent restarts skip it.

## Typical Workflow: Staging to Production

1. Configure your organization in the staging environment (settings, roles, groups, clients) using the admin dashboard or API.
2. Export the config:
   ```bash
   rampart config export --org acme --server https://staging.example.com --output acme-config.json
   ```
3. Review the JSON file and commit it to your Git repo for auditability.
4. Import into production:
   ```bash
   rampart config import --org acme --server https://prod.example.com --input acme-config.json
   ```
5. Regenerate client secrets for confidential clients in production.

## What Is Included

| Resource | Included | Notes |
|----------|----------|-------|
| Organization name/slug | Yes | |
| Organization settings | Yes | Password policy, MFA, token TTL, login branding |
| Roles | Yes | Name and description |
| Groups | Yes | Name, description, and role assignments |
| OAuth clients | Yes | All fields except client secret |
| Users | No | Users are never exported (security) |
| Sessions | No | Sessions are instance-specific |
| Audit logs | No | Logs are append-only and instance-specific |

## Comparison with Keycloak

Keycloak's realm export is notoriously fragile and does not include all configuration. It requires a running server, often misses client secrets and federated identity config, and the import can fail silently on version mismatches.

**Rampart exports everything in one clean JSON file.** The format is stable, human-readable, and version-controlled. Import is deterministic — it either succeeds fully or returns clear errors.
