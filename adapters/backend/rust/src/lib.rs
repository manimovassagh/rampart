//! Rampart Rust adapter — JWT verification middleware for Actix-web and Axum.
//!
//! Validates RS256 tokens issued by a Rampart IAM server, extracts typed
//! claims, and supports role-based access control. The adapter fetches JWKS
//! from the issuer's `/.well-known/jwks.json` endpoint and caches keys
//! in memory.
//!
//! # Quick Start (Axum)
//!
//! ```rust,no_run
//! use rampart_rust::RampartAuth;
//!
//! let auth = RampartAuth::new("https://auth.example.com");
//! ```
//!
//! # Quick Start (Actix-web)
//!
//! ```rust,no_run
//! use rampart_rust::RampartAuth;
//!
//! let auth = RampartAuth::new("https://auth.example.com");
//! ```

pub mod claims;
pub mod error;
pub mod middleware;

pub use claims::Claims;
pub use error::{ErrorResponse, RampartError};
pub use middleware::RequireRoles;

use jsonwebtoken::{decode, Algorithm, DecodingKey, TokenData, Validation};
use serde::Deserialize;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

/// JWKS key entry from `/.well-known/jwks.json`.
#[derive(Debug, Clone, Deserialize)]
struct JwkKey {
    kty: String,
    #[serde(default)]
    kid: Option<String>,
    #[serde(default)]
    #[allow(dead_code)]
    alg: Option<String>,
    /// RSA modulus (base64url-encoded).
    n: String,
    /// RSA exponent (base64url-encoded).
    e: String,
}

/// JWKS response from `/.well-known/jwks.json`.
#[derive(Debug, Clone, Deserialize)]
struct JwksResponse {
    keys: Vec<JwkKey>,
}

/// Cached JWKS key set with expiration.
struct CachedJwks {
    keys: Vec<JwkKey>,
    fetched_at: Instant,
}

/// Configuration for the Rampart authentication layer.
#[derive(Debug, Clone)]
pub struct RampartConfig {
    /// Base URL of the Rampart server, without trailing slash.
    pub issuer: String,
    /// How long to cache JWKS keys. Defaults to 5 minutes.
    pub cache_ttl: Duration,
}

/// Core authentication verifier for Rampart tokens.
///
/// `RampartAuth` fetches JWKS from `{issuer}/.well-known/jwks.json`,
/// caches keys in memory, and verifies RS256 JWT tokens. It is
/// designed to be shared across requests via `Arc`.
///
/// Use the framework-specific middleware in the [`middleware`] module
/// to integrate with Actix-web or Axum, or call [`verify_token`](Self::verify_token)
/// directly for custom integrations.
#[derive(Clone)]
pub struct RampartAuth {
    config: RampartConfig,
    client: reqwest::Client,
    cache: Arc<RwLock<Option<CachedJwks>>>,
}

impl RampartAuth {
    /// Creates a new `RampartAuth` with the given issuer URL.
    ///
    /// Uses a default cache TTL of 5 minutes. For custom configuration,
    /// use [`RampartAuth::with_config`].
    pub fn new(issuer: &str) -> Self {
        Self::with_config(RampartConfig {
            issuer: issuer.trim_end_matches('/').to_string(),
            cache_ttl: Duration::from_secs(300),
        })
    }

    /// Creates a new `RampartAuth` with the given configuration.
    pub fn with_config(config: RampartConfig) -> Self {
        Self {
            config,
            client: reqwest::Client::new(),
            cache: Arc::new(RwLock::new(None)),
        }
    }

    /// Returns the configured issuer URL.
    pub fn issuer(&self) -> &str {
        &self.config.issuer
    }

    /// Extracts the Bearer token from an Authorization header value.
    pub fn extract_bearer(auth_header: &str) -> Result<&str, RampartError> {
        let parts: Vec<&str> = auth_header.splitn(2, ' ').collect();
        if parts.len() != 2 || !parts[0].eq_ignore_ascii_case("bearer") {
            return Err(RampartError::InvalidAuthHeader);
        }
        Ok(parts[1])
    }

    /// Verifies a JWT token string and returns the decoded claims.
    ///
    /// This method:
    /// 1. Fetches (and caches) the JWKS from `{issuer}/.well-known/jwks.json`.
    /// 2. Decodes the token header to find the `kid`.
    /// 3. Finds the matching key in the JWKS.
    /// 4. Validates the token signature, issuer, and expiration.
    /// 5. Returns the parsed [`Claims`].
    pub async fn verify_token(&self, token: &str) -> Result<Claims, RampartError> {
        let keys = self.get_jwks().await?;

        // Decode the token header to get the kid.
        let header = jsonwebtoken::decode_header(token)
            .map_err(RampartError::InvalidToken)?;

        // Find a matching key. Try kid first, then fall back to first RSA key.
        let key = if let Some(ref kid) = header.kid {
            keys.iter().find(|k| k.kid.as_deref() == Some(kid))
        } else {
            keys.iter().find(|k| k.kty == "RSA")
        }
        .ok_or(RampartError::NoMatchingKey)?;

        let decoding_key = DecodingKey::from_rsa_components(&key.n, &key.e)
            .map_err(RampartError::InvalidToken)?;

        let mut validation = Validation::new(Algorithm::RS256);
        validation.set_issuer(&[&self.config.issuer]);
        validation.set_required_spec_claims(&["exp", "iss", "sub"]);

        let token_data: TokenData<Claims> =
            decode(token, &decoding_key, &validation).map_err(RampartError::InvalidToken)?;

        Ok(token_data.claims)
    }

    /// Fetches JWKS keys, using the cache if still valid.
    async fn get_jwks(&self) -> Result<Vec<JwkKey>, RampartError> {
        // Check cache first.
        {
            let cache = self.cache.read().await;
            if let Some(ref cached) = *cache {
                if cached.fetched_at.elapsed() < self.config.cache_ttl {
                    return Ok(cached.keys.clone());
                }
            }
        }

        // Cache miss or expired — fetch fresh keys.
        let url = format!("{}/.well-known/jwks.json", self.config.issuer);
        let resp: JwksResponse = self
            .client
            .get(&url)
            .send()
            .await
            .map_err(RampartError::JwksFetchError)?
            .json()
            .await
            .map_err(RampartError::JwksFetchError)?;

        let keys = resp.keys;

        // Update cache.
        {
            let mut cache = self.cache.write().await;
            *cache = Some(CachedJwks {
                keys: keys.clone(),
                fetched_at: Instant::now(),
            });
        }

        Ok(keys)
    }
}
