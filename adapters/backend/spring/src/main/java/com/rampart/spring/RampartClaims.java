package com.rampart.spring;

import org.springframework.security.oauth2.jwt.Jwt;

import java.util.Collections;
import java.util.List;
import java.util.Objects;

/**
 * Typed representation of Rampart JWT claims.
 *
 * <p>Use {@link #fromJwt(Jwt)} to extract claims from a validated JWT token.
 */
public final class RampartClaims {

    private final String sub;
    private final String orgId;
    private final String preferredUsername;
    private final String email;
    private final boolean emailVerified;
    private final String givenName;
    private final String familyName;
    private final List<String> roles;

    private RampartClaims(String sub,
                          String orgId,
                          String preferredUsername,
                          String email,
                          boolean emailVerified,
                          String givenName,
                          String familyName,
                          List<String> roles) {
        this.sub = sub;
        this.orgId = orgId;
        this.preferredUsername = preferredUsername;
        this.email = email;
        this.emailVerified = emailVerified;
        this.givenName = givenName;
        this.familyName = familyName;
        this.roles = roles;
    }

    /**
     * Extract Rampart-specific claims from a validated JWT.
     *
     * @param jwt the validated JWT token
     * @return typed claims object
     */
    @SuppressWarnings("unchecked")
    public static RampartClaims fromJwt(Jwt jwt) {
        Objects.requireNonNull(jwt, "jwt must not be null");

        List<String> roles = jwt.getClaimAsStringList("roles");
        if (roles == null) {
            roles = Collections.emptyList();
        }

        Boolean verified = jwt.getClaim("email_verified");

        return new RampartClaims(
                jwt.getSubject(),
                jwt.getClaimAsString("org_id"),
                jwt.getClaimAsString("preferred_username"),
                jwt.getClaimAsString("email"),
                verified != null && verified,
                jwt.getClaimAsString("given_name"),
                jwt.getClaimAsString("family_name"),
                Collections.unmodifiableList(roles)
        );
    }

    public String getSub() {
        return sub;
    }

    public String getOrgId() {
        return orgId;
    }

    public String getPreferredUsername() {
        return preferredUsername;
    }

    public String getEmail() {
        return email;
    }

    public boolean isEmailVerified() {
        return emailVerified;
    }

    public String getGivenName() {
        return givenName;
    }

    public String getFamilyName() {
        return familyName;
    }

    public List<String> getRoles() {
        return roles;
    }

    @Override
    public String toString() {
        return "RampartClaims{sub='" + sub + "', email='" + email + "', roles=" + roles + "}";
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        RampartClaims that = (RampartClaims) o;
        return emailVerified == that.emailVerified
                && Objects.equals(sub, that.sub)
                && Objects.equals(orgId, that.orgId)
                && Objects.equals(preferredUsername, that.preferredUsername)
                && Objects.equals(email, that.email)
                && Objects.equals(givenName, that.givenName)
                && Objects.equals(familyName, that.familyName)
                && Objects.equals(roles, that.roles);
    }

    @Override
    public int hashCode() {
        return Objects.hash(sub, orgId, preferredUsername, email, emailVerified, givenName, familyName, roles);
    }
}
