package handler

import (
	"context"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/oauth"
)

//go:embed templates/login.html
var loginTemplateFS embed.FS

var loginTmpl = template.Must(template.ParseFS(loginTemplateFS, "templates/login.html"))

const authCodeTTL = 10 * time.Minute

// AuthorizeStore defines the database operations required by AuthorizeHandler.
type AuthorizeStore interface {
	GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error)
	GetDefaultOrganizationID(ctx context.Context) (uuid.UUID, error)
	GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error)
	StoreAuthorizationCode(ctx context.Context, code string, clientID string, userID, orgID uuid.UUID, redirectURI, codeChallenge, scope string, expiresAt time.Time) error
	UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error
}

// AuthorizeHandler handles the OAuth 2.0 authorization endpoint.
type AuthorizeHandler struct {
	store  AuthorizeStore
	logger *slog.Logger
	audit  *audit.Logger
}

// NewAuthorizeHandler creates a new authorization endpoint handler.
func NewAuthorizeHandler(store AuthorizeStore, logger *slog.Logger, auditLogger *audit.Logger) *AuthorizeHandler {
	return &AuthorizeHandler{store: store, logger: logger, audit: auditLogger}
}

type loginPageData struct {
	ClientID      string
	ClientName    string
	RedirectURI   string
	Scope         string
	State         string
	CodeChallenge string
	Error         string
}

// Authorize handles both GET (render login) and POST (authenticate + redirect).
func (h *AuthorizeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	}
}

func (h *AuthorizeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")
	scope := q.Get("scope")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")

	// Validate required params — show error page, do NOT redirect (RFC 6749 §4.1.2.1)
	if clientID == "" || redirectURI == "" {
		h.renderError(w, http.StatusBadRequest, "Missing required parameters: client_id and redirect_uri.")
		return
	}

	client, err := h.store.GetOAuthClient(r.Context(), clientID)
	if err != nil {
		h.logger.Error("failed to fetch oauth client", "error", err)
		h.renderError(w, http.StatusInternalServerError, msgUnexpectedErr)
		return
	}
	if client == nil {
		h.renderError(w, http.StatusBadRequest, "Unknown client_id.")
		return
	}

	if !database.ValidateRedirectURI(client, redirectURI) {
		h.renderError(w, http.StatusBadRequest, "Invalid redirect_uri.")
		return
	}

	if responseType != "code" {
		h.renderError(w, http.StatusBadRequest, "Unsupported response_type. Only 'code' is supported.")
		return
	}

	if state == "" {
		h.renderError(w, http.StatusBadRequest, "Missing required parameter: state.")
		return
	}

	if codeChallenge == "" || codeChallengeMethod != "S256" {
		h.renderError(w, http.StatusBadRequest, "PKCE is required. Provide code_challenge with method S256.")
		return
	}

	if scope == "" {
		scope = "openid"
	}

	h.renderLoginPage(w, &loginPageData{
		ClientID:      clientID,
		ClientName:    client.Name,
		RedirectURI:   redirectURI,
		Scope:         scope,
		State:         state,
		CodeChallenge: codeChallenge,
	})
}

func (h *AuthorizeHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed < minResponseDuration {
			time.Sleep(minResponseDuration - elapsed)
		}
	}()

	if err := r.ParseForm(); err != nil {
		apierror.BadRequest(w, "Invalid form data.")
		return
	}

	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	scope := r.FormValue("scope")
	state := r.FormValue("state")
	codeChallenge := r.FormValue("code_challenge")
	identifier := strings.TrimSpace(r.FormValue("identifier"))
	password := r.FormValue("password")

	// Re-validate client and redirect_uri (don't trust hidden form fields blindly)
	if clientID == "" || redirectURI == "" {
		h.renderError(w, http.StatusBadRequest, "Missing required parameters.")
		return
	}

	ctx := r.Context()

	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil {
		h.logger.Error("failed to fetch oauth client", "error", err)
		h.renderError(w, http.StatusInternalServerError, msgUnexpectedErr)
		return
	}
	if client == nil {
		h.renderError(w, http.StatusBadRequest, "Unknown client_id.")
		return
	}

	if !database.ValidateRedirectURI(client, redirectURI) {
		h.renderError(w, http.StatusBadRequest, "Invalid redirect_uri.")
		return
	}

	pageData := loginPageData{
		ClientID:      clientID,
		ClientName:    client.Name,
		RedirectURI:   redirectURI,
		Scope:         scope,
		State:         state,
		CodeChallenge: codeChallenge,
	}

	if identifier == "" || password == "" {
		pageData.Error = "Username/email and password are required."
		h.renderLoginPage(w, &pageData)
		return
	}

	// Resolve org (use client's org)
	orgID := client.OrgID

	// Try email first, then username
	user, err := h.store.GetUserByEmail(ctx, strings.ToLower(identifier), orgID)
	if err != nil {
		h.logger.Error("failed to lookup user by email", "error", err)
		h.renderError(w, http.StatusInternalServerError, msgUnexpectedErr)
		return
	}
	if user == nil {
		user, err = h.store.GetUserByUsername(ctx, identifier, orgID)
		if err != nil {
			h.logger.Error("failed to lookup user by username", "error", err)
			h.renderError(w, http.StatusInternalServerError, msgUnexpectedErr)
			return
		}
	}

	if user == nil {
		h.audit.Log(ctx, r, orgID, model.EventUserLoginFailed, nil, identifier, "user", "", identifier, map[string]any{"reason": "user_not_found", "client_id": clientID})
		pageData.Error = msgInvalidLogin
		h.renderLoginPage(w, &pageData)
		return
	}

	if !user.Enabled {
		h.audit.Log(ctx, r, orgID, model.EventUserLoginFailed, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"reason": "account_disabled", "client_id": clientID})
		pageData.Error = msgInvalidLogin
		h.renderLoginPage(w, &pageData)
		return
	}

	ok, err := auth.VerifyPassword(password, string(user.PasswordHash))
	if err != nil {
		h.logger.Error("failed to verify password", "error", err)
		h.renderError(w, http.StatusInternalServerError, msgUnexpectedErr)
		return
	}
	if !ok {
		h.audit.Log(ctx, r, orgID, model.EventUserLoginFailed, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"reason": "invalid_password", "client_id": clientID})
		pageData.Error = msgInvalidLogin
		h.renderLoginPage(w, &pageData)
		return
	}

	// Generate authorization code
	code, err := oauth.GenerateAuthorizationCode()
	if err != nil {
		h.logger.Error("failed to generate authorization code", "error", err)
		h.renderError(w, http.StatusInternalServerError, msgUnexpectedErr)
		return
	}

	expiresAt := time.Now().Add(authCodeTTL)
	if err := h.store.StoreAuthorizationCode(ctx, code, clientID, user.ID, orgID, redirectURI, codeChallenge, scope, expiresAt); err != nil {
		h.logger.Error("failed to store authorization code", "error", err)
		h.renderError(w, http.StatusInternalServerError, msgUnexpectedErr)
		return
	}

	if err := h.store.UpdateLastLoginAt(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last_login_at", "error", err, "user_id", user.ID)
	}

	h.audit.Log(ctx, r, orgID, model.EventUserLogin, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"client_id": clientID})

	// Redirect back to the client with the authorization code
	redirectURL := redirectURI + "?code=" + code + "&state=" + state
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *AuthorizeHandler) renderLoginPage(w http.ResponseWriter, data *loginPageData) {
	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", cacheNoStore)
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(http.StatusOK)
	if err := loginTmpl.Execute(w, data); err != nil {
		h.logger.Error("failed to render login template", "error", err)
	}
}

func (h *AuthorizeHandler) renderError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", cacheNoStore)
	w.WriteHeader(status)
	data := struct{ Error string }{Error: message}
	tmpl := template.Must(template.New("error").Parse(`<!DOCTYPE html>
<html><head><title>Error — Rampart</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;background:#f5f5f5}
.err{background:#fff;padding:2rem;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,.08);max-width:400px;text-align:center}
h1{color:#dc2626;font-size:1.25rem;margin-bottom:.5rem}p{color:#666;font-size:.9rem}</style></head>
<body><div class="err"><h1>Authorization Error</h1><p>{{.Error}}</p></div></body></html>`))
	if err := tmpl.Execute(w, data); err != nil {
		h.logger.Error("failed to render error page", "error", err)
	}
}
