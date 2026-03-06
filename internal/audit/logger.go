package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// EventStore defines the database operations for audit events.
type EventStore interface {
	CreateAuditEvent(ctx context.Context, event *model.AuditEvent) error
}

// WebhookDispatcher defines the interface for dispatching webhook events.
type WebhookDispatcher interface {
	Dispatch(ctx context.Context, orgID uuid.UUID, eventType string, data any) error
}

// Logger provides fire-and-forget audit logging.
type Logger struct {
	store      EventStore
	logger     *slog.Logger
	dispatcher WebhookDispatcher
}

// NewLogger creates a new audit logger.
func NewLogger(store EventStore, logger *slog.Logger) *Logger {
	return &Logger{store: store, logger: logger}
}

// SetDispatcher attaches a webhook dispatcher to the audit logger.
// When set, audit events will also be dispatched as webhook events.
func (l *Logger) SetDispatcher(d WebhookDispatcher) {
	if l == nil {
		return
	}
	l.dispatcher = d
}

// Log records an audit event asynchronously (fire-and-forget).
// Safe to call on a nil receiver (no-op).
func (l *Logger) Log(ctx context.Context, r *http.Request, orgID uuid.UUID, eventType string, actorID *uuid.UUID, actorName, targetType, targetID, targetName string, details map[string]any) {
	if l == nil {
		return
	}
	event := &model.AuditEvent{
		OrgID:      orgID,
		EventType:  eventType,
		ActorID:    actorID,
		ActorName:  actorName,
		TargetType: targetType,
		TargetID:   targetID,
		TargetName: targetName,
		Details:    details,
	}

	if r != nil {
		event.IPAddress = extractIP(r)
		ua := r.UserAgent()
		if len(ua) > 500 {
			ua = ua[:500]
		}
		event.UserAgent = ua
	}

	go func() {
		if err := l.store.CreateAuditEvent(context.Background(), event); err != nil {
			l.logger.Error("failed to write audit event", "event_type", eventType, "error", err)
		}

		if l.dispatcher != nil {
			eventData := map[string]any{
				"actor_name":  actorName,
				"target_type": targetType,
				"target_id":   targetID,
				"target_name": targetName,
			}
			if details != nil {
				eventData["details"] = details
			}
			if err := l.dispatcher.Dispatch(context.Background(), orgID, eventType, eventData); err != nil {
				l.logger.Error("failed to dispatch webhook", "event_type", eventType, "error", err)
			}
		}
	}()
}

// LogSimple is a convenience wrapper for events with no extra details.
// Safe to call on a nil receiver (no-op).
func (l *Logger) LogSimple(ctx context.Context, r *http.Request, orgID uuid.UUID, eventType string, actorID *uuid.UUID, actorName, targetType, targetID, targetName string) {
	if l == nil {
		return
	}
	l.Log(ctx, r, orgID, eventType, actorID, actorName, targetType, targetID, targetName, nil)
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// MarshalDetails converts a struct to a map for the details field.
func MarshalDetails(v any) map[string]any {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
