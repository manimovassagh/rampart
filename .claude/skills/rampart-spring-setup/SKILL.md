---
name: rampart-spring-setup
description: Add Rampart authentication to a Spring Boot app. Configures Spring Security OAuth2 Resource Server with JWT validation against Rampart's JWKS. Use when securing a Java/Kotlin Spring Boot API with Rampart.
argument-hint: [issuer-url]
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Add Rampart Authentication to a Spring Boot App

Configure Spring Security to validate Rampart JWTs using the OAuth2 Resource Server.

## What This Skill Does

1. Adds Spring Security OAuth2 Resource Server dependency
2. Configures `application.yml` with Rampart's issuer/JWKS
3. Sets up `SecurityFilterChain` with JWT validation
4. Creates a helper to extract Rampart claims from the security context
5. Protects endpoints with `@PreAuthorize` or `SecurityFilterChain` rules

## Step-by-Step

### 1. Add dependencies

#### Maven (`pom.xml`):

```xml
<dependency>
  <groupId>org.springframework.boot</groupId>
  <artifactId>spring-boot-starter-oauth2-resource-server</artifactId>
</dependency>
```

#### Gradle (`build.gradle.kts`):

```kotlin
implementation("org.springframework.boot:spring-boot-starter-oauth2-resource-server")
```

### 2. Configure application properties

Add to `application.yml` (or `application.properties`):

```yaml
spring:
  security:
    oauth2:
      resourceserver:
        jwt:
          issuer-uri: ${RAMPART_ISSUER:http://localhost:8080}
          jwk-set-uri: ${RAMPART_ISSUER:http://localhost:8080}/.well-known/jwks.json
```

If `$ARGUMENTS` is provided, use it instead of the default.

### 3. Create Security Configuration

Create `config/SecurityConfig.java`:

```java
package com.example.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .csrf(csrf -> csrf.disable())
            .sessionManagement(sm -> sm.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/api/public/**", "/health", "/actuator/**").permitAll()
                .requestMatchers("/api/**").authenticated()
                .anyRequest().permitAll()
            )
            .oauth2ResourceServer(oauth2 -> oauth2.jwt(jwt -> {}));

        return http.build();
    }
}
```

### 4. Create a claims helper

Create `security/RampartClaims.java`:

```java
package com.example.security;

import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.jwt.Jwt;

public class RampartClaims {

    public static Jwt getJwt() {
        var auth = SecurityContextHolder.getContext().getAuthentication();
        return (Jwt) auth.getPrincipal();
    }

    public static String getUserId() {
        return getJwt().getSubject();
    }

    public static String getEmail() {
        return getJwt().getClaimAsString("email");
    }

    public static String getUsername() {
        return getJwt().getClaimAsString("preferred_username");
    }

    public static String getOrgId() {
        return getJwt().getClaimAsString("org_id");
    }
}
```

### 5. Use in a controller

```java
@RestController
@RequestMapping("/api")
public class ProfileController {

    @GetMapping("/profile")
    public Map<String, String> profile(@AuthenticationPrincipal Jwt jwt) {
        return Map.of(
            "id", jwt.getSubject(),
            "email", jwt.getClaimAsString("email"),
            "username", jwt.getClaimAsString("preferred_username"),
            "org", jwt.getClaimAsString("org_id")
        );
    }

    @GetMapping("/admin")
    @PreAuthorize("hasAuthority('SCOPE_admin')")
    public Map<String, String> admin(@AuthenticationPrincipal Jwt jwt) {
        return Map.of("message", "Hello admin " + jwt.getClaimAsString("preferred_username"));
    }
}
```

### 6. Custom error response (optional)

Create `config/AuthEntryPoint.java` to return Rampart-style 401 JSON:

```java
@Component
public class AuthEntryPoint implements AuthenticationEntryPoint {
    @Override
    public void commence(HttpServletRequest req, HttpServletResponse res, AuthenticationException ex)
            throws IOException {
        res.setContentType("application/json");
        res.setStatus(401);
        res.getWriter().write(
            "{\"error\":\"unauthorized\",\"error_description\":\"Invalid or expired access token.\",\"status\":401}"
        );
    }
}
```

Then wire it in `SecurityConfig`:

```java
.oauth2ResourceServer(oauth2 -> oauth2
    .jwt(jwt -> {})
    .authenticationEntryPoint(authEntryPoint)
)
```

## Checklist

- [ ] `spring-boot-starter-oauth2-resource-server` added
- [ ] Issuer URI and JWK set URI configured
- [ ] `SecurityFilterChain` with JWT validation
- [ ] Public vs protected endpoints defined
- [ ] Controllers use `@AuthenticationPrincipal Jwt` for claims
- [ ] Environment variable `RAMPART_ISSUER` set
