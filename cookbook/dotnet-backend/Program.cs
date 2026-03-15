using System.Security.Claims;
using System.Text.Json;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.IdentityModel.Tokens;

var builder = WebApplication.CreateBuilder(args);

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------
var issuer = builder.Configuration["Rampart:Issuer"]
    ?? Environment.GetEnvironmentVariable("RAMPART_ISSUER")
    ?? "http://localhost:8080";

issuer = issuer.TrimEnd('/');

// ---------------------------------------------------------------------------
// Authentication — JWT Bearer validated against the Rampart OIDC discovery
// ---------------------------------------------------------------------------
builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        options.Authority = issuer;
        options.MetadataAddress = $"{issuer}/.well-known/openid-configuration";
        options.RequireHttpsMetadata = !issuer.StartsWith("http://");
        options.MapInboundClaims = false;
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuer = true,
            ValidIssuer = issuer,
            ValidateAudience = false,
            ValidateLifetime = true,
            NameClaimType = "preferred_username",
            RoleClaimType = "roles",
        };

        // Custom error responses matching the Express backend format.
        options.Events = new JwtBearerEvents
        {
            OnChallenge = context =>
            {
                // Suppress the default WWW-Authenticate behaviour.
                context.HandleResponse();

                context.Response.StatusCode = 401;
                context.Response.ContentType = "application/json";

                var description = string.IsNullOrEmpty(context.ErrorDescription)
                    ? "Invalid or expired access token."
                    : context.ErrorDescription;

                if (string.IsNullOrEmpty(context.Error) && string.IsNullOrEmpty(context.ErrorDescription))
                {
                    description = "Missing authorization header.";
                }

                var body = JsonSerializer.Serialize(new
                {
                    error = "unauthorized",
                    error_description = description,
                    status = 401,
                });

                return context.Response.WriteAsync(body);
            },
        };
    });

builder.Services.AddAuthorization();

// ---------------------------------------------------------------------------
// CORS — allow all origins (matches the Express sample)
// ---------------------------------------------------------------------------
builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(policy =>
    {
        policy.AllowAnyOrigin()
              .AllowAnyHeader()
              .AllowAnyMethod();
    });
});

// ---------------------------------------------------------------------------
// Kestrel — listen on port 3001
// ---------------------------------------------------------------------------
builder.WebHost.UseUrls("http://0.0.0.0:3001");

var app = builder.Build();

app.UseCors();
app.UseAuthentication();
app.UseAuthorization();

// ---------------------------------------------------------------------------
// Helper — extract Rampart claims from the authenticated user
// ---------------------------------------------------------------------------
static object ExtractUser(ClaimsPrincipal principal)
{
    var claims = principal.Claims.ToList();

    string? Claim(string type) => claims.FirstOrDefault(c => c.Type == type)?.Value;

    return new
    {
        id = Claim("sub"),
        email = Claim("email"),
        username = Claim("preferred_username"),
        org_id = Claim("org_id"),
        email_verified = bool.TryParse(Claim("email_verified"), out var ev) && ev,
        given_name = Claim("given_name"),
        family_name = Claim("family_name"),
        roles = claims.Where(c => c.Type == "roles").Select(c => c.Value).ToArray(),
    };
}

static Dictionary<string, object?> ExtractAllClaims(ClaimsPrincipal principal)
{
    var dict = new Dictionary<string, object?>();

    foreach (var group in principal.Claims.GroupBy(c => c.Type))
    {
        var values = group.Select(c => c.Value).ToList();
        if (values.Count == 1)
            dict[group.Key] = values[0];
        else
            dict[group.Key] = values;
    }

    // Convert numeric strings back to numbers for iat/exp
    foreach (var key in new[] { "iat", "exp", "nbf" })
    {
        if (dict.TryGetValue(key, out var val) && val is string s && long.TryParse(s, out var n))
            dict[key] = n;
    }

    // Convert email_verified to boolean
    if (dict.TryGetValue("email_verified", out var evVal) && evVal is string evStr)
        dict["email_verified"] = bool.TryParse(evStr, out var b) && b;

    return dict;
}

static string[] GetRoles(ClaimsPrincipal principal) =>
    principal.Claims.Where(c => c.Type == "roles").Select(c => c.Value).ToArray();

// ---------------------------------------------------------------------------
// Middleware — role requirement helper (returns 403 matching Express format)
// ---------------------------------------------------------------------------
static IResult RequireRoles(ClaimsPrincipal user, params string[] requiredRoles)
{
    var userRoles = GetRoles(user);
    var missing = requiredRoles.Where(r => !userRoles.Contains(r)).ToArray();

    if (missing.Length > 0)
    {
        return Results.Json(new
        {
            error = "forbidden",
            error_description = $"Missing required role(s): {string.Join(", ", missing)}",
            status = 403,
        }, statusCode: 403);
    }

    return null!; // no missing roles
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

// Public — no auth required
app.MapGet("/api/health", () => Results.Json(new { status = "ok", issuer }));

// Protected — requires valid Rampart JWT
app.MapGet("/api/profile", (ClaimsPrincipal user) =>
{
    return Results.Json(new
    {
        message = "Authenticated!",
        user = ExtractUser(user),
    });
}).RequireAuthorization();

// Protected — returns all raw claims
app.MapGet("/api/claims", (ClaimsPrincipal user) =>
{
    return Results.Json(ExtractAllClaims(user));
}).RequireAuthorization();

// Role-protected — requires "editor" role
app.MapGet("/api/editor/dashboard", (ClaimsPrincipal user) =>
{
    var check = RequireRoles(user, "editor");
    if (check is not null) return check;

    return Results.Json(new
    {
        message = "Welcome, Editor!",
        user = user.Claims.FirstOrDefault(c => c.Type == "preferred_username")?.Value,
        roles = GetRoles(user),
        data = new
        {
            drafts = 3,
            published = 12,
            pending_review = 2,
        },
    });
}).RequireAuthorization();

// Role-protected — requires "manager" role
app.MapGet("/api/manager/reports", (ClaimsPrincipal user) =>
{
    var check = RequireRoles(user, "manager");
    if (check is not null) return check;

    return Results.Json(new
    {
        message = "Manager Reports",
        user = user.Claims.FirstOrDefault(c => c.Type == "preferred_username")?.Value,
        roles = GetRoles(user),
        reports = new[]
        {
            new { name = "Q1 Revenue", status = "complete" },
            new { name = "User Growth", status = "in_progress" },
        },
    });
}).RequireAuthorization();

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------
Console.WriteLine($"Sample backend running on http://localhost:3001");
Console.WriteLine($"Rampart issuer: {issuer}");
Console.WriteLine();
Console.WriteLine("Routes:");
Console.WriteLine("  GET /api/health            - public");
Console.WriteLine("  GET /api/profile           - protected (any authenticated user)");
Console.WriteLine("  GET /api/claims            - protected (any authenticated user)");
Console.WriteLine("  GET /api/editor/dashboard  - protected (requires \"editor\" role)");
Console.WriteLine("  GET /api/manager/reports   - protected (requires \"manager\" role)");

app.Run();
