use actix_cors::Cors;
use actix_web::{web, App, HttpServer, HttpRequest, HttpResponse};
use jsonwebtoken::{decode, decode_header, Algorithm, DecodingKey, Validation};
use serde::{Deserialize, Serialize};
use std::sync::RwLock;
use std::env;

/// JWKS structures
#[derive(Debug, Deserialize, Clone)]
struct JwkKey {
    kid: String,
    n: String,
    e: String,
}

#[derive(Debug, Deserialize)]
struct JwksResponse {
    keys: Vec<JwkKey>,
}

/// JWT claims matching Rampart token format
#[derive(Debug, Serialize, Deserialize, Clone)]
struct Claims {
    iss: Option<String>,
    sub: Option<String>,
    iat: Option<serde_json::Value>,
    exp: Option<serde_json::Value>,
    org_id: Option<String>,
    preferred_username: Option<String>,
    email: Option<String>,
    email_verified: Option<bool>,
    given_name: Option<String>,
    family_name: Option<String>,
    roles: Option<Vec<String>>,
}

/// Cached JWKS keys
struct AppState {
    issuer: String,
    jwks_keys: RwLock<Vec<JwkKey>>,
}

/// Fetch JWKS keys from Rampart
async fn fetch_jwks(issuer: &str) -> Result<Vec<JwkKey>, String> {
    let url = format!("{}/.well-known/jwks.json", issuer);
    let client = reqwest::Client::new();
    let resp = client.get(&url).send().await.map_err(|e| format!("JWKS fetch error: {}", e))?;
    let jwks: JwksResponse = resp.json().await.map_err(|e| format!("JWKS parse error: {}", e))?;
    Ok(jwks.keys)
}

/// Extract and verify JWT from Authorization header
async fn verify_token(req: &HttpRequest, state: &web::Data<AppState>) -> Result<Claims, HttpResponse> {
    let auth_header = req.headers().get("Authorization")
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| HttpResponse::Unauthorized().json(serde_json::json!({"error": "Missing Authorization header"})))?;

    if !auth_header.starts_with("Bearer ") {
        return Err(HttpResponse::Unauthorized().json(serde_json::json!({"error": "Invalid Authorization format"})));
    }
    let token = &auth_header[7..];

    let header = decode_header(token)
        .map_err(|e| HttpResponse::Unauthorized().json(serde_json::json!({"error": format!("Invalid token header: {}", e)})))?;

    let kid = header.kid
        .ok_or_else(|| HttpResponse::Unauthorized().json(serde_json::json!({"error": "Token missing kid"})))?;

    // Try cached keys first, refresh if kid not found
    let key = {
        let keys = state.jwks_keys.read().unwrap();
        keys.iter().find(|k| k.kid == kid).cloned()
    };

    let jwk = match key {
        Some(k) => k,
        None => {
            // Refresh JWKS
            let new_keys = fetch_jwks(&state.issuer).await
                .map_err(|e| HttpResponse::InternalServerError().json(serde_json::json!({"error": e})))?;
            let found = new_keys.iter().find(|k| k.kid == kid).cloned();
            {
                let mut keys = state.jwks_keys.write().unwrap();
                *keys = new_keys;
            }
            found.ok_or_else(|| HttpResponse::Unauthorized().json(serde_json::json!({"error": "No matching key found in JWKS"})))?
        }
    };

    let decoding_key = DecodingKey::from_rsa_components(&jwk.n, &jwk.e)
        .map_err(|e| HttpResponse::InternalServerError().json(serde_json::json!({"error": format!("RSA key error: {}", e)})))?;

    let mut validation = Validation::new(Algorithm::RS256);
    validation.set_issuer(&[&state.issuer]);
    validation.validate_exp = true;
    validation.validate_aud = false;

    let token_data = decode::<Claims>(token, &decoding_key, &validation)
        .map_err(|e| HttpResponse::Unauthorized().json(serde_json::json!({"error": format!("Token validation failed: {}", e)})))?;

    Ok(token_data.claims)
}

// ── Route Handlers ──

async fn health(state: web::Data<AppState>) -> HttpResponse {
    HttpResponse::Ok().json(serde_json::json!({
        "status": "ok",
        "issuer": state.issuer
    }))
}

async fn profile(req: HttpRequest, state: web::Data<AppState>) -> HttpResponse {
    let claims = match verify_token(&req, &state).await {
        Ok(c) => c,
        Err(resp) => return resp,
    };

    let roles = claims.roles.clone().unwrap_or_default();

    HttpResponse::Ok().json(serde_json::json!({
        "message": "Authenticated!",
        "user": {
            "id": claims.sub,
            "email": claims.email,
            "username": claims.preferred_username,
            "org_id": claims.org_id,
            "email_verified": claims.email_verified,
            "given_name": claims.given_name,
            "family_name": claims.family_name,
            "roles": roles,
        }
    }))
}

async fn claims_handler(req: HttpRequest, state: web::Data<AppState>) -> HttpResponse {
    let claims = match verify_token(&req, &state).await {
        Ok(c) => c,
        Err(resp) => return resp,
    };

    let roles = claims.roles.clone().unwrap_or_default();

    HttpResponse::Ok().json(serde_json::json!({
        "iss": claims.iss,
        "sub": claims.sub,
        "iat": claims.iat,
        "exp": claims.exp,
        "org_id": claims.org_id,
        "preferred_username": claims.preferred_username,
        "email": claims.email,
        "email_verified": claims.email_verified,
        "roles": roles,
    }))
}

async fn editor_dashboard(req: HttpRequest, state: web::Data<AppState>) -> HttpResponse {
    let claims = match verify_token(&req, &state).await {
        Ok(c) => c,
        Err(resp) => return resp,
    };

    let roles = claims.roles.clone().unwrap_or_default();
    if !roles.contains(&"editor".to_string()) {
        return HttpResponse::Forbidden().json(serde_json::json!({
            "error": "Forbidden: requires editor role"
        }));
    }

    HttpResponse::Ok().json(serde_json::json!({
        "message": "Welcome, Editor!",
        "user": claims.preferred_username,
        "roles": roles,
        "data": {
            "drafts": 3,
            "published": 12,
            "pending_review": 2,
        }
    }))
}

async fn manager_reports(req: HttpRequest, state: web::Data<AppState>) -> HttpResponse {
    let claims = match verify_token(&req, &state).await {
        Ok(c) => c,
        Err(resp) => return resp,
    };

    let roles = claims.roles.clone().unwrap_or_default();
    if !roles.contains(&"manager".to_string()) {
        return HttpResponse::Forbidden().json(serde_json::json!({
            "error": "Forbidden: requires manager role"
        }));
    }

    HttpResponse::Ok().json(serde_json::json!({
        "message": "Manager Reports",
        "user": claims.preferred_username,
        "roles": roles,
        "reports": [
            {"name": "Q1 Revenue", "status": "complete"},
            {"name": "User Growth", "status": "in_progress"},
        ]
    }))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let issuer = env::var("RAMPART_ISSUER").unwrap_or_else(|_| "http://localhost:8080".to_string());
    let port: u16 = env::var("PORT").unwrap_or_else(|_| "3001".to_string())
        .parse().expect("PORT must be a number");

    // Pre-fetch JWKS
    let keys = fetch_jwks(&issuer).await.unwrap_or_else(|e| {
        eprintln!("Warning: could not pre-fetch JWKS: {}. Will retry on first request.", e);
        vec![]
    });

    let state = web::Data::new(AppState {
        issuer: issuer.clone(),
        jwks_keys: RwLock::new(keys),
    });

    println!("Sample Rust backend running on http://localhost:{}", port);
    println!("Rampart issuer: {}", issuer);
    println!("\nRoutes:");
    println!("  GET /api/health            — public");
    println!("  GET /api/profile           — protected (any authenticated user)");
    println!("  GET /api/claims            — protected (any authenticated user)");
    println!("  GET /api/editor/dashboard  — protected (requires \"editor\" role)");
    println!("  GET /api/manager/reports   — protected (requires \"manager\" role)");

    HttpServer::new(move || {
        // WARNING: Restrict to your frontend domain in production. Never use "*" in production.
        let cors = Cors::default()
            .allow_any_origin()
            .allow_any_method()
            .allow_any_header();

        App::new()
            .wrap(cors)
            .app_data(state.clone())
            .route("/api/health", web::get().to(health))
            .route("/api/profile", web::get().to(profile))
            .route("/api/claims", web::get().to(claims_handler))
            .route("/api/editor/dashboard", web::get().to(editor_dashboard))
            .route("/api/manager/reports", web::get().to(manager_reports))
    })
    .bind(("0.0.0.0", port))?
    .run()
    .await
}
