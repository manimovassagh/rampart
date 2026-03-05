---
sidebar_position: 7
title: Spring Boot
description: Integrate Rampart authentication into Spring Boot applications using OAuth2 Resource Server and Spring Security.
---

# Spring Boot Adapter

The `rampart-spring-boot-starter` provides auto-configuration for integrating Rampart with Spring Boot applications. It builds on Spring Security's OAuth2 Resource Server support, adding Rampart-specific claim mapping, role extraction, and multi-tenancy support.

## Installation

### Maven

```xml
<dependency>
    <groupId>com.rampart</groupId>
    <artifactId>rampart-spring-boot-starter</artifactId>
    <version>0.1.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'com.rampart:rampart-spring-boot-starter:0.1.0'
```

The starter brings in `spring-boot-starter-oauth2-resource-server` and `spring-boot-starter-security` transitively.

## Quick Start

### 1. Configure `application.yml`

```yaml
rampart:
  issuer-url: https://auth.example.com
  audience: my-api
  realm: default
```

This is all you need. The starter auto-discovers the OIDC endpoints via `{issuer-url}/.well-known/openid-configuration` and configures JWKS-based JWT verification.

### 2. Create a Security Configuration

```java
package com.example.config;

import com.rampart.spring.RampartSecurityConfigurer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.web.SecurityFilterChain;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain securityFilterChain(
            HttpSecurity http,
            RampartSecurityConfigurer rampart
    ) throws Exception {
        http
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/health", "/actuator/**").permitAll()
                .requestMatchers("/api/admin/**").hasRole("ADMIN")
                .requestMatchers("/api/**").authenticated()
                .anyRequest().permitAll()
            )
            .with(rampart, configurer -> {});

        return http.build();
    }
}
```

### 3. Create a Controller

```java
package com.example.controller;

import com.rampart.spring.RampartUser;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
public class ProfileController {

    @GetMapping("/api/profile")
    public Map<String, Object> profile(@AuthenticationPrincipal RampartUser user) {
        return Map.of(
            "userId", user.getId(),
            "email", user.getEmail(),
            "name", user.getName(),
            "roles", user.getRoles()
        );
    }
}
```

## Configuration Reference

### `application.yml`

```yaml
rampart:
  # Required
  issuer-url: https://auth.example.com
  audience: my-api

  # Optional
  realm: default                          # Organization/realm
  clock-tolerance: 5s                     # Clock skew tolerance (default: 5s)
  jwks-cache-ttl: 10m                     # JWKS cache duration (default: 10m)
  required-claims:                        # Claims that must be present
    - email
  role-claim: roles                       # JWT claim containing roles (default: roles)
  role-prefix: ROLE_                      # Spring Security role prefix (default: ROLE_)
```

### Environment Variables

All properties can be set via environment variables:

```bash
RAMPART_ISSUER_URL=https://auth.example.com
RAMPART_AUDIENCE=my-api
RAMPART_REALM=default
```

## SecurityFilterChain Configuration

### Basic Setup

```java
@Bean
public SecurityFilterChain securityFilterChain(
        HttpSecurity http,
        RampartSecurityConfigurer rampart
) throws Exception {
    http
        .authorizeHttpRequests(auth -> auth
            .requestMatchers("/health").permitAll()
            .requestMatchers("/api/**").authenticated()
            .anyRequest().denyAll()
        )
        .with(rampart, configurer -> {});

    return http.build();
}
```

### With CORS

```java
@Bean
public SecurityFilterChain securityFilterChain(
        HttpSecurity http,
        RampartSecurityConfigurer rampart
) throws Exception {
    http
        .cors(cors -> cors.configurationSource(corsConfigurationSource()))
        .csrf(csrf -> csrf.disable())
        .authorizeHttpRequests(auth -> auth
            .requestMatchers("/health").permitAll()
            .requestMatchers("/api/**").authenticated()
        )
        .with(rampart, configurer -> {});

    return http.build();
}

@Bean
public CorsConfigurationSource corsConfigurationSource() {
    CorsConfiguration config = new CorsConfiguration();
    config.setAllowedOrigins(List.of("http://localhost:5173"));
    config.setAllowedMethods(List.of("GET", "POST", "PUT", "DELETE"));
    config.setAllowedHeaders(List.of("Authorization", "Content-Type"));
    config.setAllowCredentials(true);

    UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
    source.registerCorsConfiguration("/**", config);
    return source;
}
```

### Role-Based Access in SecurityFilterChain

```java
http.authorizeHttpRequests(auth -> auth
    .requestMatchers("/health").permitAll()
    .requestMatchers("/api/admin/**").hasRole("ADMIN")
    .requestMatchers("/api/billing/**").hasAnyRole("ADMIN", "BILLING")
    .requestMatchers(HttpMethod.GET, "/api/**").authenticated()
    .requestMatchers(HttpMethod.POST, "/api/**").hasAuthority("SCOPE_write")
    .anyRequest().denyAll()
);
```

## Method-Level Security with `@PreAuthorize`

Enable method security with `@EnableMethodSecurity` on your configuration class.

### Role Checks

```java
@RestController
@RequestMapping("/api/admin")
public class AdminController {

    @GetMapping("/stats")
    @PreAuthorize("hasRole('ADMIN')")
    public Map<String, Object> getStats() {
        return Map.of(
            "totalUsers", 1234,
            "activeToday", 567
        );
    }

    @DeleteMapping("/users/{id}")
    @PreAuthorize("hasRole('SUPER_ADMIN')")
    public Map<String, String> deleteUser(@PathVariable String id) {
        return Map.of("deleted", id);
    }

    @GetMapping("/reports")
    @PreAuthorize("hasAnyRole('ADMIN', 'ANALYST')")
    public Map<String, Object> getReports() {
        return Map.of("reports", List.of());
    }
}
```

### Scope Checks

```java
@PostMapping("/api/emails/send")
@PreAuthorize("hasAuthority('SCOPE_email:send')")
public Map<String, Boolean> sendEmail() {
    return Map.of("sent", true);
}
```

### Custom SpEL Expressions

```java
// Only allow users to access their own data
@GetMapping("/api/users/{userId}/tasks")
@PreAuthorize("#userId == authentication.principal.id")
public List<Task> getUserTasks(@PathVariable String userId) {
    return taskService.findByUserId(userId);
}

// Combine role and ownership checks
@PutMapping("/api/tasks/{taskId}")
@PreAuthorize("hasRole('ADMIN') or @taskService.isOwner(#taskId, authentication.principal.id)")
public Task updateTask(@PathVariable String taskId, @RequestBody TaskUpdate update) {
    return taskService.update(taskId, update);
}
```

## RampartUser Principal

The `RampartUser` object is available as the `@AuthenticationPrincipal` in any controller method:

```java
public class RampartUser {
    public String getId();              // sub claim
    public String getEmail();           // email claim
    public String getName();            // name claim
    public List<String> getRoles();     // roles claim
    public String getScope();           // scope claim
    public String getOrgId();           // org_id claim
    public String getIssuer();          // iss claim
    public String getAudience();        // aud claim

    public boolean hasRole(String role);
    public boolean hasAnyRole(String... roles);
    public boolean hasScope(String scope);
    public boolean hasAllScopes(String... scopes);
}
```

## Full Working Example

### `application.yml`

```yaml
server:
  port: 8080

rampart:
  issuer-url: ${RAMPART_URL:https://auth.example.com}
  audience: ${RAMPART_CLIENT_ID:task-api}
  realm: default

logging:
  level:
    com.rampart: DEBUG
```

### Security Configuration

```java
package com.example.taskapi.config;

import com.rampart.spring.RampartSecurityConfigurer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpMethod;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.CorsConfigurationSource;
import org.springframework.web.cors.UrlBasedCorsConfigurationSource;

import java.util.List;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain securityFilterChain(
            HttpSecurity http,
            RampartSecurityConfigurer rampart
    ) throws Exception {
        http
            .cors(cors -> cors.configurationSource(corsConfigurationSource()))
            .csrf(csrf -> csrf.disable())
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/health", "/actuator/**").permitAll()
                .requestMatchers("/api/admin/**").hasRole("ADMIN")
                .requestMatchers(HttpMethod.POST, "/api/tasks").hasAuthority("SCOPE_tasks:write")
                .requestMatchers("/api/**").authenticated()
                .anyRequest().denyAll()
            )
            .with(rampart, configurer -> {});

        return http.build();
    }

    @Bean
    public CorsConfigurationSource corsConfigurationSource() {
        CorsConfiguration config = new CorsConfiguration();
        config.setAllowedOrigins(List.of("http://localhost:5173"));
        config.setAllowedMethods(List.of("GET", "POST", "PUT", "DELETE"));
        config.setAllowedHeaders(List.of("Authorization", "Content-Type"));
        config.setAllowCredentials(true);

        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/**", config);
        return source;
    }
}
```

### Task Controller

```java
package com.example.taskapi.controller;

import com.rampart.spring.RampartUser;
import org.springframework.http.HttpStatus;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.*;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

@RestController
public class TaskController {

    private final Map<String, List<Map<String, String>>> tasks = new ConcurrentHashMap<>();

    @GetMapping("/health")
    public Map<String, String> health() {
        return Map.of("status", "ok");
    }

    @GetMapping("/api/tasks")
    public Map<String, Object> listTasks(@AuthenticationPrincipal RampartUser user) {
        List<Map<String, String>> userTasks = tasks.getOrDefault(user.getId(), List.of());
        return Map.of("tasks", userTasks);
    }

    @PostMapping("/api/tasks")
    @ResponseStatus(HttpStatus.CREATED)
    public Map<String, String> createTask(
            @AuthenticationPrincipal RampartUser user,
            @RequestBody Map<String, String> body
    ) {
        tasks.computeIfAbsent(user.getId(), k -> new ArrayList<>());

        Map<String, String> task = Map.of(
            "id", UUID.randomUUID().toString(),
            "title", body.get("title"),
            "assignee", user.getId()
        );

        tasks.get(user.getId()).add(task);
        return task;
    }

    @GetMapping("/api/admin/stats")
    @PreAuthorize("hasRole('ADMIN')")
    public Map<String, Object> adminStats() {
        long totalTasks = tasks.values().stream().mapToLong(List::size).sum();
        return Map.of(
            "totalUsers", tasks.size(),
            "totalTasks", totalTasks
        );
    }

    @GetMapping("/api/admin/tasks")
    @PreAuthorize("hasRole('ADMIN')")
    public Map<String, Object> adminListTasks() {
        List<Map<String, String>> allTasks = tasks.values().stream()
            .flatMap(List::stream)
            .toList();
        return Map.of("tasks", allTasks, "total", allTasks.size());
    }
}
```

### Application Entry Point

```java
package com.example.taskapi;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class TaskApiApplication {
    public static void main(String[] args) {
        SpringApplication.run(TaskApiApplication.class, args);
    }
}
```

### `pom.xml` Dependencies

```xml
<dependencies>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-web</artifactId>
    </dependency>
    <dependency>
        <groupId>com.rampart</groupId>
        <artifactId>rampart-spring-boot-starter</artifactId>
        <version>0.1.0</version>
    </dependency>
</dependencies>
```

Run with:

```bash
RAMPART_URL=https://auth.example.com RAMPART_CLIENT_ID=task-api \
  mvn spring-boot:run
```

## Using Spring Security Without the Starter

If you prefer to use Spring Security's built-in OAuth2 Resource Server support directly (without the Rampart starter), configure it manually:

### `application.yml`

```yaml
spring:
  security:
    oauth2:
      resourceserver:
        jwt:
          issuer-uri: https://auth.example.com
          jwk-set-uri: https://auth.example.com/.well-known/jwks.json
          audiences: my-api
```

### Custom JWT Converter

```java
@Configuration
@EnableWebSecurity
@EnableMethodSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain securityFilterChain(HttpSecurity http) throws Exception {
        http
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/health").permitAll()
                .requestMatchers("/api/**").authenticated()
            )
            .oauth2ResourceServer(oauth2 -> oauth2
                .jwt(jwt -> jwt.jwtAuthenticationConverter(rampartJwtConverter()))
            );

        return http.build();
    }

    @Bean
    public JwtAuthenticationConverter rampartJwtConverter() {
        JwtGrantedAuthoritiesConverter authoritiesConverter = new JwtGrantedAuthoritiesConverter();
        authoritiesConverter.setAuthoritiesClaimName("roles");
        authoritiesConverter.setAuthorityPrefix("ROLE_");

        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(authoritiesConverter);
        return converter;
    }
}
```

This approach uses only standard Spring dependencies and works with any OIDC-compliant provider, including Rampart.

## Testing

### Mock the RampartUser in Tests

```java
@WebMvcTest(TaskController.class)
class TaskControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @Test
    @WithMockRampartUser(id = "user-1", email = "test@example.com", roles = {"USER"})
    void shouldReturnProfile() throws Exception {
        mockMvc.perform(get("/api/profile"))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.userId").value("user-1"))
            .andExpect(jsonPath("$.email").value("test@example.com"));
    }

    @Test
    @WithMockRampartUser(id = "user-1", roles = {"USER"})
    void shouldDenyAdminEndpoint() throws Exception {
        mockMvc.perform(get("/api/admin/stats"))
            .andExpect(status().isForbidden());
    }

    @Test
    @WithMockRampartUser(id = "admin-1", roles = {"ADMIN"})
    void shouldAllowAdminEndpoint() throws Exception {
        mockMvc.perform(get("/api/admin/stats"))
            .andExpect(status().isOk());
    }

    @Test
    void shouldRejectUnauthenticated() throws Exception {
        mockMvc.perform(get("/api/profile"))
            .andExpect(status().isUnauthorized());
    }
}
```

The `@WithMockRampartUser` annotation is provided by the starter for test support. Add the test dependency:

```xml
<dependency>
    <groupId>com.rampart</groupId>
    <artifactId>rampart-spring-boot-starter-test</artifactId>
    <version>0.1.0</version>
    <scope>test</scope>
</dependency>
```
