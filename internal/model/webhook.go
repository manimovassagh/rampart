package model

import (
	"time"

	"github.com/google/uuid"
)

// Webhook represents a row in the webhooks table.
type Webhook struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	URL         string    `json:"url"`
	Secret      string    `json:"secret,omitempty"`
	Events      []string  `json:"events"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WebhookResponse is the admin-facing representation (secret omitted).
type WebhookResponse struct {
	ID          uuid.UUID `json:"id"`
	URL         string    `json:"url"`
	Events      []string  `json:"events"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts a Webhook to its admin response representation.
func (w *Webhook) ToResponse() *WebhookResponse {
	return &WebhookResponse{
		ID:          w.ID,
		URL:         w.URL,
		Events:      w.Events,
		Enabled:     w.Enabled,
		Description: w.Description,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}

// WebhookDelivery represents a row in the webhook_deliveries table.
type WebhookDelivery struct {
	ID             uuid.UUID  `json:"id"`
	WebhookID      uuid.UUID  `json:"webhook_id"`
	EventType      string     `json:"event_type"`
	Payload        []byte     `json:"payload"`
	ResponseStatus int        `json:"response_status,omitempty"`
	ResponseBody   string     `json:"response_body,omitempty"`
	DeliveredAt    time.Time  `json:"delivered_at"`
	Success        bool       `json:"success"`
	Attempt        int        `json:"attempt"`
	NextRetryAt    *time.Time `json:"next_retry_at,omitempty"`
	Error          string     `json:"error,omitempty"`
}

// WebhookPayload is the JSON body sent to webhook endpoints.
type WebhookPayload struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
	WebhookID uuid.UUID `json:"webhook_id"`
}

// CreateWebhookRequest is the JSON body for POST /api/v1/admin/webhooks.
type CreateWebhookRequest struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Enabled     bool     `json:"enabled"`
	Description string   `json:"description"`
}

// UpdateWebhookRequest is the JSON body for PUT /api/v1/admin/webhooks/{id}.
type UpdateWebhookRequest struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Enabled     bool     `json:"enabled"`
	Description string   `json:"description"`
}
