using System.Security.Claims;

namespace Rampart.AspNetCore;

/// <summary>
/// Typed representation of Rampart JWT claims.
/// Use <see cref="FromPrincipal"/> to extract claims from a validated <see cref="ClaimsPrincipal"/>.
/// </summary>
public sealed class RampartClaims
{
    /// <summary>Subject identifier (user ID).</summary>
    public string Sub { get; init; } = string.Empty;

    /// <summary>Organization ID.</summary>
    public string? OrgId { get; init; }

    /// <summary>Username.</summary>
    public string? PreferredUsername { get; init; }

    /// <summary>Email address.</summary>
    public string? Email { get; init; }

    /// <summary>Whether the email address has been verified.</summary>
    public bool EmailVerified { get; init; }

    /// <summary>First name (if set).</summary>
    public string? GivenName { get; init; }

    /// <summary>Last name (if set).</summary>
    public string? FamilyName { get; init; }

    /// <summary>Assigned roles.</summary>
    public IReadOnlyList<string> Roles { get; init; } = Array.Empty<string>();

    /// <summary>
    /// Extracts typed Rampart claims from a validated <see cref="ClaimsPrincipal"/>.
    /// </summary>
    /// <param name="principal">The claims principal from the authenticated user.</param>
    /// <returns>A <see cref="RampartClaims"/> instance with extracted claim values.</returns>
    /// <exception cref="ArgumentNullException">Thrown when <paramref name="principal"/> is null.</exception>
    public static RampartClaims FromPrincipal(ClaimsPrincipal principal)
    {
        ArgumentNullException.ThrowIfNull(principal, nameof(principal));

        var identity = principal.Identity as ClaimsIdentity;
        if (identity == null)
        {
            return new RampartClaims();
        }

        var roles = principal.FindAll(ClaimTypes.Role)
            .Select(c => c.Value)
            .Concat(principal.FindAll("roles").Select(c => c.Value))
            .Distinct()
            .ToList();

        var emailVerifiedClaim = identity.FindFirst("email_verified");
        var emailVerified = emailVerifiedClaim != null &&
            bool.TryParse(emailVerifiedClaim.Value, out var ev) && ev;

        return new RampartClaims
        {
            Sub = identity.FindFirst(ClaimTypes.NameIdentifier)?.Value
                  ?? identity.FindFirst("sub")?.Value
                  ?? string.Empty,
            OrgId = identity.FindFirst("org_id")?.Value,
            PreferredUsername = identity.FindFirst("preferred_username")?.Value,
            Email = identity.FindFirst(ClaimTypes.Email)?.Value
                    ?? identity.FindFirst("email")?.Value,
            EmailVerified = emailVerified,
            GivenName = identity.FindFirst(ClaimTypes.GivenName)?.Value
                        ?? identity.FindFirst("given_name")?.Value,
            FamilyName = identity.FindFirst(ClaimTypes.Surname)?.Value
                         ?? identity.FindFirst("family_name")?.Value,
            Roles = roles.AsReadOnly(),
        };
    }

    /// <inheritdoc />
    public override string ToString()
    {
        return $"RampartClaims{{Sub='{Sub}', Email='{Email}', Roles=[{string.Join(", ", Roles)}]}}";
    }
}
