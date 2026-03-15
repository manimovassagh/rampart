# Rampart.AspNetCore

[![NuGet](https://img.shields.io/nuget/v/Rampart.AspNetCore?logo=nuget&color=004880)](https://www.nuget.org/packages/Rampart.AspNetCore)
[![.NET](https://img.shields.io/badge/.NET-8.0+-512BD4?logo=dotnet)](https://dotnet.microsoft.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

ASP.NET Core middleware for verifying [Rampart](https://github.com/manimovassagh/rampart) JWTs. Drop-in authentication and authorization with zero configuration beyond the issuer URL.

## Features

- **Automatic JWKS discovery** -- fetches and caches signing keys from your Rampart server
- **RS256 JWT verification** -- validates signatures, issuer, and token expiry out of the box
- **Typed claim extraction** -- `RampartClaims` gives you `Sub`, `Email`, `OrgId`, `Roles`, and more with full IntelliSense
- **Role-based authorization** -- works with ASP.NET Core `[Authorize(Roles = "...")]` and a custom `[RequireRoles]` attribute
- **Consistent error responses** -- returns structured JSON error bodies (401/403) matching the Rampart error format across all adapters

## Quick Start

```bash
dotnet add package Rampart.AspNetCore
```

Requires **.NET 8.0** or later.

Add three lines to `Program.cs` to enable Rampart authentication:

```csharp
builder.Services.AddRampartAuth("https://auth.example.com"); // 1. Configure
app.UseAuthentication();                                       // 2. Authenticate
app.UseAuthorization();                                        // 3. Authorize
```

Full minimal example:

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

1. Fetches the JWKS from `{issuer}/.well-known/jwks.json` (cached automatically by the underlying OIDC metadata handler)
2. Validates RS256 signatures, issuer claim, and token lifetime
3. Maps the `roles` JWT claim to ASP.NET Core role claims so `[Authorize(Roles = "...")]` works
4. Configures JSON error responses for 401 challenges (instead of the default `WWW-Authenticate` header)
5. Disables HTTPS metadata requirement for `http://` issuers (local development)

**Options applied under the hood:**

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

### Claims middleware

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

### Using `[Authorize]` attribute (recommended)

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

### Using `[RequireRoles]` attribute

For Rampart-specific JSON error responses (consistent across all Rampart adapters):

```csharp
/// <summary>
/// Requires both "editor" AND the authenticated state.
/// Returns 403 JSON: {"error":"forbidden","error_description":"Missing required role(s): editor","status":403}
/// </summary>
[Authorize]
[RequireRoles("editor")]
[HttpPut("articles/{id}")]
public IActionResult UpdateArticle(string id) => Ok();
```

### Using middleware

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

## Error Responses

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

## API Reference

| Method / Class                         | Description                                           |
|----------------------------------------|-------------------------------------------------------|
| `AddRampartAuth(issuer)`               | Configures JWT Bearer auth with Rampart JWKS          |
| `RampartClaims.FromPrincipal(user)`    | Extracts typed claims from `ClaimsPrincipal`          |
| `UseRampartClaims()`                   | Middleware: stores claims in `HttpContext.Items`       |
| `HttpContext.GetRampartClaims()`       | Retrieves typed claims from `HttpContext.Items`       |
| `[RequireRoles("role")]`               | Filter attribute with Rampart JSON error format       |
| `RequireRampartRoles("role")`          | Inline middleware for role enforcement                |

## Related

- [Rampart IAM Server](https://github.com/manimovassagh/rampart) -- the OAuth 2.0 / OIDC server this adapter connects to
- [Rampart Node.js Adapter](https://github.com/manimovassagh/rampart/tree/main/adapters/backend/node) -- Express.js middleware
- [Rampart Python Adapter](https://github.com/manimovassagh/rampart/tree/main/adapters/backend/python) -- FastAPI/Flask middleware
- [Rampart Go Adapter](https://github.com/manimovassagh/rampart/tree/main/adapters/backend/go) -- Go middleware

## License

[MIT](https://opensource.org/licenses/MIT)
