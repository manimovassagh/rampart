package com.rampart.sample;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

/**
 * API endpoints matching the Express sample backend exactly.
 *
 * All protected endpoints receive the validated JWT via @AuthenticationPrincipal.
 * Claims are extracted following the same pattern as RampartClaims in the starter.
 */
@RestController
@RequestMapping("/api")
public class ApiController {

    @Value("${rampart.issuer}")
    private String issuer;

    /**
     * Public health check.
     */
    @GetMapping("/health")
    public Map<String, Object> health() {
        Map<String, Object> response = new LinkedHashMap<>();
        response.put("status", "ok");
        response.put("issuer", issuer);
        return response;
    }

    /**
     * Protected: returns user profile extracted from JWT claims.
     */
    @GetMapping("/profile")
    public Map<String, Object> profile(@AuthenticationPrincipal Jwt jwt) {
        Map<String, Object> user = new LinkedHashMap<>();
        user.put("id", jwt.getSubject());
        user.put("email", claimOrEmpty(jwt, "email"));
        user.put("username", claimOrEmpty(jwt, "preferred_username"));
        user.put("org_id", claimOrEmpty(jwt, "org_id"));
        user.put("email_verified", booleanClaim(jwt, "email_verified"));
        user.put("given_name", claimOrEmpty(jwt, "given_name"));
        user.put("family_name", claimOrEmpty(jwt, "family_name"));
        user.put("roles", rolesList(jwt));

        Map<String, Object> response = new LinkedHashMap<>();
        response.put("message", "Authenticated!");
        response.put("user", user);
        return response;
    }

    /**
     * Protected: returns all raw JWT claims.
     */
    @GetMapping("/claims")
    public Map<String, Object> claims(@AuthenticationPrincipal Jwt jwt) {
        return jwt.getClaims();
    }

    /**
     * Protected: requires "editor" role.
     */
    @GetMapping("/editor/dashboard")
    public Map<String, Object> editorDashboard(@AuthenticationPrincipal Jwt jwt) {
        Map<String, Object> data = new LinkedHashMap<>();
        data.put("drafts", 3);
        data.put("published", 12);
        data.put("pending_review", 2);

        Map<String, Object> response = new LinkedHashMap<>();
        response.put("message", "Welcome, Editor!");
        response.put("user", claimOrEmpty(jwt, "preferred_username"));
        response.put("roles", rolesList(jwt));
        response.put("data", data);
        return response;
    }

    /**
     * Protected: requires "manager" role.
     */
    @GetMapping("/manager/reports")
    public Map<String, Object> managerReports(@AuthenticationPrincipal Jwt jwt) {
        Map<String, Object> report1 = new LinkedHashMap<>();
        report1.put("name", "Q1 Revenue");
        report1.put("status", "complete");

        Map<String, Object> report2 = new LinkedHashMap<>();
        report2.put("name", "User Growth");
        report2.put("status", "in_progress");

        Map<String, Object> response = new LinkedHashMap<>();
        response.put("message", "Manager Reports");
        response.put("user", claimOrEmpty(jwt, "preferred_username"));
        response.put("roles", rolesList(jwt));
        response.put("reports", List.of(report1, report2));
        return response;
    }

    // --- Helpers matching RampartClaims extraction ---

    private static String claimOrEmpty(Jwt jwt, String claim) {
        String value = jwt.getClaimAsString(claim);
        return value != null ? value : "";
    }

    private static boolean booleanClaim(Jwt jwt, String claim) {
        Boolean value = jwt.getClaim(claim);
        return value != null && value;
    }

    private static List<String> rolesList(Jwt jwt) {
        List<String> roles = jwt.getClaimAsStringList("roles");
        return roles != null ? roles : Collections.emptyList();
    }
}
