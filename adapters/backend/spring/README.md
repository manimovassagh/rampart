# Rampart Spring Boot Starter

Spring Boot starter for integrating your Spring application with [Rampart](https://github.com/manimovassagh/rampart) IAM server as an OAuth2 Resource Server.

## Installation

Add the dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>com.rampart</groupId>
    <artifactId>rampart-spring-boot-starter</artifactId>
    <version>0.1.0</version>
</dependency>
```

## Configuration

Add to your `application.yml`:

```yaml
rampart:
  issuer: https://auth.example.com
  audience: my-api  # optional
```

The starter auto-configures:
- A `JwtDecoder` that validates tokens against Rampart's JWKS endpoint (`{issuer}/.well-known/jwks.json`)
- A `JwtAuthenticationConverter` that maps the `roles` claim to Spring Security `ROLE_` authorities

### Full `application.yml` Reference

```yaml
rampart:
  # (Required) The issuer URL of your Rampart server.
  # Used to build the JWKS endpoint: {issuer}/.well-known/jwks.json
  # Must match the "iss" claim in tokens exactly (trailing slash matters).
  issuer: https://auth.example.com

  # (Optional) Expected "aud" claim value. When set, tokens without a
  # matching audience are rejected.
  audience: my-api

# Standard Spring logging — useful for debugging token validation:
logging:
  level:
    org.springframework.security: DEBUG
    com.rampart.spring: DEBUG

# If your Rampart server uses a self-signed certificate during development:
server:
  ssl:
    trust-store: classpath:truststore.jks
    trust-store-password: changeit
```

### Configuration Properties

| Property           | Type     | Required | Default | Description                                          |
|--------------------|----------|----------|---------|------------------------------------------------------|
| `rampart.issuer`   | `String` | Yes      | --      | Rampart issuer URL. Activates the auto-configuration |
| `rampart.audience` | `String` | No       | --      | Expected `aud` claim value for token validation      |

## Usage

### Basic Security Configuration

```java
@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http,
            JwtAuthenticationConverter rampartJwtAuthenticationConverter) throws Exception {
        RampartSecurityConfigurer.applyDefaults(http, rampartJwtAuthenticationConverter);

        http.authorizeHttpRequests(auth -> auth
            .requestMatchers("/public/**").permitAll()
            .requestMatchers("/admin/**").hasRole("admin")
            .anyRequest().authenticated()
        );

        return http.build();
    }
}
```

### Accessing Claims in Controllers

```java
@RestController
public class UserController {

    @GetMapping("/me")
    public Map<String, Object> me(@AuthenticationPrincipal Jwt jwt) {
        RampartClaims claims = RampartClaims.fromJwt(jwt);
        return Map.of(
            "sub", claims.getSub(),
            "email", claims.getEmail(),
            "roles", claims.getRoles()
        );
    }
}
```

### Role-Based Access

The starter maps the `roles` claim from Rampart JWTs to Spring Security authorities with the `ROLE_` prefix. A token with `"roles": ["admin", "editor"]` produces authorities `ROLE_admin` and `ROLE_editor`.

Use `@PreAuthorize` or `.hasRole()` as usual:

```java
@PreAuthorize("hasRole('admin')")
@DeleteMapping("/users/{id}")
public void deleteUser(@PathVariable String id) {
    // only accessible to users with the "admin" role
}
```

## JWT Claims

The `RampartClaims` class provides typed access to Rampart-specific JWT claims:

| Claim               | Field              | Type           |
|---------------------|--------------------|----------------|
| `sub`               | `sub`              | `String`       |
| `org_id`            | `orgId`            | `String`       |
| `preferred_username`| `preferredUsername` | `String`       |
| `email`             | `email`            | `String`       |
| `email_verified`    | `emailVerified`    | `boolean`      |
| `given_name`        | `givenName`        | `String`       |
| `family_name`       | `familyName`       | `String`       |
| `roles`             | `roles`            | `List<String>` |

## Error Responses

By default, Spring Security returns errors in its own format (`WWW-Authenticate` header, HTML pages). To return JSON errors that match Rampart's `{error, error_description, status}` format, register a custom `AuthenticationEntryPoint` (401) and `AccessDeniedHandler` (403).

### Default Spring Security Error (what you get without customization)

```
HTTP/1.1 401
WWW-Authenticate: Bearer error="invalid_token", error_description="An error occurred..."
```

### Rampart-Style JSON Errors

Add custom handlers to your `SecurityFilterChain`:

```java
@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http,
            JwtAuthenticationConverter rampartJwtAuthenticationConverter) throws Exception {
        RampartSecurityConfigurer.applyDefaults(http, rampartJwtAuthenticationConverter);

        http.authorizeHttpRequests(auth -> auth
            .requestMatchers("/public/**").permitAll()
            .requestMatchers("/admin/**").hasRole("admin")
            .anyRequest().authenticated()
        );

        // Override default error handling with Rampart-style JSON responses
        http.oauth2ResourceServer(oauth2 -> oauth2
            .jwt(jwt -> jwt.jwtAuthenticationConverter(rampartJwtAuthenticationConverter))
            .authenticationEntryPoint(rampartAuthenticationEntryPoint())
            .accessDeniedHandler(rampartAccessDeniedHandler())
        );

        return http.build();
    }

    /**
     * Returns 401 as JSON: {"error":"unauthorized","error_description":"...","status":401}
     */
    @Bean
    public AuthenticationEntryPoint rampartAuthenticationEntryPoint() {
        return (request, response, authException) -> {
            response.setStatus(401);
            response.setContentType("application/json");
            response.getWriter().write(
                "{\"error\":\"unauthorized\","
                + "\"error_description\":\"Missing or invalid authorization header.\","
                + "\"status\":401}");
        };
    }

    /**
     * Returns 403 as JSON: {"error":"forbidden","error_description":"...","status":403}
     */
    @Bean
    public AccessDeniedHandler rampartAccessDeniedHandler() {
        return (request, response, accessDeniedException) -> {
            response.setStatus(403);
            response.setContentType("application/json");
            response.getWriter().write(
                "{\"error\":\"forbidden\","
                + "\"error_description\":\"" + accessDeniedException.getMessage() + "\","
                + "\"status\":403}");
        };
    }
}
```

> See `cookbook/spring-backend/src/main/java/com/rampart/sample/SecurityConfig.java` for a complete working example with CORS included.

## Troubleshooting

### JWKS fetch failure

**Symptom:** Application fails to start or first request returns 500 with `JwtDecoderInitializationException`.

**Cause:** The starter fetches keys from `{issuer}/.well-known/jwks.json` at first token validation. If the issuer URL is unreachable, this fails.

**Fixes:**
- Verify the URL is reachable: `curl -v https://auth.example.com/.well-known/jwks.json`
- If using HTTPS with a self-signed cert, add the CA to Java's truststore or configure `server.ssl.trust-store` in `application.yml`
- Check firewall/network rules between your Spring app and the Rampart server
- In Docker/Kubernetes, ensure DNS resolution works (`rampart` hostname vs `localhost`)

### Role not mapping

**Symptom:** `hasRole("admin")` returns 403 even though the user has the admin role.

**Cause:** The auto-configuration reads the `roles` claim (an array of strings) from the JWT and maps each entry to a `ROLE_` authority. If the claim is missing or has a different name, no authorities are granted.

**Fixes:**
- Decode your JWT at [jwt.io](https://jwt.io) and verify it contains a top-level `roles` claim as an array: `"roles": ["admin", "editor"]`
- Rampart includes `roles` by default. If you customized token claims on the server side, ensure `roles` is still present
- `hasRole("admin")` checks for `ROLE_admin` authority -- do not include the `ROLE_` prefix in your role names

### 401 on valid token

**Symptom:** A token that works against Rampart's `/me` endpoint returns 401 from Spring.

**Cause:** Issuer (`iss`) claim validation is strict. The `rampart.issuer` property must match the `iss` claim in the JWT exactly, including scheme, port, and trailing slash.

**Fixes:**
- Compare your config value against the actual `iss` claim in the token: a trailing slash (`https://auth.example.com/` vs `https://auth.example.com`) causes a mismatch
- The starter normalizes trailing slashes for the JWKS URL, but the issuer validation compares the raw `rampart.issuer` value against the token's `iss` claim
- Ensure your Rampart server's `issuer` configuration matches what you put in `application.yml`
- Check that the token has not expired (`exp` claim)

### CORS with Rampart

**Symptom:** Browser requests from your SPA to the Spring backend fail with CORS errors, even though tokens are valid.

**Cause:** Spring Security does not configure CORS by default. Your frontend origin must be explicitly allowed.

**Fix:** Add a `CorsConfigurationSource` bean and enable it in your security chain:

```java
@Bean
public CorsConfigurationSource corsConfigurationSource() {
    CorsConfiguration config = new CorsConfiguration();
    config.setAllowedOrigins(List.of("http://localhost:5173", "https://app.example.com"));
    config.setAllowedMethods(List.of("GET", "POST", "PUT", "DELETE", "OPTIONS"));
    config.setAllowedHeaders(List.of("Authorization", "Content-Type"));
    config.setAllowCredentials(true);

    UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
    source.registerCorsConfiguration("/**", config);
    return source;
}
```

Then enable CORS in your `SecurityFilterChain`:

```java
http.cors(cors -> cors.configurationSource(corsConfigurationSource()));
```

> For local development you can use `config.addAllowedOrigin("*")`, but never use wildcards in production when `allowCredentials` is true.

## Requirements

- Java 17+
- Spring Boot 3.3+
