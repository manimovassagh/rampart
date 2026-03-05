package com.rampart.spring;

import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Configuration properties for Rampart IAM integration.
 *
 * <p>Example {@code application.yml}:
 * <pre>
 * rampart:
 *   issuer: https://auth.example.com
 *   audience: my-api
 * </pre>
 */
@ConfigurationProperties(prefix = "rampart")
public class RampartProperties {

    /**
     * The Rampart issuer URL (e.g. {@code https://auth.example.com}).
     * Used to construct the JWKS endpoint at {@code {issuer}/.well-known/jwks.json}.
     */
    private String issuer;

    /**
     * Expected audience claim in JWTs. Optional — when set, tokens without
     * a matching {@code aud} claim are rejected.
     */
    private String audience;

    public String getIssuer() {
        return issuer;
    }

    public void setIssuer(String issuer) {
        this.issuer = issuer;
    }

    public String getAudience() {
        return audience;
    }

    public void setAudience(String audience) {
        this.audience = audience;
    }
}
