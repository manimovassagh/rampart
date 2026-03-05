---
name: rampart-login-theme
description: Customize the Rampart login page theme and branding. Choose a built-in theme, set custom colors, logo, title, and message. Use when you want to brand the Rampart login page for your organization.
argument-hint: [org-id]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Customize the Rampart Login Page Theme

Configure the look and feel of the Rampart login page for your organization using built-in themes or custom branding.

## What This Skill Does

1. Helps you choose a built-in login page theme
2. Customizes colors (primary, background, accent) via the admin API
3. Sets a custom logo URL for the login page
4. Customizes the login page title and welcome message
5. Shows how to preview the result

## Built-In Themes

Rampart ships with these built-in themes:

| Theme | Description |
|-------|-------------|
| `default` | Clean light theme with Rampart blue accents |
| `dark` | Dark background with light text, modern feel |
| `minimal` | White background, minimal borders, stripped-down look |
| `corporate` | Professional navy and gray tones, formal layout |
| `gradient` | Colorful gradient background, vibrant accents |

## Step-by-Step

### 1. Choose a built-in theme

Set the theme for your organization via the Admin API:

```bash
# Replace ORG_ID with your organization ID (or use "default")
ORG_ID="${ARGUMENTS:-default}"

curl -s -X PUT "http://localhost:8080/api/v1/admin/orgs/${ORG_ID}/theme" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "theme": "dark"
  }'
```

Available values for `theme`: `default`, `dark`, `minimal`, `corporate`, `gradient`.

### 2. Customize colors

Override individual colors on top of the chosen theme:

```bash
curl -s -X PUT "http://localhost:8080/api/v1/admin/orgs/${ORG_ID}/theme" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "theme": "dark",
    "colors": {
      "primary": "#4F46E5",
      "background": "#0F172A",
      "accent": "#22D3EE",
      "text": "#F8FAFC",
      "error": "#EF4444",
      "border": "#334155"
    }
  }'
```

Color fields (all optional -- omitted fields inherit from the base theme):

| Field | Description | Example |
|-------|-------------|---------|
| `primary` | Buttons, links, active states | `#4F46E5` |
| `background` | Page background color | `#0F172A` |
| `accent` | Secondary highlights, focus rings | `#22D3EE` |
| `text` | Primary text color | `#F8FAFC` |
| `error` | Error messages, validation | `#EF4444` |
| `border` | Input borders, dividers | `#334155` |

### 3. Set a custom logo

```bash
curl -s -X PUT "http://localhost:8080/api/v1/admin/orgs/${ORG_ID}/theme" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "logo_url": "https://example.com/logo.svg"
  }'
```

The logo appears centered above the login form. Recommended: SVG or PNG, max height 48px, transparent background.

### 4. Customize title and message

```bash
curl -s -X PUT "http://localhost:8080/api/v1/admin/orgs/${ORG_ID}/theme" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "login_title": "Welcome to Acme Corp",
    "login_message": "Sign in with your corporate account to continue."
  }'
```

### 5. Full example -- all settings at once

```bash
curl -s -X PUT "http://localhost:8080/api/v1/admin/orgs/${ORG_ID}/theme" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "theme": "corporate",
    "colors": {
      "primary": "#1E3A5F",
      "background": "#F0F4F8",
      "accent": "#2563EB"
    },
    "logo_url": "https://example.com/logo.svg",
    "login_title": "Acme Corp Portal",
    "login_message": "Authorized personnel only. Sign in to continue."
  }'
```

### 6. Preview the result

Open the login page in your browser to see the changes:

```
http://localhost:8080/login?org=${ORG_ID}
```

Changes take effect immediately -- no restart required.

### 7. Read the current theme

```bash
curl -s "http://localhost:8080/api/v1/admin/orgs/${ORG_ID}/theme" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq .
```

## Setting the Theme via Docker Environment Variable

You can set the default theme without the API by passing environment variables in `docker-compose.yml`:

```yaml
services:
  rampart:
    image: ghcr.io/manimovassagh/rampart:latest
    environment:
      RAMPART_LOGIN_THEME: dark
      RAMPART_LOGIN_LOGO_URL: https://example.com/logo.svg
      RAMPART_LOGIN_TITLE: "Welcome to Acme Corp"
      RAMPART_LOGIN_MESSAGE: "Sign in to continue."
      RAMPART_LOGIN_COLOR_PRIMARY: "#4F46E5"
      RAMPART_LOGIN_COLOR_BACKGROUND: "#0F172A"
      RAMPART_LOGIN_COLOR_ACCENT: "#22D3EE"
```

Environment variables set the default for all organizations. Per-org API settings override the env defaults.

## How Themes Work

Rampart login themes use CSS custom properties (CSS variables). Each theme defines a set of variables that the login page consumes:

```css
:root {
  --rampart-color-primary: #4F46E5;
  --rampart-color-background: #FFFFFF;
  --rampart-color-accent: #22D3EE;
  --rampart-color-text: #1E293B;
  --rampart-color-error: #EF4444;
  --rampart-color-border: #E2E8F0;
}
```

When you select a theme or override colors via the API, Rampart injects the corresponding CSS variables into the login page. This means no component re-rendering -- just CSS updates, instant and flicker-free.

## Checklist

- [ ] Built-in theme selected (or staying with `default`)
- [ ] Custom colors configured (if needed)
- [ ] Logo URL set (if branding required)
- [ ] Login title and message customized
- [ ] Login page previewed at `http://localhost:8080/login`
- [ ] Theme persisted via API or Docker env vars
