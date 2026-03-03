package handler

import (
	"encoding/json"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/middleware"
)

// MeResponse is the JSON response for GET /me.
type MeResponse struct {
	ID                string `json:"id"`
	OrgID             string `json:"org_id"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
}

// Me handles GET /me — returns the authenticated user's identity from the JWT.
func Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, "Not authenticated.")
		return
	}

	resp := MeResponse{
		ID:                user.UserID.String(),
		OrgID:             user.OrgID.String(),
		PreferredUsername: user.PreferredUsername,
		Email:             user.Email,
		EmailVerified:     user.EmailVerified,
		GivenName:         user.GivenName,
		FamilyName:        user.FamilyName,
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Log is best-effort; response is already committed.
		_ = err
	}
}
