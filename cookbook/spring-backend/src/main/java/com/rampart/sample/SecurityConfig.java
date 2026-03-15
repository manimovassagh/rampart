package com.rampart.sample;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.convert.converter.Converter;
import org.springframework.security.authentication.AbstractAuthenticationToken;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationConverter;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.access.AccessDeniedHandler;
import org.springframework.security.web.AuthenticationEntryPoint;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.CorsConfigurationSource;
import org.springframework.web.cors.UrlBasedCorsConfigurationSource;

import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Security configuration that mirrors the Rampart Spring Boot starter approach:
 * - Stateless sessions (no CSRF)
 * - JWT validation via OAuth2 Resource Server
 * - Custom authorities converter mapping "roles" claim to ROLE_* authorities
 * - CORS open for development
 */
@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
                .csrf(AbstractHttpConfigurer::disable)
                .cors(cors -> cors.configurationSource(corsConfigurationSource()))
                .sessionManagement(session ->
                        session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
                .authorizeHttpRequests(auth -> auth
                        .requestMatchers("/api/health").permitAll()
                        .requestMatchers("/api/editor/**").hasRole("editor")
                        .requestMatchers("/api/manager/**").hasRole("manager")
                        .anyRequest().authenticated()
                )
                .oauth2ResourceServer(oauth2 ->
                        oauth2.jwt(jwt -> jwt.jwtAuthenticationConverter(rampartJwtAuthenticationConverter()))
                                .authenticationEntryPoint(rampartAuthenticationEntryPoint())
                                .accessDeniedHandler(rampartAccessDeniedHandler()));

        return http.build();
    }

    /**
     * Maps the "roles" claim from Rampart JWTs to Spring Security ROLE_* authorities.
     * This is the same approach used by RampartAutoConfiguration in the starter.
     */
    @Bean
    public JwtAuthenticationConverter rampartJwtAuthenticationConverter() {
        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(jwt -> {
            List<String> roles = jwt.getClaimAsStringList("roles");
            if (roles == null) {
                return Collections.emptyList();
            }
            return roles.stream()
                    .map(role -> (GrantedAuthority) new SimpleGrantedAuthority("ROLE_" + role))
                    .collect(Collectors.toUnmodifiableList());
        });
        return converter;
    }

    /**
     * Custom 401 handler matching the Express error format.
     */
    @Bean
    public AuthenticationEntryPoint rampartAuthenticationEntryPoint() {
        return (request, response, authException) -> {
            response.setStatus(401);
            response.setContentType("application/json");
            response.getWriter().write(
                    "{\"error\":\"unauthorized\",\"error_description\":\"Missing or invalid authorization header.\",\"status\":401}");
        };
    }

    /**
     * Custom 403 handler matching the Express error format.
     */
    @Bean
    public AccessDeniedHandler rampartAccessDeniedHandler() {
        return (request, response, accessDeniedException) -> {
            response.setStatus(403);
            response.setContentType("application/json");
            String uri = request.getRequestURI();
            String role = uri.contains("/editor/") ? "editor" : uri.contains("/manager/") ? "manager" : "unknown";
            response.getWriter().write(
                    "{\"error\":\"forbidden\",\"error_description\":\"Missing required role(s): " + role + "\",\"status\":403}");
        };
    }

    /**
     * CORS configuration: allow all origins for local development.
     * WARNING: Restrict to your frontend domain in production. Never use "*" in production.
     */
    @Bean
    public CorsConfigurationSource corsConfigurationSource() {
        CorsConfiguration config = new CorsConfiguration();
        config.addAllowedOrigin("*");
        config.addAllowedMethod("*");
        config.addAllowedHeader("*");

        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/**", config);
        return source;
    }
}
