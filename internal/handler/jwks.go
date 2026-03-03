package handler

import (
	"log/slog"
	"net/http"

	"github.com/manimovassagh/rampart/internal/signing"
)

// JWKSHandler returns the JWKS JSON containing the RSA public key.
func JWKSHandler(kp *signing.KeyPair, logger *slog.Logger) http.HandlerFunc {
	data, err := kp.JWKSResponse()
	if err != nil {
		logger.Error("failed to marshal JWKS response", "error", err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(data); err != nil {
			logger.Error("failed to write JWKS response", "error", err)
		}
	}
}
