package handler

import (
	"encoding/json"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

// MeStore defines the database operations required by the /me handler.
type MeStore interface {
	store.SocialAccountStore
}

// MeResponse is the JSON response for GET /me.
type MeResponse struct {
	ID                string                        `json:"id"`
	OrgID             string                        `json:"org_id"`
	PreferredUsername string                        `json:"preferred_username"`
	Email             string                        `json:"email"`
	EmailVerified     bool                          `json:"email_verified"`
	GivenName         string                        `json:"given_name,omitempty"`
	FamilyName        string                        `json:"family_name,omitempty"`
	SocialAccounts    []model.SocialAccountResponse `json:"social_accounts,omitempty"`
}

// MeHandler handles the /me endpoint.
type MeHandler struct {
	store MeStore
}

// NewMeHandler creates a new MeHandler.
func NewMeHandler(s MeStore) *MeHandler {
	return &MeHandler{store: s}
}

// Me handles GET /me — returns the authenticated user's identity from the JWT.
func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
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

	accounts, err := h.store.GetSocialAccountsByUserID(r.Context(), user.UserID)
	if err == nil && len(accounts) > 0 {
		resp.SocialAccounts = make([]model.SocialAccountResponse, len(accounts))
		for i, a := range accounts {
			resp.SocialAccounts[i] = *a.ToResponse()
		}
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Log is best-effort; response is already committed.
		_ = err
	}
}
