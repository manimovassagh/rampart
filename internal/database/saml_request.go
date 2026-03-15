package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StoreSAMLRequest stores an issued AuthnRequest ID so we can validate InResponseTo later.
func (db *DB) StoreSAMLRequest(ctx context.Context, requestID string, providerID uuid.UUID, expiresAt time.Time) error {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	_, err := db.Pool.Exec(ctx,
		`INSERT INTO saml_requests (request_id, provider_id, expires_at) VALUES ($1, $2, $3)`,
		requestID, providerID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("storing SAML request: %w", err)
	}
	return nil
}

// ConsumeSAMLRequest removes and returns whether a SAML request ID exists and is not expired.
func (db *DB) ConsumeSAMLRequest(ctx context.Context, requestID string, providerID uuid.UUID) (bool, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM saml_requests WHERE request_id = $1 AND provider_id = $2 AND expires_at > now()`,
		requestID, providerID,
	)
	if err != nil {
		return false, fmt.Errorf("consuming SAML request: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ConsumeOrRecordSAMLAssertion atomically attempts to record a SAML assertion ID.
// It returns (true, nil) if the assertion was already consumed (replay detected),
// or (false, nil) if the assertion was successfully recorded for the first time.
// This avoids the TOCTOU race of separate check-then-insert calls.
func (db *DB) ConsumeOrRecordSAMLAssertion(ctx context.Context, assertionID string, providerID uuid.UUID, expiresAt time.Time) (bool, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	tag, err := db.Pool.Exec(ctx,
		`INSERT INTO saml_consumed_assertions (assertion_id, provider_id, expires_at) VALUES ($1, $2, $3)
		 ON CONFLICT (assertion_id, provider_id) DO NOTHING`,
		assertionID, providerID, expiresAt,
	)
	if err != nil {
		return false, fmt.Errorf("consuming SAML assertion: %w", err)
	}
	// If no rows were inserted, the assertion was already consumed (replay).
	return tag.RowsAffected() == 0, nil
}

// DeleteExpiredSAMLRequests removes expired SAML request and assertion records.
func (db *DB) DeleteExpiredSAMLRequests(ctx context.Context) (int64, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	tag1, err := db.Pool.Exec(ctx, `DELETE FROM saml_requests WHERE expires_at < now()`)
	if err != nil {
		return 0, fmt.Errorf("deleting expired SAML requests: %w", err)
	}
	tag2, err := db.Pool.Exec(ctx, `DELETE FROM saml_consumed_assertions WHERE expires_at < now()`)
	if err != nil {
		return tag1.RowsAffected(), fmt.Errorf("deleting expired SAML assertions: %w", err)
	}
	return tag1.RowsAffected() + tag2.RowsAffected(), nil
}
