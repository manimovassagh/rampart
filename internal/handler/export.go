package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

// ExportImportStore defines the database operations required by export/import handlers.
type ExportImportStore interface {
	store.ExportImportStore
}

// ExportHandler returns an http.HandlerFunc that exports an organization's configuration as JSON.
// GET /api/v1/admin/organizations/{id}/export
func ExportHandler(s ExportImportStore, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := parseUUIDParam(w, r)
		if !ok {
			return
		}

		export, err := s.ExportOrganization(r.Context(), orgID)
		if err != nil {
			logger.Error("failed to export organization", "org_id", orgID, "error", err)
			apierror.InternalError(w)
			return
		}

		writeJSON(w, http.StatusOK, export, logger)
	}
}

// ImportHandler returns an http.HandlerFunc that imports an organization's configuration from JSON.
// POST /api/v1/admin/organizations/{id}/import
func ImportHandler(s ExportImportStore, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := parseUUIDParam(w, r)
		if !ok {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

		var export model.OrgExport
		if err := json.NewDecoder(r.Body).Decode(&export); err != nil {
			apierror.BadRequest(w, msgInvalidJSON)
			return
		}

		if export.Organization.Name == "" || export.Organization.Slug == "" {
			apierror.BadRequest(w, "Organization name and slug are required in the import payload.")
			return
		}

		if err := s.ImportOrganization(r.Context(), &export); err != nil {
			logger.Error("failed to import organization", "error", err)
			apierror.InternalError(w)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
