# Rampart.AspNetCore

ASP.NET Core middleware for verifying [Rampart](https://github.com/manimovassagh/rampart) JWTs. Handles JWKS fetching, RS256 verification, claim extraction, and role-based authorization with zero configuration beyond the issuer URL.

## Install

```bash
dotnet add package Rampart.AspNetCore
```

Requires .NET 8.0 or later.

## Quick Start

Configure authentication in `Program.cs`:

```csharp
using Rampart.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddRampartAuth("http://localhost:8080");
builder.Services.AddControllers();

var app = builder.Build();

app.UseAuthentication();
app.UseAuthorization();
app.MapControllers();

app.Run();
```

Protect a controller:

```csharp
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Rampart.AspNetCore;

[ApiController]
[Route("api")]
[Authorize]
public class ProfileController : ControllerBase
{
    [HttpGet("me")]
    public IActionResult Me()
    {
        var claims = RampartClaims.FromPrincipal(User);
        return Ok(new { userId = claims.Sub, org = claims.OrgId });
    }
}
```

## Configuration

### `AddRampartAuth(issuer)`

Call on `IServiceCollection` to configure JWT Bearer authentication:

```csharp
builder.Services.AddRampartAuth("https://auth.example.com");
```

This:

1. Fetches the JWKS from `{issuer}/.well-known/jwks.json` (cached automatically)
2. Validates RS256 signatures, issuer, and token expiry
3. Maps the `roles` JWT claim to `ClaimsPrincipal` role claims for `[Authorize(Roles = "...")]`

## Claims

### `RampartClaims`

Typed representation of Rampart JWT claims. Extract from the authenticated user:

```csharp
var claims = RampartClaims.FromPrincipal(User);
```

| Property            | Type                   | Description                |
|---------------------|------------------------|----------------------------|
| `Sub`               | `string`               | User ID (UUID)             |
| `OrgId`             | `string?`              | Organization ID (UUID)     |
| `PreferredUsername`  | `string?`              | Username                   |
| `Email`             | `string?`              | Email address              |
| `EmailVerified`     | `bool`                 | Whether email is verified  |
| `GivenName`         | `string?`              | First name (if set)        |
| `FamilyName`        | `string?`              | Last name (if set)         |
| `Roles`             | `IReadOnlyList<string>`| Assigned roles             |

### Claims middleware

Optionally, register the claims middleware to make `RampartClaims` available via `HttpContext.Items`:

```csharp
app.UseAuthentication();
app.UseRampartClaims(); // extracts claims into HttpContext.Items
app.UseAuthorization();
```

Then access claims from anywhere with access to `HttpContext`:

```csharp
var claims = HttpContext.GetRampartClaims();
```

## Role-Based Authorization

### Using `[Authorize]` attribute (recommended)

Rampart roles are mapped to ASP.NET Core role claims automatically:

```csharp
[Authorize(Roles = "admin")]
[HttpGet("admin/dashboard")]
public IActionResult Dashboard() => Ok(new { area = "admin" });

[Authorize(Roles = "editor,admin")]
[HttpPost("articles")]
public IActionResult CreateArticle() => Ok();
```

### Using `[RequireRoles]` attribute

For Rampart-specific error format (matches other adapter error responses):

```csharp
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
    // ... admin routes
});
```

## Error Responses

On authentication failure, returns a `401` JSON response matching Rampart's error format:

```json
{
  "error": "unauthorized",
  "error_description": "Missing or invalid authorization header.",
  "status": 401
}
```

On authorization failure (missing roles), returns `403`:

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
| `UseRampartClaims()`                   | Middleware to set `HttpContext.Items["RampartClaims"]` |
| `HttpContext.GetRampartClaims()`       | Gets typed claims from `HttpContext.Items`            |
| `[RequireRoles("role")]`               | Authorization filter with Rampart error format        |
| `RequireRampartRoles("role")`          | Inline middleware for role enforcement                |

## License

MIT
