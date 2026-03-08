package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateWebhook inserts a new webhook subscription.
func (db *DB) CreateWebhook(ctx context.Context, w *model.Webhook) (*model.Webhook, error) {
	var wh model.Webhook
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO webhooks (org_id, url, secret, description, event_types, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, org_id, url, secret, description, event_types, enabled, created_at, updated_at`,
		w.OrgID, w.URL, w.Secret, w.Description, w.EventTypes, w.Enabled,
	).Scan(&wh.ID, &wh.OrgID, &wh.URL, &wh.Secret, &wh.Description, &wh.EventTypes, &wh.Enabled, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating webhook: %w", err)
	}
	return &wh, nil
}

// GetWebhookByID returns a webhook by its ID.
func (db *DB) GetWebhookByID(ctx context.Context, id uuid.UUID) (*model.Webhook, error) {
	var wh model.Webhook
	err := db.Pool.QueryRow(ctx,
		`SELECT id, org_id, url, secret, description, event_types, enabled, created_at, updated_at
		 FROM webhooks WHERE id = $1`,
		id,
	).Scan(&wh.ID, &wh.OrgID, &wh.URL, &wh.Secret, &wh.Description, &wh.EventTypes, &wh.Enabled, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting webhook: %w", err)
	}
	return &wh, nil
}

// ListWebhooks returns paginated webhooks for an organization.
func (db *DB) ListWebhooks(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.Webhook, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhooks WHERE org_id = $1`, orgID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting webhooks: %w", err)
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, org_id, url, secret, description, event_types, enabled, created_at, updated_at
		 FROM webhooks WHERE org_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		orgID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("listing webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.Webhook
	for rows.Next() {
		var wh model.Webhook
		if err := rows.Scan(&wh.ID, &wh.OrgID, &wh.URL, &wh.Secret, &wh.Description, &wh.EventTypes, &wh.Enabled, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning webhook: %w", err)
		}
		webhooks = append(webhooks, &wh)
	}
	return webhooks, total, nil
}

// UpdateWebhook updates a webhook's settings.
func (db *DB) UpdateWebhook(ctx context.Context, id uuid.UUID, req *model.UpdateWebhookRequest) (*model.Webhook, error) {
	var wh model.Webhook
	err := db.Pool.QueryRow(ctx,
		`UPDATE webhooks SET url = $2, description = $3, event_types = $4, enabled = $5, updated_at = now()
		 WHERE id = $1
		 RETURNING id, org_id, url, secret, description, event_types, enabled, created_at, updated_at`,
		id, req.URL, req.Description, req.EventTypes, req.Enabled,
	).Scan(&wh.ID, &wh.OrgID, &wh.URL, &wh.Secret, &wh.Description, &wh.EventTypes, &wh.Enabled, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("updating webhook: %w", err)
	}
	return &wh, nil
}

// DeleteWebhook deletes a webhook by ID.
func (db *DB) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	return err
}

// GetEnabledWebhooksForEvent returns all enabled webhooks for an org that match the given event type.
func (db *DB) GetEnabledWebhooksForEvent(ctx context.Context, orgID uuid.UUID, eventType string) ([]*model.Webhook, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, org_id, url, secret, description, event_types, enabled, created_at, updated_at
		 FROM webhooks
		 WHERE org_id = $1 AND enabled = true
		   AND (event_types @> ARRAY[$2]::text[] OR event_types @> ARRAY['*']::text[])`,
		orgID, eventType,
	)
	if err != nil {
		return nil, fmt.Errorf("getting webhooks for event: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.Webhook
	for rows.Next() {
		var wh model.Webhook
		if err := rows.Scan(&wh.ID, &wh.OrgID, &wh.URL, &wh.Secret, &wh.Description, &wh.EventTypes, &wh.Enabled, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning webhook: %w", err)
		}
		webhooks = append(webhooks, &wh)
	}
	return webhooks, nil
}

// CreateWebhookDelivery inserts a new delivery record.
func (db *DB) CreateWebhookDelivery(ctx context.Context, d *model.WebhookDelivery) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO webhook_deliveries (webhook_id, event_id, status, attempts, next_retry_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		d.WebhookID, d.EventID, d.Status, d.Attempts, d.NextRetryAt,
	)
	return err
}

// UpdateWebhookDelivery updates a delivery record after an attempt.
func (db *DB) UpdateWebhookDelivery(ctx context.Context, id uuid.UUID, status string, attempts int, responseCode *int, lastError string, nextRetry *time.Time, completedAt *time.Time) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE webhook_deliveries
		 SET status = $2, attempts = $3, last_response_code = $4, last_error = $5, next_retry_at = $6, completed_at = $7
		 WHERE id = $1`,
		id, status, attempts, responseCode, lastError, nextRetry, completedAt,
	)
	return err
}

// GetPendingDeliveries returns deliveries ready for retry.
// Uses FOR UPDATE SKIP LOCKED to prevent duplicate delivery in multi-instance deployments.
func (db *DB) GetPendingDeliveries(ctx context.Context, limit int) ([]*model.WebhookDelivery, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT d.id, d.webhook_id, d.event_id, d.status, d.attempts, d.next_retry_at,
		        d.last_response_code, d.last_error, d.created_at, d.completed_at
		 FROM webhook_deliveries d
		 WHERE d.status = 'pending' AND (d.next_retry_at IS NULL OR d.next_retry_at <= now())
		 ORDER BY d.created_at ASC
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("getting pending deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventID, &d.Status, &d.Attempts, &d.NextRetryAt, &d.LastResponseCode, &d.LastError, &d.CreatedAt, &d.CompletedAt); err != nil {
			return nil, fmt.Errorf("scanning delivery: %w", err)
		}
		deliveries = append(deliveries, &d)
	}
	return deliveries, nil
}

// DeleteOldDeliveries removes completed deliveries older than the given duration.
func (db *DB) DeleteOldDeliveries(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM webhook_deliveries WHERE status IN ('success', 'failed') AND completed_at < $1`,
		time.Now().Add(-olderThan),
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListWebhookDeliveries returns recent deliveries for a webhook.
func (db *DB) ListWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*model.WebhookDelivery, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_deliveries WHERE webhook_id = $1`, webhookID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting deliveries: %w", err)
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, webhook_id, event_id, status, attempts, next_retry_at,
		        last_response_code, last_error, created_at, completed_at
		 FROM webhook_deliveries WHERE webhook_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		webhookID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("listing deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventID, &d.Status, &d.Attempts, &d.NextRetryAt, &d.LastResponseCode, &d.LastError, &d.CreatedAt, &d.CompletedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning delivery: %w", err)
		}
		deliveries = append(deliveries, &d)
	}
	return deliveries, total, nil
}
