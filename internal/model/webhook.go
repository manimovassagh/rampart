package model

import (
	"time"

	"github.com/google/uuid"
)

// Webhook represents a webhook subscription for event delivery.
type Webhook struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	URL         string
	Secret      string `json:"-"`
	Description string
	EventTypes  []string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WebhookDelivery tracks the delivery status of an event to a webhook.
type WebhookDelivery struct {
	ID               uuid.UUID
	WebhookID        uuid.UUID
	EventID          uuid.UUID
	Status           string // pending, success, failed
	Attempts         int
	NextRetryAt      *time.Time
	LastResponseCode *int
	LastError        *string
	CreatedAt        time.Time
	CompletedAt      *time.Time
}

// CreateWebhookRequest is used when creating a new webhook.
type CreateWebhookRequest struct {
	URL         string   `json:"url"`
	Description string   `json:"description"`
	EventTypes  []string `json:"event_types"`
}

// UpdateWebhookRequest is used when updating a webhook.
type UpdateWebhookRequest struct {
	URL         string   `json:"url"`
	Description string   `json:"description"`
	EventTypes  []string `json:"event_types"`
	Enabled     bool     `json:"enabled"`
}

// WebhookPayload is the JSON body sent to webhook endpoints.
type WebhookPayload struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	OrgID     string         `json:"org_id"`
	Actor     *WebhookActor  `json:"actor,omitempty"`
	Target    *WebhookTarget `json:"target,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// WebhookActor identifies who performed the action.
type WebhookActor struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// WebhookTarget identifies the resource affected.
type WebhookTarget struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}
