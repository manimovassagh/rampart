---
sidebar_position: 7
title: .NET / ASP.NET Core
description: Integrate Rampart authentication into ASP.NET Core applications with the Rampart.AspNetCore NuGet package.
---

# .NET / ASP.NET Core Adapter

The `Rampart.AspNetCore` adapter provides ASP.NET Core middleware for verifying Rampart-issued JWT tokens. It handles JWKS discovery, RS256 token verification, typed claim extraction, and role-based authorization with zero configuration beyond the issuer URL.

## Installation

```bash
dotnet add package Rampart.AspNetCore
```

Requires **.NET 8.0** or later.

## Quick Start

Add three lines to `Program.cs` to enable Rampart authentication:

```csharp
using Rampart.AspNetCore;

var builder = WebApplication.CreateBuilder(args);
builder.Services.AddRampartAuth("https://auth.example.com");

var app = builder.Build();
app.UseAuthentication();
app.UseAuthorization();

app.MapGet("/me", (HttpContext ctx) =>
{
    var claims = RampartClaims.FromPrincipal(ctx.User);
    return Results.Ok(new { userId = claims.Sub, org = claims.OrgId });
}).RequireAuthorization();

app.Run();
```

## Configuration

### `AddRampartAuth(issuer)`

Call on `IServiceCollection` to configure JWT Bearer authentication:

```csharp
builder.Services.AddRampartAuth("https://auth.example.com");
```

This single call:

1. Fetches the JWKS from `{issuer}/.well-known/jwks.json` (cached automatically)
2. Validates RS256 signatures, issuer claim, and token lifetime
3. Maps the `roles` JWT claim to ASP.NET Core role claims so `[Authorize(Roles = "...")]` works
4. Configures JSON error responses for 401 challenges
5. Disables HTTPS metadata requirement for `http://` issuers (local development)

### Options Applied Under the Hood

| Setting                       | Value                                  |
|-------------------------------|----------------------------------------|
| `ValidateIssuer`              | `true`                                 |
| `ValidIssuer`                 | The issuer URL you provide             |
| `ValidateAudience`            | `false` (Rampart tokens are not audience-scoped) |
| `ValidateLifetime`            | `true`                                 |
| `ValidateIssuerSigningKey`    | `true`                                 |
| `ValidAlgorithms`             | `RS256`                                |
| `RoleClaimType`               | `roles`                                |
| `NameClaimType`               | `preferred_username`                   |
| `RequireHttpsMetadata`        | `true` for `https://`, `false` for `http://` |

### Environment Variables

You can also read the issuer from configuration:

```csharp
var issuer = builder.Configuration["Rampart:Issuer"] ?? "https://auth.example.com";
builder.Services.AddRampartAuth(issuer);
```

```json
// appsettings.json
{
  "Rampart": {
    "Issuer": "https://auth.example.com"
  }
}
```

## Claims

### `RampartClaims`

Typed representation of Rampart JWT claims. Extract from the authenticated user:

```csharp
var claims = RampartClaims.FromPrincipal(User);

// IntelliSense-friendly properties:
// claims.Sub               -> "550e8400-e29b-41d4-a716-446655440000"
// claims.Email             -> "alice@example.com"
// claims.EmailVerified     -> true
// claims.OrgId             -> "org_abc123"
// claims.PreferredUsername  -> "alice"
// claims.Roles             -> ["admin", "editor"]
// claims.GivenName         -> "Alice"
// claims.FamilyName        -> "Smith"
```

| Property            | Type                   | JWT Claim              | Description                |
|---------------------|------------------------|------------------------|----------------------------|
| `Sub`               | `string`               | `sub`                  | User ID (UUID)             |
| `OrgId`             | `string?`              | `org_id`               | Organization ID            |
| `PreferredUsername`  | `string?`              | `preferred_username`   | Username                   |
| `Email`             | `string?`              | `email`                | Email address              |
| `EmailVerified`     | `bool`                 | `email_verified`       | Whether email is verified  |
| `GivenName`         | `string?`              | `given_name`           | First name                 |
| `FamilyName`        | `string?`              | `family_name`          | Last name                  |
| `Roles`             | `IReadOnlyList<string>`| `roles`                | Assigned roles             |

### Claims Middleware

Optionally, register the claims middleware to make `RampartClaims` available via `HttpContext.Items`:

```csharp
app.UseAuthentication();
app.UseRampartClaims();   // must be after UseAuthentication()
app.UseAuthorization();
```

Then access claims from anywhere with access to `HttpContext`:

```csharp
var claims = HttpContext.GetRampartClaims();
// Returns null if the user is not authenticated
```

## Role-Based Authorization

### Using `[Authorize]` Attribute (Recommended)

Rampart roles are mapped to ASP.NET Core role claims automatically:

```csharp
[Authorize(Roles = "admin")]
[HttpGet("admin/dashboard")]
public IActionResult Dashboard() => Ok(new { area = "admin" });

// Comma-separated = any of these roles
[Authorize(Roles = "editor,admin")]
[HttpPost("articles")]
public IActionResult CreateArticle() => Ok();
```

### Using `[RequireRoles]` Attribute

For Rampart-specific JSON error responses (consistent across all Rampart adapters):

```csharp
[Authorize]
[RequireRoles("editor")]
[HttpPut("articles/{id}")]
public IActionResult UpdateArticle(string id) => Ok();
```

Returns a structured 403 response:

```json
{
  "error": "forbidden",
  "error_description": "Missing required role(s): editor",
  "status": 403
}
```

### Using Middleware

For route-group-level role enforcement:

```csharp
app.UseAuthentication();
app.UseAuthorization();

app.Map("/api/admin", adminApp =>
{
    adminApp.RequireRampartRoles("admin");
    // all routes in this group require the "admin" role
});
```

## Error Handling

All error responses use a consistent JSON format that matches across Rampart's Go, Node.js, Python, and .NET adapters.

**401 Unauthorized** (missing or invalid token):

```json
{
  "error": "unauthorized",
  "error_description": "Missing or invalid authorization header.",
  "status": 401
}
```

**403 Forbidden** (authenticated but missing required roles):

```json
{
  "error": "forbidden",
  "error_description": "Missing required role(s): admin, editor",
  "status": 403
}
```

## Full Working Example

```csharp
using Rampart.AspNetCore;

var builder = WebApplication.CreateBuilder(args);
builder.Services.AddRampartAuth(
    builder.Configuration["Rampart:Issuer"] ?? "https://auth.example.com"
);

var app = builder.Build();
app.UseAuthentication();
app.UseRampartClaims();
app.UseAuthorization();

// Public route
app.MapGet("/health", () => Results.Ok(new { status = "ok" }));

// Protected route — any authenticated user
app.MapGet("/api/profile", (HttpContext ctx) =>
{
    var claims = ctx.GetRampartClaims()!;
    return Results.Ok(new
    {
        userId = claims.Sub,
        email = claims.Email,
        org = claims.OrgId,
        roles = claims.Roles,
    });
}).RequireAuthorization();

// Admin route — requires "admin" role
app.MapGet("/api/admin/stats", [Authorize(Roles = "admin")] () =>
{
    return Results.Ok(new
    {
        totalUsers = 1234,
        activeToday = 567,
    });
}).RequireAuthorization();

app.Run();
```

## API Reference

| Method / Class                         | Description                                           |
|----------------------------------------|-------------------------------------------------------|
| `AddRampartAuth(issuer)`               | Configures JWT Bearer auth with Rampart JWKS          |
| `RampartClaims.FromPrincipal(user)`    | Extracts typed claims from `ClaimsPrincipal`          |
| `UseRampartClaims()`                   | Middleware: stores claims in `HttpContext.Items`       |
| `HttpContext.GetRampartClaims()`       | Retrieves typed claims from `HttpContext.Items`       |
| `[RequireRoles("role")]`               | Filter attribute with Rampart JSON error format       |
| `RequireRampartRoles("role")`          | Inline middleware for role enforcement                |

## Security Considerations

- **Always use HTTPS in production.** The adapter automatically enforces HTTPS metadata for `https://` issuers.
- **Keep access token lifetimes short** (5-15 minutes) to limit the window of compromise.
- **Use `[Authorize]` at the route level** rather than relying solely on global middleware, so authorization intent is explicit in your code.
- **Validate `OrgId`** in multi-tenant applications to prevent cross-tenant data access.
