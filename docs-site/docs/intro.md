---
sidebar_position: 1
title: Introduction
description: Rampart is a lightweight, modern identity and access management server built with Go and PostgreSQL.
---

# Introduction

Rampart is a lightweight, production-grade identity and access management (IAM) server. It provides OAuth 2.0 and OpenID Connect out of the box, with an admin console, CLI tool, and SDK adapters for every major stack — all shipped as a single binary.

Built in Go with PostgreSQL, Rampart starts in under 1 second and runs at approximately 30 MB of memory. No Java runtime, no complex dependency chains. Just download, configure, and run.

## Key Features

- **OAuth 2.0 and OpenID Connect** — Authorization Code with PKCE, Client Credentials, Refresh Tokens, OIDC Discovery, JWKS
- **Social Login** — Sign in with Google, GitHub, and Apple out of the box
- **SAML 2.0** — Service Provider and Identity Provider support for enterprise SSO
- **MFA** — Multi-factor authentication with TOTP and WebAuthn (passkeys, security keys)
- **Single Binary Deployment** — One binary, no runtime dependencies beyond PostgreSQL
- **Admin Console** — Full-featured web UI for managing users, organizations, roles, sessions, and audit events
- **Multi-Tenancy** — Built-in organization support with user isolation and per-org configuration
- **RBAC** — Role-based access control with built-in and custom roles
- **Session Management** — PostgreSQL-backed sessions with listing, single and bulk revocation
- **Audit Logging** — Every security-relevant event tracked with IP addresses, timestamps, and actor information
- **Webhooks** — Real-time event notifications for user lifecycle, login, and admin actions
- **SCIM 2.0 Provisioning** — Automated user and group provisioning from external identity providers
- **Compliance Reporting** — Built-in reports for access reviews, login activity, and policy enforcement
- **CLI Tool** — Manage users, inspect tokens, and authenticate via the command line
- **SDK Adapters** — Official integrations for Node.js, React, Next.js, Go, Python, Spring Boot, .NET, and vanilla JavaScript
- **Themeable Login Pages** — CSS variable-based themes, selectable per organization
- **Low Resource Footprint** — ~30 MB memory, sub-second startup, minimal CPU usage

## Who Is Rampart For?

- **Application developers** who need authentication and authorization without building it from scratch
- **Platform teams** who want a self-hosted identity provider that is easy to deploy and operate
- **Organizations** looking for a lightweight, self-hosted IAM solution
- **Startups** that need production-grade auth from day one without the operational overhead

## How It Compares

| | Rampart | Keycloak | Ory | Zitadel | Authentik |
|---|---|---|---|---|---|
| Language | Go | Java | Go | Go | Python |
| Startup Time | < 1s | 10-30s | < 1s | ~3s | ~10s |
| Memory | ~30 MB | 512 MB+ | ~50 MB | ~100 MB | ~300 MB |
| Deployment | Single binary | Docker or Kubernetes (Quarkus-based) | Multiple services | Single binary | Docker required |
| Admin UI | Yes | Yes | No (headless) | Yes | Yes |

## Next Steps

Ready to get started? Head to the [Quickstart guide](./getting-started/quickstart.md) to have Rampart running in under five minutes.
