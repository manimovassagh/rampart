package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateWebhook inserts a new webhook and returns it with generated fields.
func (db *DB) CreateWebhook(ctx context.Context, webhook *model.Webhook) (*model.Webhook, error) {
	query := `
		INSERT INTO webhooks (org_id, url, secret, events, enabled, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, url, secret, events, enabled, description, created_at, updated_at`

	var w model.Webhook
	err := db.Pool.QueryRow(ctx, query,
		webhook.OrgID, webhook.URL, webhook.Secret, webhook.Events,
		webhook.Enabled, webhook.Description,
	).Scan(&w.ID, &w.OrgID, &w.URL, &w.Secret, &w.Events, &w.Enabled,
		&w.Description, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting webhook: %w", err)
	}
	return &w, nil
}

// GetWebhook returns a webhook by ID.
func (db *DB) GetWebhook(ctx context.Context, id uuid.UUID) (*model.Webhook, error) {
	query := `
		SELECT id, org_id, url, secret, events, enabled, description, created_at, updated_at
		FROM webhooks WHERE id = $1`

	var w model.Webhook
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&w.ID, &w.OrgID, &w.URL, &w.Secret, &w.Events, &w.Enabled,
		&w.Description, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying webhook by id: %w", err)
	}
	return &w, nil
}

// ListWebhooks returns all webhooks for an organization.
func (db *DB) ListWebhooks(ctx context.Context, orgID uuid.UUID) ([]*model.Webhook, error) {
	query := `
		SELECT id, org_id, url, secret, events, enabled, description, created_at, updated_at
		FROM webhooks WHERE org_id = $1
		ORDER BY created_at DESC`

	rows, err := db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("listing webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.OrgID, &w.URL, &w.Secret, &w.Events, &w.Enabled,
			&w.Description, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning webhook row: %w", err)
		}
		webhooks = append(webhooks, &w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating webhook rows: %w", err)
	}
	return webhooks, nil
}

// UpdateWebhook updates a webhook's mutable fields.
func (db *DB) UpdateWebhook(ctx context.Context, webhook *model.Webhook) error {
	query := `
		UPDATE webhooks
		SET url = $2, events = $3, enabled = $4, description = $5, updated_at = now()
		WHERE id = $1`

	tag, err := db.Pool.Exec(ctx, query,
		webhook.ID, webhook.URL, webhook.Events, webhook.Enabled, webhook.Description)
	if err != nil {
		return fmt.Errorf("updating webhook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}

// DeleteWebhook removes a webhook by ID.
func (db *DB) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, "DELETE FROM webhooks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting webhook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}

// GetWebhooksForEvent returns all enabled webhooks for an org subscribed to a given event type.
func (db *DB) GetWebhooksForEvent(ctx context.Context, orgID uuid.UUID, eventType string) ([]*model.Webhook, error) {
	query := `
		SELECT id, org_id, url, secret, events, enabled, description, created_at, updated_at
		FROM webhooks
		WHERE org_id = $1 AND enabled = TRUE AND $2 = ANY(events)`

	rows, err := db.Pool.Query(ctx, query, orgID, eventType)
	if err != nil {
		return nil, fmt.Errorf("querying webhooks for event: %w", err)
	}
	defer rows.Close()

	var webhooks []*model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.OrgID, &w.URL, &w.Secret, &w.Events, &w.Enabled,
			&w.Description, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning webhook row: %w", err)
		}
		webhooks = append(webhooks, &w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating webhook rows: %w", err)
	}
	return webhooks, nil
}

// CreateWebhookDelivery inserts a delivery record.
func (db *DB) CreateWebhookDelivery(ctx context.Context, delivery *model.WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (webhook_id, event_type, payload, response_status, response_body, success, attempt, next_retry_at, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, delivered_at`

	return db.Pool.QueryRow(ctx, query,
		delivery.WebhookID, delivery.EventType, delivery.Payload,
		delivery.ResponseStatus, delivery.ResponseBody, delivery.Success,
		delivery.Attempt, delivery.NextRetryAt, delivery.Error,
	).Scan(&delivery.ID, &delivery.DeliveredAt)
}

// GetWebhookDeliveries returns recent deliveries for a webhook.
func (db *DB) GetWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*model.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_type, payload, response_status, response_body,
		       delivered_at, success, attempt, next_retry_at, error
		FROM webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY delivered_at DESC
		LIMIT $2`

	rows, err := db.Pool.Query(ctx, query, webhookID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		var payloadJSON []byte
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventType, &payloadJSON,
			&d.ResponseStatus, &d.ResponseBody, &d.DeliveredAt, &d.Success,
			&d.Attempt, &d.NextRetryAt, &d.Error); err != nil {
			return nil, fmt.Errorf("scanning webhook delivery row: %w", err)
		}
		d.Payload = payloadJSON
		deliveries = append(deliveries, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating webhook delivery rows: %w", err)
	}
	return deliveries, nil
}

// GetPendingRetries returns deliveries that need to be retried.
func (db *DB) GetPendingRetries(ctx context.Context, limit int) ([]*model.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_type, payload, response_status, response_body,
		       delivered_at, success, attempt, next_retry_at, error
		FROM webhook_deliveries
		WHERE success = FALSE AND next_retry_at IS NOT NULL AND next_retry_at <= now()
		ORDER BY next_retry_at ASC
		LIMIT $1`

	rows, err := db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("querying pending retries: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		var payloadJSON []byte
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventType, &payloadJSON,
			&d.ResponseStatus, &d.ResponseBody, &d.DeliveredAt, &d.Success,
			&d.Attempt, &d.NextRetryAt, &d.Error); err != nil {
			return nil, fmt.Errorf("scanning pending retry row: %w", err)
		}
		d.Payload = payloadJSON
		deliveries = append(deliveries, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating pending retry rows: %w", err)
	}
	return deliveries, nil
}

// UpdateWebhookDelivery updates a delivery record (used for retries).
func (db *DB) UpdateWebhookDelivery(ctx context.Context, delivery *model.WebhookDelivery) error {
	query := `
		UPDATE webhook_deliveries
		SET response_status = $2, response_body = $3, success = $4, attempt = $5,
		    next_retry_at = $6, error = $7, delivered_at = now()
		WHERE id = $1`

	tag, err := db.Pool.Exec(ctx, query,
		delivery.ID, delivery.ResponseStatus, delivery.ResponseBody,
		delivery.Success, delivery.Attempt, delivery.NextRetryAt, delivery.Error)
	if err != nil {
		return fmt.Errorf("updating webhook delivery: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("webhook delivery not found")
	}
	return nil
}
