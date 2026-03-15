using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;

namespace Rampart.AspNetCore;

/// <summary>
/// Optional middleware that extracts <see cref="RampartClaims"/> from the authenticated
/// user and stores them in <c>HttpContext.Items["RampartClaims"]</c> for easy access
/// in controllers and other middleware.
/// </summary>
public sealed class RampartClaimsMiddleware
{
    /// <summary>
    /// The key used to store <see cref="RampartClaims"/> in <see cref="HttpContext.Items"/>.
    /// </summary>
    public const string HttpContextKey = "RampartClaims";

    private readonly RequestDelegate _next;

    /// <summary>
    /// Creates a new instance of <see cref="RampartClaimsMiddleware"/>.
    /// </summary>
    /// <param name="next">The next middleware in the pipeline.</param>
    public RampartClaimsMiddleware(RequestDelegate next)
    {
        _next = next;
    }

    /// <summary>
    /// Processes the request. If the user is authenticated, extracts typed
    /// <see cref="RampartClaims"/> and stores them in <see cref="HttpContext.Items"/>.
    /// </summary>
    public async Task InvokeAsync(HttpContext context)
    {
        if (context.User.Identity?.IsAuthenticated == true)
        {
            var claims = RampartClaims.FromPrincipal(context.User);
            context.Items[HttpContextKey] = claims;
        }

        await _next(context);
    }
}

/// <summary>
/// Extension methods for registering <see cref="RampartClaimsMiddleware"/>.
/// </summary>
public static class RampartClaimsMiddlewareExtensions
{
    /// <summary>
    /// Adds the Rampart claims middleware to the pipeline. This extracts typed
    /// <see cref="RampartClaims"/> from the authenticated user and stores them
    /// in <c>HttpContext.Items["RampartClaims"]</c>.
    /// Must be placed after <c>UseAuthentication()</c>.
    /// </summary>
    /// <param name="app">The application builder.</param>
    /// <returns>The application builder for chaining.</returns>
    public static IApplicationBuilder UseRampartClaims(this IApplicationBuilder app)
    {
        return app.UseMiddleware<RampartClaimsMiddleware>();
    }

    /// <summary>
    /// Gets the <see cref="RampartClaims"/> from <see cref="HttpContext.Items"/>.
    /// Returns <c>null</c> if the claims middleware has not run or the user is not authenticated.
    /// </summary>
    /// <param name="context">The HTTP context.</param>
    /// <returns>The Rampart claims, or <c>null</c>.</returns>
    public static RampartClaims? GetRampartClaims(this HttpContext context)
    {
        return context.Items.TryGetValue(RampartClaimsMiddleware.HttpContextKey, out var value)
            ? value as RampartClaims
            : null;
    }
}
