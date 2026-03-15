//! Error types matching the Rampart server error format.

use serde::{Deserialize, Serialize};

/// JSON error response matching the Rampart server format.
///
/// Returned by middleware on authentication or authorization failures.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorResponse {
    /// Machine-readable error code (e.g., "unauthorized", "forbidden").
    pub error: String,

    /// Human-readable explanation of the failure.
    pub error_description: String,

    /// HTTP status code (e.g., 401, 403).
    pub status: u16,
}

impl ErrorResponse {
    /// Creates a 401 Unauthorized error response.
    pub fn unauthorized(description: &str) -> Self {
        Self {
            error: "unauthorized".to_string(),
            error_description: description.to_string(),
            status: 401,
        }
    }

    /// Creates a 403 Forbidden error response.
    pub fn forbidden(description: &str) -> Self {
        Self {
            error: "forbidden".to_string(),
            error_description: description.to_string(),
            status: 403,
        }
    }
}

/// Errors that can occur during token verification.
#[derive(Debug, thiserror::Error)]
pub enum RampartError {
    #[error("Missing authorization header.")]
    MissingAuthHeader,

    #[error("Invalid authorization header format.")]
    InvalidAuthHeader,

    #[error("Failed to fetch JWKS.")]
    JwksFetchError(#[source] reqwest::Error),

    #[error("No matching key found in JWKS.")]
    NoMatchingKey,

    #[error("Invalid or expired access token.")]
    InvalidToken(#[source] jsonwebtoken::errors::Error),

    #[error("Authentication required.")]
    NotAuthenticated,

    #[error("Missing required role(s): {0}")]
    MissingRoles(String),
}

impl RampartError {
    /// Converts the error into a Rampart-format [`ErrorResponse`].
    pub fn to_error_response(&self) -> ErrorResponse {
        match self {
            Self::MissingRoles(roles) => ErrorResponse::forbidden(
                &format!("Missing required role(s): {roles}"),
            ),
            Self::MissingAuthHeader => {
                ErrorResponse::unauthorized("Missing authorization header.")
            }
            Self::InvalidAuthHeader => {
                ErrorResponse::unauthorized("Invalid authorization header format.")
            }
            Self::JwksFetchError(_) => {
                ErrorResponse::unauthorized("Failed to fetch JWKS.")
            }
            Self::NoMatchingKey => {
                ErrorResponse::unauthorized("Invalid or expired access token.")
            }
            Self::InvalidToken(_) => {
                ErrorResponse::unauthorized("Invalid or expired access token.")
            }
            Self::NotAuthenticated => {
                ErrorResponse::unauthorized("Authentication required.")
            }
        }
    }

    /// Returns the appropriate HTTP status code for this error.
    pub fn status_code(&self) -> u16 {
        match self {
            Self::MissingRoles(_) => 403,
            _ => 401,
        }
    }
}
