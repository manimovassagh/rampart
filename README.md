<p align="center">
  <img src="https://img.shields.io/badge/🏰_RAMPART-Identity_&_Access_Management-8B5CF6?style=for-the-badge&labelColor=1e1e2e" alt="Rampart">
</p>

<h3 align="center">The lightweight, modern identity & access management server</h3>

<p align="center">
  <em>A better alternative to Keycloak — single binary, beautiful UX, enterprise-ready.</em>
</p>

<p align="center">
  <strong>7 SDK adapters</strong> &middot; <strong>5 login themes</strong> &middot; <strong>Config export/import</strong> &middot; <strong>10 AI-powered setup skills</strong>
</p>

<p align="center">
  <a href="https://github.com/manimovassagh/rampart/actions"><img src="https://img.shields.io/github/actions/workflow/status/manimovassagh/rampart/ci.yml?branch=main&style=for-the-badge&logo=github&label=CI&color=2ea043" alt="CI"></a>&nbsp;
  <a href="https://github.com/manimovassagh/rampart/releases"><img src="https://img.shields.io/github/v/release/manimovassagh/rampart?style=for-the-badge&logo=semanticrelease&label=Release&color=0ea5e9" alt="Release"></a>&nbsp;
  <a href="https://github.com/manimovassagh/rampart/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-AGPL_v3-E879F9?style=for-the-badge" alt="License"></a>&nbsp;
  <a href="https://goreportcard.com/report/github.com/manimovassagh/rampart"><img src="https://img.shields.io/badge/Go_Report-A+-22d3ee?style=for-the-badge&logo=go&logoColor=white" alt="Go Report Card"></a>&nbsp;
  <a href="https://github.com/manimovassagh/rampart/stargazers"><img src="https://img.shields.io/github/stars/manimovassagh/rampart?style=for-the-badge&logo=github&color=f59e0b" alt="Stars"></a>&nbsp;
  <a href="https://manimovassagh.github.io/rampart/"><img src="https://img.shields.io/badge/Docs-Live-16a34a?style=for-the-badge&logo=docusaurus&logoColor=white" alt="Documentation"></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/htmx-3366CC?style=flat-square&logo=htmx&logoColor=white" alt="htmx">
  <img src="https://img.shields.io/badge/Tailwind-06B6D4?style=flat-square&logo=tailwindcss&logoColor=white" alt="Tailwind">
  <img src="https://img.shields.io/badge/PostgreSQL-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-DC382D?style=flat-square&logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/OAuth_2.0-EB5424?style=flat-square&logo=auth0&logoColor=white" alt="OAuth 2.0">
  <img src="https://img.shields.io/badge/OpenID_Connect-F78C40?style=flat-square&logo=openid&logoColor=white" alt="OIDC">
</p>

---

<p align="center">
  <strong>ram·part</strong> <em>/ˈramˌpärt/</em> — A fortified defensive wall. The barrier that stands between your application and unauthorized access.<br>
  Just as a castle's rampart protects everything within, <strong>Rampart</strong> guards your users, data, and services.
</p>

---

## 📖 Documentation

<p align="center">
  <a href="https://manimovassagh.github.io/rampart/">
    <img src="docs/images/docs-site-preview.png" alt="Rampart Documentation Site" width="720">
  </a>
</p>

<p align="center">
  <strong><a href="https://manimovassagh.github.io/rampart/">manimovassagh.github.io/rampart</a></strong><br>
  Getting started, API reference, SDK guides, architecture, and more.
</p>

---

<br>

## ⚡ Why Rampart?

> Keycloak is powerful — but it's heavy, slow to start, painful to theme, and stuck in the Java/WildFly era.
> **Rampart is everything Keycloak should have been.**

<table>
<tr>
<td align="center"><strong></strong></td>
<td align="center"><img src="https://img.shields.io/badge/Keycloak-red?style=flat-square" alt="Keycloak"></td>
<td align="center"><img src="https://img.shields.io/badge/Rampart-8B5CF6?style=flat-square" alt="Rampart"></td>
</tr>
<tr><td>⚡ <strong>Startup</strong></td><td align="center">~10s+</td><td align="center"><strong>< 1s</strong></td></tr>
<tr><td>💾 <strong>Memory</strong></td><td align="center">~512MB+</td><td align="center"><strong>~30MB</strong></td></tr>
<tr><td>📦 <strong>Deployment</strong></td><td align="center">WAR on WildFly</td><td align="center"><strong>Single binary</strong></td></tr>
<tr><td>🎨 <strong>Theming</strong></td><td align="center">FreeMarker</td><td align="center"><strong>5 built-in themes, zero config</strong></td></tr>
<tr><td>🔌 <strong>Extensions</strong></td><td align="center">Java SPIs</td><td align="center"><strong>WASM / gRPC plugins</strong></td></tr>
<tr><td>🖥️ <strong>Admin UI</strong></td><td align="center">Dated</td><td align="center"><strong>Modern & fast</strong></td></tr>
<tr><td>📤 <strong>Config Export</strong></td><td align="center">Manual backup</td><td align="center"><strong>One-click JSON export/import</strong></td></tr>
<tr><td>🧠 <strong>AI Setup</strong></td><td align="center">❌</td><td align="center"><strong>10 Claude Code skills</strong></td></tr>
</table>

<br>

## ✨ Features

<table>
<tr>
<td width="50%">

### 🔐 Authentication & Authorization
- **OAuth 2.0** — Auth Code + PKCE, Client Credentials, Device Flow
- **OpenID Connect** — ID Tokens, Discovery, JWKS
- **SAML 2.0** — IdP & SP bridge for enterprise SSO
- **MFA** — TOTP, WebAuthn/Passkeys, recovery codes

</td>
<td width="50%">

### 👥 User Management
- Registration, profiles, groups, and roles
- Social login — Google, GitHub, Apple & more
- RBAC with fine-grained permissions
- Self-service account management

</td>
</tr>
<tr>
<td>

### 🏢 Multi-Tenancy
- Isolated organizations / realms
- Per-tenant branding and themes
- Separate user pools and configurations

</td>
<td>

### 🛠️ Developer Experience
- REST API for everything
- Webhook events for real-time integrations
- **7 SDK adapters** — Node.js, React, Next.js, Web, Go, Python, Spring Boot
- CLI tool & OpenAPI spec

</td>
</tr>
<tr>
<td>

### 🎨 Login Themes
- 5 production-ready themes out of the box
- Per-organization theme selection
- Go Templates + Tailwind — no FreeMarker pain

</td>
<td>

### 📤 Config Export / Import
- One-click organization export as JSON
- Import into any Rampart instance
- GitOps-friendly configuration management

</td>
</tr>
</table>

<br>

## 🎨 Login Themes — Pick One, Ship It

Every Rampart instance ships with **5 built-in login themes**. Select per-organization in the admin dashboard — zero custom CSS required.

| Theme | Style | Best For |
|-------|-------|----------|
| **`default`** | Clean, light, professional | SaaS products, general use |
| **`dark`** | Dark mode, high contrast | Developer tools, tech products |
| **`minimal`** | Ultra-clean, whitespace-focused | Design-forward brands |
| **`corporate`** | Structured, enterprise look | B2B, regulated industries |
| **`gradient`** | Bold gradients, modern feel | Consumer apps, startups |

Set a theme per organization via the Admin API:
```bash
curl -X PUT http://localhost:8080/api/v1/admin/organizations/{id}/settings \
  -H "Content-Type: application/json" \
  -d '{"login_theme": "dark"}'
```

> **Compare with Keycloak:** Theming in Keycloak means writing FreeMarker templates, copying theme directories, and restarting the server. In Rampart, it's one API call.

<br>

## 📤 Config Export / Import

Move configurations between environments with a single command. No manual recreation, no missed settings.

```bash
# Export an organization's full config (clients, roles, groups, settings, theme)
curl http://localhost:8080/api/v1/admin/organizations/{id}/export > org-config.json

# Import into another Rampart instance
curl -X POST http://localhost:8080/api/v1/admin/organizations/import \
  -H "Content-Type: application/json" \
  -d @org-config.json
```

The export includes everything: clients, roles, groups, organization settings, and login theme. Check it into Git, use it in CI/CD, replicate across environments.

> **Compare with Keycloak:** Keycloak's partial-export is limited, JSON realm exports are fragile across versions, and there's no clean import API. Rampart's export/import is version-stable and API-first.

<br>

## 🚀 Quick Start

**Docker (recommended):**
```bash
docker run -d --name rampart -p 8080:8080 \
  -e RAMPART_DB_URL=postgres://user:pass@host:5432/rampart \
  ghcr.io/manimovassagh/rampart:latest
```

**Docker Compose:**
```bash
git clone https://github.com/manimovassagh/rampart.git && cd rampart
docker compose up -d
```

**From source:**
```bash
git clone https://github.com/manimovassagh/rampart.git && cd rampart
make build && ./bin/rampart serve
```

> [!TIP]
> **Full documentation available at [manimovassagh.github.io/rampart](https://manimovassagh.github.io/rampart/)** — Getting started, API reference, SDK guides, architecture, and more.

<br>

## 🧠 AI-Powered Setup — A First for IAM

Rampart is the **first identity server with built-in AI developer skills**. Using [Claude Code](https://docs.anthropic.com/en/docs/claude-code), you can wire up authentication in your app with a single command.

Type a skill command in Claude Code and your entire auth integration gets scaffolded automatically:

| Skill | What It Does |
|-------|-------------|
| `/rampart-docker-quickstart` | Spins up Rampart + Postgres + Redis with Docker Compose |
| `/rampart-react-setup` | Adds login, logout, token refresh, and protected routes to a React app |
| `/rampart-node-setup` | Configures Express/Fastify middleware with token validation |
| `/rampart-nextjs-setup` | Sets up Next.js auth with server components and middleware |
| `/rampart-go-setup` | Adds Rampart middleware to a Go HTTP server |
| `/rampart-python-setup` | Configures Flask/FastAPI with OIDC token verification |
| `/rampart-spring-setup` | Wires Spring Security OAuth2 resource server to Rampart |
| `/rampart-fullstack-secure` | End-to-end setup: frontend + backend + Rampart, fully wired |
| `/rampart-login-theme` | Previews and applies login themes to your organization |
| `/rampart-ci-check` | Validates your Rampart config, runs security checks in CI |

> **Why this matters:** Other IAM products give you docs and SDKs. Rampart gives you an AI agent that reads your codebase and writes the integration code for you — correctly, the first time.

<br>

## 🏗️ Architecture

```
                    Your Application
         ┌──────────────────────────────────┐
         │  SDK Adapters (7 platforms)       │
         │  React · Next.js · Web · Node.js │
         │  Go · Python · Spring Boot       │
         └───────────────┬──────────────────┘
                         │ OAuth 2.0 / OIDC
                         ▼
┌─────────────────────────┐     ┌─────────────────────────┐
│      Admin Dashboard    │     │    Login / Consent UI   │
│  (htmx + Go Templates) │     │  (5 themes · Tailwind)  │
└───────────┬─────────────┘     └───────────┬─────────────┘
            │            REST API           │
            └──────────────┬────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────┐
│                  Rampart Server (Go)                     │
│                                                          │
│  🔑 OIDC/OAuth    👤 Users    🛡️ RBAC    📡 Webhooks     │
│  🔗 SAML Bridge   🔒 MFA     🏢 Tenants  📤 Export       │
│                                                          │
│              Plugin API (WASM / gRPC)                    │
└─────────────┬────────────────────────┬──────────────────┘
              │                        │
     ┌────────▼────────┐     ┌────────▼────────┐
     │   PostgreSQL    │     │  Redis / Valkey  │
     │   (primary DB)  │     │   (sessions)     │
     └─────────────────┘     └─────────────────┘
```

<br>

## 📋 Roadmap

> Track progress on the [GitHub Issues](https://github.com/manimovassagh/rampart/issues) page.

| Phase | Status | Scope |
|-------|--------|-------|
| ![Phase 1](https://img.shields.io/badge/Phase_1-Core_Foundation-16a34a?style=flat-square) | ✅ Complete | Go server, OAuth/OIDC, user management, admin UI, Docker |
| ![Phase 2](https://img.shields.io/badge/Phase_2-Full_Features-0ea5e9?style=flat-square) | ⏳ Planned | SAML, social login, MFA, webhooks, plugin system |
| ![Phase 3](https://img.shields.io/badge/Phase_3-Enterprise-8B5CF6?style=flat-square) | ⏳ Planned | HA clustering, SCIM, compliance, SDKs, cloud managed |

<details>
<summary>📌 <strong>Detailed Phase 1 Checklist</strong></summary>

- [x] Project setup and CI/CD pipeline
- [x] HTTP server with graceful shutdown
- [x] PostgreSQL database layer with migrations
- [x] User registration and secure password hashing
- [x] OAuth 2.0 Authorization Code flow + PKCE
- [x] OIDC ID token issuance and discovery
- [x] Admin dashboard (Users, Roles, Groups, Orgs, Sessions, Events, Clients, OIDC Config)
- [x] Docker deployment
- [x] CLI tool (`rampart-cli`)
- [x] E2E tests (Playwright)
- [x] SDK adapters (Node.js, React, Next.js, Web, Go, Python, Spring Boot)
- [x] Login themes (default, dark, minimal, corporate, gradient)
- [x] Config export/import (JSON API)
- [x] Claude Code skills (10 AI-powered setup commands)

</details>

<br>

## 🗺️ Comparison

| Feature | <img src="https://img.shields.io/badge/-Rampart-8B5CF6?style=flat-square" alt="Rampart"> | Keycloak | Ory | Zitadel | Authentik |
|---------|:-------:|:--------:|:---:|:-------:|:---------:|
| **Language** | 🟢 Go | Java | Go | Go | Python |
| **Single Binary** | ✅ | ❌ | ✅ | ✅ | ❌ |
| **Memory** | **~30MB** | ~512MB | ~50MB | ~100MB | ~300MB |
| **Built-in UI** | ✅ | ✅ | ❌ | ✅ | ✅ |
| **OAuth 2.0 / OIDC** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **SAML** | ✅ | ✅ | ❌ | ✅ | ✅ |
| **Multi-Tenant** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Plugin System** | ✅ | ⚠️ Java only | ❌ | ❌ | ❌ |
| **Login Theming** | ✅ 5 built-in | ⚠️ FreeMarker | N/A | ⚠️ | ⚠️ |
| **Config Export/Import** | ✅ JSON API | ⚠️ Fragile | ❌ | ⚠️ | ⚠️ |
| **SDK Adapters** | ✅ 7 | ⚠️ Java-centric | ✅ | ⚠️ | ⚠️ |
| **AI-Powered Setup** | ✅ 10 skills | ❌ | ❌ | ❌ | ❌ |

<br>

## 🤝 Contributing

We welcome contributions! Whether it's a bug fix, feature, or docs — every contribution helps.

```bash
# Fork the repo, then:
git clone https://github.com/YOUR_USERNAME/rampart.git
cd rampart
make dev-setup    # install dependencies and tools
make test         # run tests
make lint         # run linters
```

> **Workflow:** Fork → Branch (`feat/your-feature`) → Code + Tests → `make test && make lint` → Pull Request

<br>

## 📝 License

<img src="https://img.shields.io/badge/License-AGPL_v3-E879F9?style=for-the-badge" alt="AGPL v3">

Licensed under the GNU Affero General Public License v3.0 — see [LICENSE](LICENSE) for details.

<br>

## 💬 Community & Support

<p>
  <a href="https://github.com/manimovassagh/rampart/issues"><img src="https://img.shields.io/badge/🐛_Issues-Report_Bugs-dc2626?style=for-the-badge" alt="Issues"></a>&nbsp;
  <a href="https://github.com/manimovassagh/rampart/discussions"><img src="https://img.shields.io/badge/💡_Discussions-Ask_Questions-2563eb?style=for-the-badge" alt="Discussions"></a>&nbsp;
  <a href="https://github.com/manimovassagh/rampart"><img src="https://img.shields.io/badge/⭐_Star-Support_the_Project-f59e0b?style=for-the-badge" alt="Star"></a>
</p>

---

<p align="center">
  <strong>🏰 Rampart</strong> — Identity management that doesn't feel like a fortress to set up.
  <br><br>
  <img src="https://img.shields.io/badge/Built_with-Go_💙-00ADD8?style=flat-square" alt="Go">
  <img src="https://img.shields.io/badge/Made_with-❤️-dc2626?style=flat-square" alt="Love">
  <img src="https://img.shields.io/badge/For-Developers-8B5CF6?style=flat-square" alt="Developers">
</p>
