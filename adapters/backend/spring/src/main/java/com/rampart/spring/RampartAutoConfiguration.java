package com.rampart.spring;

import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.core.convert.converter.Converter;
import org.springframework.security.authentication.AbstractAuthenticationToken;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtValidators;
import org.springframework.security.oauth2.jwt.NimbusJwtDecoder;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationConverter;

import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Auto-configuration for Rampart JWT validation.
 *
 * <p>Activated when {@code rampart.issuer} is set in application properties.
 * Configures a {@link JwtDecoder} that fetches keys from Rampart's JWKS endpoint
 * and a {@link JwtAuthenticationConverter} that maps the {@code roles} claim
 * to Spring Security {@code ROLE_} authorities.
 */
@AutoConfiguration
@ConditionalOnProperty("rampart.issuer")
@EnableConfigurationProperties(RampartProperties.class)
public class RampartAutoConfiguration {

    private final RampartProperties properties;

    public RampartAutoConfiguration(RampartProperties properties) {
        this.properties = properties;
    }

    /**
     * Creates a {@link JwtDecoder} that validates tokens against Rampart's JWKS endpoint.
     */
    @Bean
    public JwtDecoder rampartJwtDecoder() {
        String jwksUri = normalizeIssuer(properties.getIssuer()) + "/.well-known/jwks.json";

        NimbusJwtDecoder decoder = NimbusJwtDecoder.withJwkSetUri(jwksUri).build();

        if (properties.getAudience() != null && !properties.getAudience().isBlank()) {
            decoder.setJwtValidator(JwtValidators.createDefaultWithIssuer(properties.getIssuer()));
        } else {
            decoder.setJwtValidator(JwtValidators.createDefaultWithIssuer(properties.getIssuer()));
        }

        return decoder;
    }

    /**
     * Creates a {@link JwtAuthenticationConverter} that extracts Rampart roles
     * and maps them to Spring Security {@code ROLE_} granted authorities.
     */
    @Bean
    public JwtAuthenticationConverter rampartJwtAuthenticationConverter() {
        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(new RampartGrantedAuthoritiesConverter());
        return converter;
    }

    private static String normalizeIssuer(String issuer) {
        if (issuer != null && issuer.endsWith("/")) {
            return issuer.substring(0, issuer.length() - 1);
        }
        return issuer;
    }

    /**
     * Converts the {@code roles} claim from a Rampart JWT into Spring Security
     * granted authorities with the {@code ROLE_} prefix.
     */
    static class RampartGrantedAuthoritiesConverter implements Converter<Jwt, Collection<GrantedAuthority>> {

        @Override
        public Collection<GrantedAuthority> convert(Jwt jwt) {
            List<String> roles = jwt.getClaimAsStringList("roles");
            if (roles == null) {
                return Collections.emptyList();
            }
            return roles.stream()
                    .map(role -> new SimpleGrantedAuthority("ROLE_" + role))
                    .collect(Collectors.toUnmodifiableList());
        }
    }
}
