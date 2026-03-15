using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.Filters;

namespace Rampart.AspNetCore;

/// <summary>
/// Authorization filter attribute that requires the authenticated user to have
/// all of the specified Rampart roles. Returns a 403 JSON response matching
/// the Rampart error format if any required role is missing.
/// </summary>
/// <example>
/// <code>
/// [Authorize]
/// [RequireRoles("editor", "admin")]
/// [HttpGet("/api/admin")]
/// public IActionResult Admin() => Ok();
/// </code>
/// </example>
[AttributeUsage(AttributeTargets.Class | AttributeTargets.Method, AllowMultiple = true)]
public sealed class RequireRolesAttribute : Attribute, IAuthorizationFilter
{
    private readonly string[] _roles;

    /// <summary>
    /// Creates a new <see cref="RequireRolesAttribute"/> requiring all specified roles.
    /// </summary>
    /// <param name="roles">One or more role names that the user must have.</param>
    public RequireRolesAttribute(params string[] roles)
    {
        _roles = roles ?? throw new ArgumentNullException(nameof(roles));
    }

    /// <summary>
    /// Checks that the authenticated user has all required roles.
    /// </summary>
    public void OnAuthorization(AuthorizationFilterContext context)
    {
        var user = context.HttpContext.User;

        if (user.Identity == null || !user.Identity.IsAuthenticated)
        {
            context.Result = new JsonResult(new
            {
                error = "unauthorized",
                error_description = "Authentication required.",
                status = 401
            })
            {
                StatusCode = 401,
                ContentType = "application/json"
            };
            return;
        }

        var claims = RampartClaims.FromPrincipal(user);
        var userRoles = new HashSet<string>(claims.Roles);
        var missing = _roles.Where(r => !userRoles.Contains(r)).ToList();

        if (missing.Count > 0)
        {
            context.Result = new JsonResult(new
            {
                error = "forbidden",
                error_description = "Insufficient permissions.",
                status = 403
            })
            {
                StatusCode = 403,
                ContentType = "application/json"
            };
        }
    }
}

/// <summary>
/// Extension methods for adding role-based authorization middleware.
/// </summary>
public static class RampartRolesMiddlewareExtensions
{
    /// <summary>
    /// Adds middleware that requires the authenticated user to have all of the specified roles.
    /// Returns a 403 JSON response matching the Rampart error format if any role is missing.
    /// Must be placed after <c>UseAuthentication()</c> and <c>UseAuthorization()</c>.
    /// </summary>
    /// <param name="app">The application builder.</param>
    /// <param name="roles">One or more required role names.</param>
    /// <returns>The application builder for chaining.</returns>
    public static IApplicationBuilder RequireRampartRoles(this IApplicationBuilder app, params string[] roles)
    {
        return app.Use(async (context, next) =>
        {
            var user = context.User;

            if (user.Identity == null || !user.Identity.IsAuthenticated)
            {
                context.Response.StatusCode = 401;
                context.Response.ContentType = "application/json";
                await context.Response.WriteAsync(System.Text.Json.JsonSerializer.Serialize(new
                {
                    error = "unauthorized",
                    error_description = "Authentication required.",
                    status = 401
                }));
                return;
            }

            var claims = RampartClaims.FromPrincipal(user);
            var userRoles = new HashSet<string>(claims.Roles);
            var missing = roles.Where(r => !userRoles.Contains(r)).ToList();

            if (missing.Count > 0)
            {
                context.Response.StatusCode = 403;
                context.Response.ContentType = "application/json";
                await context.Response.WriteAsync(System.Text.Json.JsonSerializer.Serialize(new
                {
                    error = "forbidden",
                    error_description = "Insufficient permissions.",
                    status = 403
                }));
                return;
            }

            await next();
        });
    }
}
