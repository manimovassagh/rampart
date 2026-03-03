# System Context (C4 Level 1)

High-level view of Rampart and the actors that interact with it.

```mermaid
C4Context
    title System Context — Rampart IAM

    Person(endUser, "End User", "Authenticates via login page, manages own account")
    Person(admin, "Admin User", "Manages organizations, users, clients, roles via Admin UI or API")
    Person(developer, "Developer", "Integrates apps using SDKs, CLI, or REST API")

    System(rampart, "Rampart", "Lightweight IAM server — OAuth 2.0, OIDC, user management, RBAC, MFA")

    System_Ext(clientSPA, "SPA / Web App", "React, Vue, Angular, Svelte")
    System_Ext(clientMobile, "Mobile App", "iOS, Android")
    System_Ext(clientServer, "Backend Service", "Node.js, Go, Python, Java, .NET")
    System_Ext(clientCLI, "CLI Tool", "Developer or automation tooling")

    System_Ext(google, "Google", "Social login via OIDC")
    System_Ext(github, "GitHub", "Social login via OAuth")
    System_Ext(microsoft, "Microsoft Entra ID", "Enterprise SSO via OIDC/SAML")
    System_Ext(samlIdP, "SAML Identity Provider", "Enterprise SSO")

    System_Ext(smtp, "Email Service", "SendGrid, SES, SMTP — verification, password reset")
    System_Ext(sms, "SMS Service", "Twilio, SNS — MFA codes")

    System_Ext(postgres, "PostgreSQL", "Primary data store")
    System_Ext(redis, "Redis / Valkey", "Sessions, token blacklist, rate limiting")

    Rel(endUser, rampart, "Authenticates, manages account")
    Rel(admin, rampart, "Manages resources via Admin API / UI")
    Rel(developer, rampart, "Integrates via SDK / CLI / REST")

    Rel(clientSPA, rampart, "OAuth 2.0 Auth Code + PKCE")
    Rel(clientMobile, rampart, "OAuth 2.0 Auth Code + PKCE")
    Rel(clientServer, rampart, "Client Credentials, Token Introspection")
    Rel(clientCLI, rampart, "Device Authorization Flow")

    Rel(rampart, google, "Social login (OIDC)")
    Rel(rampart, github, "Social login (OAuth)")
    Rel(rampart, microsoft, "Enterprise SSO (OIDC/SAML)")
    Rel(rampart, samlIdP, "Enterprise SSO (SAML 2.0)")

    Rel(rampart, smtp, "Sends verification & reset emails")
    Rel(rampart, sms, "Sends MFA codes")

    Rel(rampart, postgres, "Reads/writes users, clients, tokens, events")
    Rel(rampart, redis, "Sessions, blacklist, rate limits")
```

## Actors

| Actor | Description |
|-------|-------------|
| **End User** | Person who authenticates to access a protected application. Interacts with login/consent pages and self-service account management. |
| **Admin User** | Person who manages Rampart configuration — creates organizations, registers OAuth clients, manages users and roles. Uses Admin Dashboard UI or Admin REST API. |
| **Developer** | Person who integrates their application with Rampart using SDKs, the CLI tool, or direct REST API calls. |

## Client Applications

Rampart supports any application that speaks OAuth 2.0 / OIDC:

| Client Type | Auth Flow | Example |
|-------------|-----------|---------|
| SPA (browser) | Authorization Code + PKCE | React, Vue, Angular dashboard |
| Mobile app | Authorization Code + PKCE | iOS / Android native app |
| Backend service | Client Credentials | Microservice-to-microservice |
| CLI / IoT | Device Authorization Flow | `rampart-cli`, smart TV app |

## External Systems

| System | Purpose |
|--------|---------|
| **External IdPs** | Social login (Google, GitHub, Apple) and enterprise SSO (SAML, OIDC) |
| **Email service** | Email verification, password reset, MFA codes |
| **SMS service** | MFA OTP delivery |
| **PostgreSQL** | Primary persistent storage for all domain data |
| **Redis / Valkey** | Session storage, token blacklisting, rate limit counters |
