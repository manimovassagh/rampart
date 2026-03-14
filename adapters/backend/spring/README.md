# Rampart Spring Boot Starter

Spring Boot starter for integrating your Spring application with [Rampart](https://github.com/manimovassagh/rampart) IAM server as an OAuth2 Resource Server.

## Installation

Add the dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>io.github.manimovassagh</groupId>
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

## Requirements

- Java 17+
- Spring Boot 3.3+
