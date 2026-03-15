//! JWT claims extracted from Rampart access tokens.

use serde::{Deserialize, Serialize};

/// Verified JWT claims from a Rampart access token.
///
/// Fields map to the JSON claim names used by the Rampart IAM server.
/// Optional fields (`given_name`, `family_name`, `roles`) default to
/// empty values when absent from the token.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    /// Subject identifier (user UUID) — JWT "sub" claim.
    pub sub: String,

    /// Issuer URL — JWT "iss" claim.
    pub iss: String,

    /// Token issued-at time as a Unix timestamp — JWT "iat" claim.
    pub iat: f64,

    /// Token expiration time as a Unix timestamp — JWT "exp" claim.
    pub exp: f64,

    /// Organization UUID the user belongs to — custom "org_id" claim.
    #[serde(default)]
    pub org_id: String,

    /// User's display name — OIDC "preferred_username" claim.
    #[serde(default)]
    pub preferred_username: String,

    /// User's email address — OIDC "email" claim.
    #[serde(default)]
    pub email: String,

    /// Whether the email address has been verified — OIDC "email_verified" claim.
    #[serde(default)]
    pub email_verified: bool,

    /// User's first name — OIDC "given_name" claim.
    #[serde(default)]
    pub given_name: String,

    /// User's last name — OIDC "family_name" claim.
    #[serde(default)]
    pub family_name: String,

    /// Roles assigned to the user — custom "roles" claim.
    #[serde(default)]
    pub roles: Vec<String>,
}

impl Claims {
    /// Returns `true` if the user has the specified role.
    pub fn has_role(&self, role: &str) -> bool {
        self.roles.iter().any(|r| r == role)
    }

    /// Returns `true` if the user has all of the specified roles.
    pub fn has_all_roles(&self, roles: &[&str]) -> bool {
        roles.iter().all(|r| self.has_role(r))
    }

    /// Returns the list of roles from `required` that the user is missing.
    pub fn missing_roles<'a>(&self, required: &[&'a str]) -> Vec<&'a str> {
        required
            .iter()
            .filter(|r| !self.has_role(r))
            .copied()
            .collect()
    }
}
