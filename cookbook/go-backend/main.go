package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	rampart "github.com/manimovassagh/rampart/adapters/backend/go"
)

func main() {
	issuer := os.Getenv("RAMPART_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:8080"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	auth := rampart.NewMiddleware(rampart.Config{Issuer: issuer})

	mux := http.NewServeMux()

	// Public route — no auth required
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"issuer": issuer,
		})
	})

	// Protected route — requires valid Rampart JWT
	mux.Handle("GET /api/profile", auth(http.HandlerFunc(profileHandler)))

	// Protected route — returns all raw claims
	mux.Handle("GET /api/claims", auth(http.HandlerFunc(claimsHandler)))

	// Role-protected route — requires "editor" role
	mux.Handle("GET /api/editor/dashboard",
		auth(rampart.RequireRoles("editor")(http.HandlerFunc(editorDashboardHandler))))

	// Role-protected route — requires "manager" role
	mux.Handle("GET /api/manager/reports",
		auth(rampart.RequireRoles("manager")(http.HandlerFunc(managerReportsHandler))))

	// WARNING: Restrict to your frontend domain in production. Never use "*" in production.
	handler := corsMiddleware(mux)

	fmt.Printf("Sample backend running on http://localhost:%s\n", port)
	fmt.Printf("Rampart issuer: %s\n", issuer)
	fmt.Println("\nRoutes:")
	fmt.Println("  GET /api/health            — public")
	fmt.Println("  GET /api/profile           — protected (any authenticated user)")
	fmt.Println("  GET /api/claims            — protected (any authenticated user)")
	fmt.Println(`  GET /api/editor/dashboard  — protected (requires "editor" role)`)
	fmt.Println(`  GET /api/manager/reports   — protected (requires "manager" role)`)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := rampart.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	roles := claims.Roles
	if roles == nil {
		roles = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Authenticated!",
		"user": map[string]any{
			"id":             claims.Sub,
			"email":          claims.Email,
			"username":       claims.PreferredUsername,
			"org_id":         claims.OrgID,
			"email_verified": claims.EmailVerified,
			"given_name":     claims.GivenName,
			"family_name":    claims.FamilyName,
			"roles":          roles,
		},
	})
}

func claimsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := rampart.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	roles := claims.Roles
	if roles == nil {
		roles = []string{}
	}

	// Build explicit map to avoid omitempty hiding empty fields
	writeJSON(w, http.StatusOK, map[string]any{
		"iss":                claims.Iss,
		"sub":                claims.Sub,
		"iat":                claims.Iat,
		"exp":                claims.Exp,
		"org_id":             claims.OrgID,
		"preferred_username": claims.PreferredUsername,
		"email":              claims.Email,
		"email_verified":     claims.EmailVerified,
		"roles":              roles,
	})
}

func editorDashboardHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := rampart.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Welcome, Editor!",
		"user":    claims.PreferredUsername,
		"roles":   claims.Roles,
		"data": map[string]any{
			"drafts":         3,
			"published":      12,
			"pending_review": 2,
		},
	})
}

func managerReportsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := rampart.ClaimsFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Manager Reports",
		"user":    claims.PreferredUsername,
		"roles":   claims.Roles,
		"reports": []map[string]any{
			{"name": "Q1 Revenue", "status": "complete"},
			{"name": "User Growth", "status": "in_progress"},
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WARNING: Restrict to your frontend domain in production. Never use "*" in production.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
