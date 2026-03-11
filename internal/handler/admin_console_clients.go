// admin_console_clients.go contains admin console handlers for OAuth client management:
// ListClientsPage, CreateClientPage, CreateClientAction, ClientDetailPage,
// UpdateClientAction, DeleteClientAction, RegenerateSecretAction, generateRandomSecret.
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// ListClientsPage handles GET /admin/clients
func (h *AdminConsoleHandler) ListClientsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	clients, total, err := h.store.ListOAuthClients(ctx, orgID, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list clients", "error", err)
		h.render(w, r, "clients_list", &pageData{Title: "OAuth Clients", ActiveNav: navClients, Error: "Failed to load clients."})
		return
	}

	adminClients := make([]*model.AdminClientResponse, len(clients))
	for i, c := range clients {
		adminClients[i] = c.ToAdminResponse()
	}

	pg := buildPagination(page, limit, total, pathAdminClients, search)

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "clients_list", "clients_table", &pageData{Clients: adminClients, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "clients_list", &pageData{
		Title:      "OAuth Clients",
		ActiveNav:  navClients,
		Clients:    adminClients,
		Search:     search,
		Pagination: pg,
	})
}

// CreateClientPage handles GET /admin/clients/new
func (h *AdminConsoleHandler) CreateClientPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients})
}

// CreateClientAction handles POST /admin/clients
func (h *AdminConsoleHandler) CreateClientAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: msgInvalidForm})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	clientType := r.FormValue("client_type")
	redirectURIsRaw := strings.TrimSpace(r.FormValue("redirect_uris"))

	formValues := map[string]string{
		"name":          name,
		"description":   description,
		"client_type":   clientType,
		"redirect_uris": redirectURIsRaw,
	}

	if name == "" {
		h.render(w, r, tmplClientCreate, &pageData{
			Title: titleCreateClient, ActiveNav: navClients,
			FormErrors: map[string]string{"name": "Client name is required."},
			FormValues: formValues,
		})
		return
	}

	if clientType != "public" && clientType != clientTypeConfidential {
		clientType = "public"
	}

	var uris []string
	for _, line := range strings.Split(redirectURIsRaw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			uris = append(uris, trimmed)
		}
	}

	client := &model.OAuthClient{
		OrgID:        orgID,
		Name:         name,
		Description:  description,
		ClientType:   clientType,
		RedirectURIs: uris,
		Enabled:      true,
	}

	var clientSecret string
	if clientType == clientTypeConfidential {
		secret, err := generateRandomSecret()
		if err != nil {
			h.logger.Error("failed to generate client secret", "error", err)
			h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: msgInternalErr})
			return
		}
		clientSecret = secret
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			h.logger.Error("failed to hash client secret", "error", err)
			h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: msgInternalErr})
			return
		}
		client.ClientSecretHash = hash
	}

	created, err := h.store.CreateOAuthClient(ctx, client)
	if err != nil {
		h.logger.Error("failed to create client", "error", err)
		h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: "Failed to create client."})
		return
	}

	h.auditLog(r, orgID, model.EventClientCreated, "client", created.ID, created.Name)

	if clientSecret != "" {
		h.render(w, r, "client_detail", &pageData{
			Title:        fmt.Sprintf("Client: %s", created.Name),
			ActiveNav:    navClients,
			ClientDetail: created.ToAdminResponse(),
			ClientSecret: clientSecret,
			Flash:        "Client created. Copy the secret now — it won't be shown again.",
		})
		return
	}

	middleware.SetFlash(w, "Client created successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, created.ID), http.StatusFound)
}

// ClientDetailPage handles GET /admin/clients/{id}
func (h *AdminConsoleHandler) ClientDetailPage(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)

	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil || client == nil || client.OrgID != authUser.OrgID {
		middleware.SetFlash(w, "Client not found.")
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	h.render(w, r, "client_detail", &pageData{
		Title:        fmt.Sprintf("Client: %s", client.Name),
		ActiveNav:    navClients,
		ClientDetail: client.ToAdminResponse(),
	})
}

// UpdateClientAction handles POST /admin/clients/{id}
func (h *AdminConsoleHandler) UpdateClientAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	req := &model.UpdateClientRequest{
		Name:         strings.TrimSpace(r.FormValue("name")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		RedirectURIs: strings.TrimSpace(r.FormValue("redirect_uris")),
		Enabled:      r.FormValue("enabled") == formValueTrue,
	}

	updateAuthUser := middleware.GetAuthenticatedUser(r.Context())
	if _, err := h.store.UpdateOAuthClient(r.Context(), clientID, updateAuthUser.OrgID, req); err != nil {
		h.logger.Error("failed to update client", "error", err)
		middleware.SetFlash(w, "Failed to update client.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Client updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
}

// DeleteClientAction handles POST /admin/clients/{id}/delete
func (h *AdminConsoleHandler) DeleteClientAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	deleteAuthUser := middleware.GetAuthenticatedUser(r.Context())
	if err := h.store.DeleteOAuthClient(r.Context(), clientID, deleteAuthUser.OrgID); err != nil {
		h.logger.Error("failed to delete client", "error", err)
		middleware.SetFlash(w, "Failed to delete client.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Client deleted.")
	http.Redirect(w, r, pathAdminClients, http.StatusFound)
}

// RegenerateSecretAction handles POST /admin/clients/{id}/regenerate-secret
func (h *AdminConsoleHandler) RegenerateSecretAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	ctx := r.Context()
	regenAuthUser := middleware.GetAuthenticatedUser(ctx)

	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil || client == nil || client.OrgID != regenAuthUser.OrgID {
		middleware.SetFlash(w, "Client not found.")
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	if client.ClientType != clientTypeConfidential {
		middleware.SetFlash(w, "Only confidential clients have secrets.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	secret, err := generateRandomSecret()
	if err != nil {
		h.logger.Error("failed to generate secret", "error", err)
		middleware.SetFlash(w, msgRegenFailed)
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash secret", "error", err)
		middleware.SetFlash(w, msgRegenFailed)
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	if err := h.store.UpdateClientSecret(ctx, clientID, regenAuthUser.OrgID, hash); err != nil {
		h.logger.Error("failed to update client secret", "error", err)
		middleware.SetFlash(w, msgRegenFailed)
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	h.render(w, r, "client_detail", &pageData{
		Title:        fmt.Sprintf("Client: %s", client.Name),
		ActiveNav:    navClients,
		ClientDetail: client.ToAdminResponse(),
		ClientSecret: secret,
		Flash:        "Secret regenerated. Copy it now — it won't be shown again.",
	})
}

func generateRandomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}
