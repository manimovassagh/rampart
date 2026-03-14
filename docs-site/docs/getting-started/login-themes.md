---
sidebar_label: Login Themes
title: Login Page Themes
---

# Login Page Themes

Rampart ships with **5 built-in themes** for the login and consent pages. Changing the theme for an organization is a single API call — no server restart, no template files, no theme archive deployments.

## Built-in Themes

| Theme | Description |
|-------|-------------|
| **default** | Clean white background with the Rampart purple accent. Balanced and professional. |
| **dark** | Dark background with high-contrast text. Easy on the eyes, great for developer-facing products. |
| **minimal** | Stripped-down layout with no background decoration. Just the form, centered. |
| **corporate** | Structured layout with a sidebar brand area and a neutral color palette. Suitable for enterprise portals. |
| **gradient** | Full-screen gradient background (purple to pink) with a frosted-glass card. Eye-catching and modern. |

## Changing the Theme

Set the login theme for an organization via the Admin API:

```bash
curl -X PUT https://rampart.example.com/api/v1/admin/organizations/{org_id}/settings \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "login_theme": "dark"
  }'
```

The change takes effect immediately for all new login page loads in that organization.

### Docker Environment Variable

You can set a default theme for the entire instance via environment variable:

```bash
docker run -d \
  -e RAMPART_LOGIN_THEME=dark \
  -e RAMPART_DB_URL=postgres://... \
  -p 8080:8080 \
  rampart/rampart:latest
```

The environment variable sets the default theme. Per-organization settings from the API take precedence.

## Customizing Branding

Beyond the theme, you can customize the login page branding per organization:

```bash
curl -X PUT https://rampart.example.com/api/v1/admin/organizations/{org_id}/settings \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "login_theme": "corporate",
    "login_page_title": "Acme Corp Sign In",
    "login_page_message": "Welcome back. Please sign in to continue.",
    "login_logo_url": "https://cdn.acme.com/logo.svg",
    "login_primary_color": "#1e40af"
  }'
```

| Field | Description |
|-------|-------------|
| `login_theme` | One of: `default`, `dark`, `minimal`, `corporate`, `gradient` |
| `login_page_title` | The heading displayed on the login form |
| `login_page_message` | A subtitle or welcome message below the heading |
| `login_logo_url` | URL to your organization logo (displayed above the form) |
| `login_primary_color` | Hex color for buttons and links (overrides the theme default) |

## CSS Variable Reference

All themes are implemented with CSS custom properties. You can override any of these for deeper customization:

```css
:root {
  /* Background */
  --rampart-bg: #ffffff;
  --rampart-bg-secondary: #f9fafb;

  /* Card / form container */
  --rampart-card-bg: #ffffff;
  --rampart-card-border: #e5e7eb;
  --rampart-card-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  --rampart-card-radius: 12px;

  /* Typography */
  --rampart-text-primary: #111827;
  --rampart-text-secondary: #6b7280;
  --rampart-text-muted: #9ca3af;

  /* Brand / accent */
  --rampart-primary: #8b5cf6;
  --rampart-primary-hover: #7c3aed;
  --rampart-primary-text: #ffffff;

  /* Inputs */
  --rampart-input-bg: #ffffff;
  --rampart-input-border: #d1d5db;
  --rampart-input-focus-border: #8b5cf6;
  --rampart-input-text: #111827;
  --rampart-input-placeholder: #9ca3af;
  --rampart-input-radius: 8px;

  /* Links */
  --rampart-link-color: #8b5cf6;
  --rampart-link-hover: #7c3aed;

  /* Errors */
  --rampart-error: #ef4444;
  --rampart-error-bg: #fef2f2;

  /* Success */
  --rampart-success: #22c55e;
  --rampart-success-bg: #f0fdf4;
}
```

To apply custom overrides, serve a CSS file from your CDN and configure it in the organization settings:

```bash
curl -X PUT https://rampart.example.com/api/v1/admin/organizations/{org_id}/settings \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "login_custom_css_url": "https://cdn.acme.com/rampart-theme.css"
  }'
```

## Theming Approach

Rampart makes theming simple — pick a built-in theme, set your logo and colors, and you are done. For deeper customization, override CSS variables. No server restart, no template language, no build step required.
