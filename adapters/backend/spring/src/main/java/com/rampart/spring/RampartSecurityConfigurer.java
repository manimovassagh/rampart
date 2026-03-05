package com.rampart.spring;

import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationConverter;

/**
 * Security configurer that applies Rampart defaults to Spring Security's
 * {@link HttpSecurity} configuration.
 *
 * <p>Usage:
 * <pre>
 * {@literal @}Bean
 * public SecurityFilterChain filterChain(HttpSecurity http,
 *         JwtAuthenticationConverter converter) throws Exception {
 *     RampartSecurityConfigurer.applyDefaults(http, converter);
 *     http.authorizeHttpRequests(auth -> auth
 *         .requestMatchers("/admin/**").hasRole("admin")
 *         .anyRequest().authenticated()
 *     );
 *     return http.build();
 * }
 * </pre>
 */
public final class RampartSecurityConfigurer {

    private RampartSecurityConfigurer() {
        // utility class
    }

    /**
     * Applies Rampart security defaults to the given {@link HttpSecurity}:
     * <ul>
     *   <li>Disables CSRF (stateless API)</li>
     *   <li>Sets session management to stateless</li>
     *   <li>Configures OAuth2 resource server with JWT and the Rampart
     *       authentication converter (which maps {@code roles} to {@code ROLE_} authorities)</li>
     * </ul>
     *
     * @param http      the HttpSecurity to configure
     * @param converter the Rampart JWT authentication converter
     * @return the configured HttpSecurity for further chaining
     * @throws Exception if configuration fails
     */
    public static HttpSecurity applyDefaults(HttpSecurity http,
                                             JwtAuthenticationConverter converter) throws Exception {
        http
                .csrf(AbstractHttpConfigurer::disable)
                .sessionManagement(session ->
                        session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
                .oauth2ResourceServer(oauth2 ->
                        oauth2.jwt(jwt -> jwt.jwtAuthenticationConverter(converter)));
        return http;
    }
}
