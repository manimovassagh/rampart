package com.rampart.spring;

import org.junit.jupiter.api.Test;
import org.springframework.security.oauth2.jwt.Jwt;

import java.time.Instant;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class RampartClaimsTest {

    @Test
    void fromJwtExtractsAllClaims() {
        Jwt jwt = buildJwt(Map.of(
                "sub", "user-123",
                "org_id", "org-456",
                "preferred_username", "jdoe",
                "email", "jdoe@example.com",
                "email_verified", true,
                "given_name", "John",
                "family_name", "Doe",
                "roles", List.of("admin", "user")
        ));

        RampartClaims claims = RampartClaims.fromJwt(jwt);

        assertEquals("user-123", claims.getSub());
        assertEquals("org-456", claims.getOrgId());
        assertEquals("jdoe", claims.getPreferredUsername());
        assertEquals("jdoe@example.com", claims.getEmail());
        assertTrue(claims.isEmailVerified());
        assertEquals("John", claims.getGivenName());
        assertEquals("Doe", claims.getFamilyName());
        assertEquals(List.of("admin", "user"), claims.getRoles());
    }

    @Test
    void fromJwtHandlesMissingOptionalClaims() {
        Jwt jwt = buildJwt(Map.of("sub", "user-789"));

        RampartClaims claims = RampartClaims.fromJwt(jwt);

        assertEquals("user-789", claims.getSub());
        assertNull(claims.getOrgId());
        assertNull(claims.getPreferredUsername());
        assertNull(claims.getEmail());
        assertFalse(claims.isEmailVerified());
        assertNull(claims.getGivenName());
        assertNull(claims.getFamilyName());
        assertTrue(claims.getRoles().isEmpty());
    }

    @Test
    void fromJwtRejectsNull() {
        assertThrows(NullPointerException.class, () -> RampartClaims.fromJwt(null));
    }

    @Test
    void rolesListIsUnmodifiable() {
        Jwt jwt = buildJwt(Map.of(
                "sub", "user-1",
                "roles", List.of("editor")
        ));

        RampartClaims claims = RampartClaims.fromJwt(jwt);

        assertThrows(UnsupportedOperationException.class, () -> claims.getRoles().add("hacker"));
    }

    @Test
    void equalsAndHashCode() {
        Jwt jwt = buildJwt(Map.of(
                "sub", "user-1",
                "email", "a@b.com",
                "roles", List.of("admin")
        ));

        RampartClaims a = RampartClaims.fromJwt(jwt);
        RampartClaims b = RampartClaims.fromJwt(jwt);

        assertEquals(a, b);
        assertEquals(a.hashCode(), b.hashCode());
    }

    @Test
    void toStringContainsKeyFields() {
        Jwt jwt = buildJwt(Map.of(
                "sub", "user-1",
                "email", "a@b.com",
                "roles", List.of("admin")
        ));

        RampartClaims claims = RampartClaims.fromJwt(jwt);
        String str = claims.toString();

        assertTrue(str.contains("user-1"));
        assertTrue(str.contains("a@b.com"));
        assertTrue(str.contains("admin"));
    }

    private static Jwt buildJwt(Map<String, Object> claims) {
        Jwt.Builder builder = Jwt.withTokenValue("token")
                .header("alg", "RS256")
                .issuedAt(Instant.now())
                .expiresAt(Instant.now().plusSeconds(3600));

        claims.forEach(builder::claim);

        // Ensure sub is set as the subject
        if (claims.containsKey("sub")) {
            builder.subject((String) claims.get("sub"));
        }

        return builder.build();
    }
}
