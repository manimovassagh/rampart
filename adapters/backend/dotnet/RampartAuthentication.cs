using System.Security.Claims;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.IdentityModel.Tokens;

namespace Rampart.AspNetCore;

/// <summary>
/// Extension methods for configuring Rampart JWT authentication in ASP.NET Core.
/// </summary>
public static class RampartAuthentication
{
    /// <summary>
    /// Adds Rampart JWT Bearer authentication to the service collection.
    /// Configures JWKS auto-discovery from <c>{issuer}/.well-known/jwks.json</c>,
    /// RS256 signature validation, issuer validation, and lifetime validation.
    /// Maps the <c>roles</c> JWT claim to <see cref="ClaimsPrincipal"/> role claims.
    /// </summary>
    /// <param name="services">The service collection.</param>
    /// <param name="issuer">The Rampart server URL (e.g. <c>https://auth.example.com</c>).</param>
    /// <returns>The service collection for chaining.</returns>
    public static IServiceCollection AddRampartAuth(this IServiceCollection services, string issuer)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(issuer, nameof(issuer));

        issuer = issuer.TrimEnd('/');

        services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
            .AddJwtBearer(options =>
            {
                options.MapInboundClaims = false;
                options.Authority = issuer;
                options.MetadataAddress = $"{issuer}/.well-known/openid-configuration";
                options.RequireHttpsMetadata = !issuer.StartsWith("http://");
                options.TokenValidationParameters = new TokenValidationParameters
                {
                    ValidateIssuer = true,
                    ValidIssuer = issuer,
                    ValidateAudience = false,
                    ValidateLifetime = true,
                    ValidateIssuerSigningKey = true,
                    ValidAlgorithms = new[] { SecurityAlgorithms.RsaSha256 },
                    RoleClaimType = "roles",
                    NameClaimType = "preferred_username",
                };

                options.Events = new JwtBearerEvents
                {
                    OnChallenge = context =>
                    {
                        // Suppress the default WWW-Authenticate response and return our JSON format
                        context.HandleResponse();

                        context.Response.StatusCode = 401;
                        context.Response.ContentType = "application/json";

                        var description = string.IsNullOrEmpty(context.ErrorDescription)
                            ? "Missing or invalid authorization header."
                            : context.ErrorDescription;

                        if (!string.IsNullOrEmpty(context.Error) &&
                            context.Error.Equals("invalid_token", StringComparison.OrdinalIgnoreCase))
                        {
                            description = "Invalid or expired access token.";
                        }

                        var json = System.Text.Json.JsonSerializer.Serialize(new
                        {
                            error = "unauthorized",
                            error_description = description,
                            status = 401
                        });

                        return context.Response.WriteAsync(json);
                    },
                    OnTokenValidated = context =>
                    {
                        // Map the "roles" array claim to individual role claims for [Authorize(Roles = "...")]
                        if (context.Principal?.Identity is ClaimsIdentity identity)
                        {
                            var rolesClaim = identity.FindFirst("roles");
                            if (rolesClaim != null)
                            {
                                // The roles claim may be a JSON array; parse individual roles
                                try
                                {
                                    var roles = System.Text.Json.JsonSerializer.Deserialize<string[]>(rolesClaim.Value);
                                    if (roles != null)
                                    {
                                        foreach (var role in roles)
                                        {
                                            identity.AddClaim(new Claim(ClaimTypes.Role, role));
                                        }
                                    }
                                }
                                catch (System.Text.Json.JsonException)
                                {
                                    // Single role value, not an array
                                    identity.AddClaim(new Claim(ClaimTypes.Role, rolesClaim.Value));
                                }
                            }

                            // Also handle if roles come as multiple individual claims
                            var existingRoles = identity.FindAll("roles").ToList();
                            if (existingRoles.Count > 1)
                            {
                                foreach (var claim in existingRoles)
                                {
                                    if (!identity.HasClaim(ClaimTypes.Role, claim.Value))
                                    {
                                        identity.AddClaim(new Claim(ClaimTypes.Role, claim.Value));
                                    }
                                }
                            }
                        }

                        return Task.CompletedTask;
                    },
                    OnMessageReceived = context =>
                    {
                        // Ensure we return proper JSON for missing auth header
                        return Task.CompletedTask;
                    },
                    OnAuthenticationFailed = context =>
                    {
                        return Task.CompletedTask;
                    }
                };
            });

        services.AddAuthorization();

        return services;
    }
}
