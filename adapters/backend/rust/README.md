# rampart-rust

Rust adapter for [Rampart](https://github.com/manimovassagh/rampart) IAM server. Provides JWT verification middleware for **Actix-web 4+** and **Axum 0.7+** with RS256 signature validation, JWKS caching, and role-based access control.

## Installation

Add to your `Cargo.toml`:

```toml
[dependencies]
rampart-rust = { path = "../path/to/adapters/backend/rust" }
```

### Feature Flags

| Feature | Default | Description |
|---------|---------|-------------|
| `actix` | Yes | Actix-web 4 middleware |
| `axum` | Yes | Axum 0.7 layer and extractor |

To use only one framework:

```toml
# Axum only
rampart-rust = { path = "...", default-features = false, features = ["axum"] }

# Actix-web only
rampart-rust = { path = "...", default-features = false, features = ["actix"] }
```

## Axum Example

```rust
use axum::{Router, routing::get, Json, Extension};
use rampart_rust::RampartAuth;
use rampart_rust::middleware::axum::{RampartLayer, RampartClaims};
use rampart_rust::Claims;

async fn profile(RampartClaims(claims): RampartClaims) -> Json<Claims> {
    Json(claims)
}

#[tokio::main]
async fn main() {
    let auth = RampartAuth::new("https://auth.example.com");

    let app = Router::new()
        .route("/profile", get(profile))
        .layer(RampartLayer::new(auth));

    let listener = tokio::net::TcpListener::bind("127.0.0.1:3000")
        .await
        .unwrap();
    axum::serve(listener, app).await.unwrap();
}
```

### Role-Based Access Control (Axum)

```rust
use axum::{Router, routing::get, middleware};
use rampart_rust::RampartAuth;
use rampart_rust::middleware::axum::{RampartLayer, require_roles_middleware};

let auth = RampartAuth::new("https://auth.example.com");

let app = Router::new()
    .route("/admin", get(|| async { "admin area" }))
    .layer(middleware::from_fn(require_roles_middleware(&["admin"])))
    .layer(RampartLayer::new(auth));
```

## Actix-web Example

```rust
use actix_web::{web, App, HttpServer, HttpResponse};
use rampart_rust::RampartAuth;
use rampart_rust::middleware::actix::RampartMiddleware;
use rampart_rust::Claims;

async fn profile(claims: web::ReqData<Claims>) -> HttpResponse {
    HttpResponse::Ok().json(&*claims)
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let auth = RampartAuth::new("https://auth.example.com");

    HttpServer::new(move || {
        App::new()
            .wrap(RampartMiddleware::new(auth.clone()))
            .route("/profile", web::get().to(profile))
    })
    .bind("127.0.0.1:8080")?
    .run()
    .await
}
```

### Role-Based Access Control (Actix-web)

```rust
use actix_web::{web, App, HttpResponse};
use rampart_rust::RampartAuth;
use rampart_rust::middleware::actix::{RampartMiddleware, require_roles};
use rampart_rust::Claims;

async fn admin_handler(claims: web::ReqData<Claims>) -> HttpResponse {
    // require_roles can be called in the handler:
    require_roles(&["admin"])(claims.clone()).unwrap();
    HttpResponse::Ok().body("admin area")
}
```

## Direct Token Verification

For custom integrations outside of Actix-web or Axum:

```rust
use rampart_rust::RampartAuth;

async fn verify(token: &str) {
    let auth = RampartAuth::new("https://auth.example.com");

    match auth.verify_token(token).await {
        Ok(claims) => {
            println!("User: {}", claims.preferred_username);
            println!("Email: {}", claims.email);
            println!("Roles: {:?}", claims.roles);
        }
        Err(e) => {
            eprintln!("Auth failed: {e}");
        }
    }
}
```

## Error Format

All errors follow the Rampart server format:

```json
{
  "error": "unauthorized",
  "error_description": "Invalid or expired access token.",
  "status": 401
}
```

## Configuration

```rust
use std::time::Duration;
use rampart_rust::{RampartAuth, RampartConfig};

let auth = RampartAuth::with_config(RampartConfig {
    issuer: "https://auth.example.com".to_string(),
    cache_ttl: Duration::from_secs(600), // 10 minute JWKS cache
});
```

## License

MIT
