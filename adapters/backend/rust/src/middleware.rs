//! Framework-specific middleware for Actix-web and Axum.

use crate::{Claims, ErrorResponse, RampartAuth, RampartError};

/// Role-based access guard that checks for required roles in the claims.
///
/// Use after authentication middleware to enforce that the user has
/// all specified roles.
#[derive(Debug, Clone)]
pub struct RequireRoles {
    roles: Vec<String>,
}

impl RequireRoles {
    /// Creates a new role guard requiring all of the specified roles.
    pub fn new(roles: &[&str]) -> Self {
        Self {
            roles: roles.iter().map(|s| s.to_string()).collect(),
        }
    }

    /// Checks claims against required roles. Returns `Ok(())` if the user
    /// has all required roles, or an appropriate error otherwise.
    pub fn check(&self, claims: &Claims) -> Result<(), RampartError> {
        let role_refs: Vec<&str> = self.roles.iter().map(|s| s.as_str()).collect();
        let missing = claims.missing_roles(&role_refs);
        if missing.is_empty() {
            Ok(())
        } else {
            Err(RampartError::MissingRoles(missing.join(", ")))
        }
    }
}

// ---------------------------------------------------------------------------
// Actix-web middleware
// ---------------------------------------------------------------------------
#[cfg(feature = "actix")]
pub mod actix {
    //! Actix-web middleware for Rampart JWT authentication.
    //!
    //! # Example
    //!
    //! ```rust,no_run
    //! use actix_web::{web, App, HttpServer, HttpResponse};
    //! use rampart_rust::RampartAuth;
    //! use rampart_rust::middleware::actix::RampartMiddleware;
    //! use rampart_rust::Claims;
    //!
    //! async fn profile(claims: web::ReqData<Claims>) -> HttpResponse {
    //!     HttpResponse::Ok().json(&*claims)
    //! }
    //!
    //! #[actix_web::main]
    //! async fn main() -> std::io::Result<()> {
    //!     let auth = RampartAuth::new("https://auth.example.com");
    //!
    //!     HttpServer::new(move || {
    //!         App::new()
    //!             .wrap(RampartMiddleware::new(auth.clone()))
    //!             .route("/profile", web::get().to(profile))
    //!     })
    //!     .bind("127.0.0.1:8080")?
    //!     .run()
    //!     .await
    //! }
    //! ```

    use super::*;
    use actix_web::body::EitherBody;
    use actix_web::dev::{forward_ready, Service, ServiceRequest, ServiceResponse, Transform};
    use actix_web::{web, HttpMessage, HttpResponse};
    use std::future::{ready, Future, Ready};
    use std::pin::Pin;
    use std::rc::Rc;

    /// Actix-web middleware that verifies Rampart JWT tokens.
    #[derive(Clone)]
    pub struct RampartMiddleware {
        auth: RampartAuth,
    }

    impl RampartMiddleware {
        /// Creates a new Actix-web middleware with the given auth verifier.
        pub fn new(auth: RampartAuth) -> Self {
            Self { auth }
        }
    }

    impl<S, B> Transform<S, ServiceRequest> for RampartMiddleware
    where
        S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = actix_web::Error>
            + 'static,
        S::Future: 'static,
        B: 'static,
    {
        type Response = ServiceResponse<EitherBody<B>>;
        type Error = actix_web::Error;
        type Transform = RampartMiddlewareService<S>;
        type InitError = ();
        type Future = Ready<Result<Self::Transform, Self::InitError>>;

        fn new_transform(&self, service: S) -> Self::Future {
            ready(Ok(RampartMiddlewareService {
                service: Rc::new(service),
                auth: self.auth.clone(),
            }))
        }
    }

    /// Inner service created by [`RampartMiddleware`].
    pub struct RampartMiddlewareService<S> {
        service: Rc<S>,
        auth: RampartAuth,
    }

    impl<S, B> Service<ServiceRequest> for RampartMiddlewareService<S>
    where
        S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = actix_web::Error>
            + 'static,
        S::Future: 'static,
        B: 'static,
    {
        type Response = ServiceResponse<EitherBody<B>>;
        type Error = actix_web::Error;
        type Future = Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>>>>;

        forward_ready!(service);

        fn call(&self, req: ServiceRequest) -> Self::Future {
            let auth = self.auth.clone();
            let service = Rc::clone(&self.service);

            Box::pin(async move {
                let auth_header = req
                    .headers()
                    .get("Authorization")
                    .and_then(|v| v.to_str().ok())
                    .map(|s| s.to_string());

                let header_value = match auth_header {
                    Some(h) => h,
                    None => {
                        let resp = ErrorResponse::unauthorized("Missing authorization header.");
                        let http_resp = HttpResponse::Unauthorized().json(resp);
                        return Ok(req.into_response(http_resp).map_into_right_body());
                    }
                };

                let token = match RampartAuth::extract_bearer(&header_value) {
                    Ok(t) => t.to_string(),
                    Err(e) => {
                        let resp = e.to_error_response();
                        let http_resp = HttpResponse::Unauthorized().json(resp);
                        return Ok(req.into_response(http_resp).map_into_right_body());
                    }
                };

                match auth.verify_token(&token).await {
                    Ok(claims) => {
                        req.extensions_mut().insert(claims);
                        let res = service.call(req).await?;
                        Ok(res.map_into_left_body())
                    }
                    Err(e) => {
                        let resp = e.to_error_response();
                        let http_resp = if e.status_code() == 403 {
                            HttpResponse::Forbidden().json(resp)
                        } else {
                            HttpResponse::Unauthorized().json(resp)
                        };
                        Ok(req.into_response(http_resp).map_into_right_body())
                    }
                }
            })
        }
    }

    /// Checks claims for required roles. Returns the claims unchanged on
    /// success, or an Actix error on failure.
    pub fn require_roles(
        roles: &[&str],
    ) -> impl Fn(
        web::ReqData<Claims>,
    ) -> Result<web::ReqData<Claims>, actix_web::Error>
           + Clone {
        let guard = super::RequireRoles::new(roles);
        move |claims: web::ReqData<Claims>| {
            guard
                .check(&claims)
                .map(|_| claims)
                .map_err(|e| {
                    let resp = e.to_error_response();
                    actix_web::error::InternalError::from_response(
                        e,
                        HttpResponse::Forbidden().json(resp),
                    )
                    .into()
                })
        }
    }
}

// ---------------------------------------------------------------------------
// Axum extractor and middleware layer
// ---------------------------------------------------------------------------
#[cfg(feature = "axum")]
pub mod axum {
    //! Axum middleware and extractors for Rampart JWT authentication.
    //!
    //! # Example
    //!
    //! ```rust,no_run
    //! use axum::{Router, routing::get, Json};
    //! use rampart_rust::RampartAuth;
    //! use rampart_rust::middleware::axum::{RampartLayer, RampartClaims};
    //! use rampart_rust::Claims;
    //!
    //! async fn profile(RampartClaims(claims): RampartClaims) -> Json<Claims> {
    //!     Json(claims)
    //! }
    //!
    //! #[tokio::main]
    //! async fn main() {
    //!     let auth = RampartAuth::new("https://auth.example.com");
    //!
    //!     let app = Router::new()
    //!         .route("/profile", get(profile))
    //!         .layer(RampartLayer::new(auth));
    //!
    //!     let listener = tokio::net::TcpListener::bind("127.0.0.1:3000")
    //!         .await
    //!         .unwrap();
    //!     axum::serve(listener, app).await.unwrap();
    //! }
    //! ```

    use super::*;
    use ::axum::extract::Request;
    use ::axum::http::StatusCode;
    use ::axum::response::{IntoResponse, Json, Response};
    use std::future::Future;
    use std::pin::Pin;
    use std::task::{Context, Poll};
    use tower_layer::Layer;
    use tower_service::Service;

    /// Tower [`Layer`] that adds Rampart JWT authentication to an Axum router.
    #[derive(Clone)]
    pub struct RampartLayer {
        auth: RampartAuth,
    }

    impl RampartLayer {
        /// Creates a new Axum layer with the given auth verifier.
        pub fn new(auth: RampartAuth) -> Self {
            Self { auth }
        }
    }

    impl<S> Layer<S> for RampartLayer {
        type Service = RampartService<S>;

        fn layer(&self, inner: S) -> Self::Service {
            RampartService {
                inner,
                auth: self.auth.clone(),
            }
        }
    }

    /// Tower [`Service`] that performs Rampart JWT verification.
    #[derive(Clone)]
    pub struct RampartService<S> {
        inner: S,
        auth: RampartAuth,
    }

    impl<S> Service<Request> for RampartService<S>
    where
        S: Service<Request, Response = Response> + Clone + Send + 'static,
        S::Future: Send + 'static,
    {
        type Response = Response;
        type Error = S::Error;
        type Future = Pin<Box<dyn Future<Output = Result<Response, S::Error>> + Send>>;

        fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
            self.inner.poll_ready(cx)
        }

        fn call(&mut self, mut req: Request) -> Self::Future {
            let auth = self.auth.clone();
            let mut inner = self.inner.clone();

            Box::pin(async move {
                let auth_header = req
                    .headers()
                    .get("Authorization")
                    .and_then(|v| v.to_str().ok())
                    .map(|s| s.to_string());

                let header_value = match auth_header {
                    Some(h) => h,
                    None => {
                        return Ok(error_response(
                            StatusCode::UNAUTHORIZED,
                            ErrorResponse::unauthorized("Missing authorization header."),
                        ));
                    }
                };

                let token = match RampartAuth::extract_bearer(&header_value) {
                    Ok(t) => t.to_string(),
                    Err(e) => {
                        let resp = e.to_error_response();
                        return Ok(error_response(StatusCode::UNAUTHORIZED, resp));
                    }
                };

                match auth.verify_token(&token).await {
                    Ok(claims) => {
                        req.extensions_mut().insert(claims);
                        inner.call(req).await
                    }
                    Err(e) => {
                        let resp = e.to_error_response();
                        let status = if e.status_code() == 403 {
                            StatusCode::FORBIDDEN
                        } else {
                            StatusCode::UNAUTHORIZED
                        };
                        Ok(error_response(status, resp))
                    }
                }
            })
        }
    }

    /// Axum extractor that pulls [`Claims`] from request extensions.
    ///
    /// Use this in handler function signatures to access the authenticated
    /// user's claims. Requires [`RampartLayer`] to be applied to the router.
    ///
    /// # Example
    ///
    /// ```rust,no_run
    /// use axum::Json;
    /// use rampart_rust::middleware::axum::RampartClaims;
    /// use rampart_rust::Claims;
    ///
    /// async fn handler(RampartClaims(claims): RampartClaims) -> Json<Claims> {
    ///     Json(claims)
    /// }
    /// ```
    pub struct RampartClaims(pub Claims);

    #[::axum::async_trait]
    impl<S> ::axum::extract::FromRequestParts<S> for RampartClaims
    where
        S: Send + Sync,
    {
        type Rejection = Response;

        async fn from_request_parts(
            parts: &mut ::axum::http::request::Parts,
            _state: &S,
        ) -> Result<Self, Self::Rejection> {
            parts
                .extensions
                .get::<Claims>()
                .cloned()
                .map(RampartClaims)
                .ok_or_else(|| {
                    error_response(
                        StatusCode::UNAUTHORIZED,
                        ErrorResponse::unauthorized("Authentication required."),
                    )
                })
        }
    }

    /// Returns a closure suitable for use with `axum::middleware::from_fn`
    /// that checks for required roles in the request extensions.
    ///
    /// Must be layered after [`RampartLayer`].
    pub fn require_roles_middleware(
        roles: &[&str],
    ) -> impl Fn(
        Request,
        ::axum::middleware::Next,
    ) -> Pin<Box<dyn Future<Output = Response> + Send>>
           + Clone
           + Send {
        let guard = super::RequireRoles::new(roles);
        move |req: Request, next: ::axum::middleware::Next| {
            let guard = guard.clone();
            Box::pin(async move {
                let claims = req.extensions().get::<Claims>();
                match claims {
                    None => error_response(
                        StatusCode::UNAUTHORIZED,
                        ErrorResponse::unauthorized("Authentication required."),
                    ),
                    Some(claims) => match guard.check(claims) {
                        Ok(()) => next.run(req).await,
                        Err(e) => {
                            let resp = e.to_error_response();
                            error_response(StatusCode::FORBIDDEN, resp)
                        }
                    },
                }
            })
        }
    }

    fn error_response(status: StatusCode, body: ErrorResponse) -> Response {
        let mut resp = Json(body).into_response();
        *resp.status_mut() = status;
        resp
    }
}
