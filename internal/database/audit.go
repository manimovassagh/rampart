package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateAuditEvent inserts a new audit event.
func (db *DB) CreateAuditEvent(ctx context.Context, event *model.AuditEvent) error {
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_events (org_id, event_type, actor_id, actor_name, target_type, target_id, target_name, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = db.Pool.Exec(ctx, query,
		event.OrgID, event.EventType, event.ActorID, event.ActorName,
		event.TargetType, event.TargetID, event.TargetName,
		event.IPAddress, event.UserAgent, detailsJSON)
	if err != nil {
		return fmt.Errorf("inserting audit event: %w", err)
	}
	return nil
}

// ListAuditEvents returns a paginated, filterable list of audit events.
func (db *DB) ListAuditEvents(ctx context.Context, orgID uuid.UUID, eventType, search string, limit, offset int) ([]*model.AuditEvent, int, error) {
	where := []string{"org_id = $1"}
	args := []any{orgID}
	paramIdx := 2

	if eventType != "" {
		where = append(where, fmt.Sprintf("event_type = $%d", paramIdx))
		args = append(args, eventType)
		paramIdx++
	}

	if search != "" {
		where = append(where, fmt.Sprintf(
			"(actor_name ILIKE $%d OR target_name ILIKE $%d OR target_id ILIKE $%d)",
			paramIdx, paramIdx, paramIdx,
		))
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_events WHERE %s", whereClause)
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting audit events: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, org_id, event_type, actor_id, actor_name, target_type, target_id, target_name, ip_address, user_agent, details, created_at
		FROM audit_events
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing audit events: %w", err)
	}
	defer rows.Close()

	var events []*model.AuditEvent
	for rows.Next() {
		var e model.AuditEvent
		var detailsJSON []byte
		if err := rows.Scan(&e.ID, &e.OrgID, &e.EventType, &e.ActorID, &e.ActorName,
			&e.TargetType, &e.TargetID, &e.TargetName, &e.IPAddress, &e.UserAgent,
			&detailsJSON, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning audit event row: %w", err)
		}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &e.Details)
		}
		events = append(events, &e)
	}

	return events, total, nil
}

// CountRecentEvents returns the number of audit events in the last N hours.
func (db *DB) CountRecentEvents(ctx context.Context, orgID uuid.UUID, hours int) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM audit_events WHERE org_id = $1 AND created_at > now() - make_interval(hours => $2)",
		orgID, hours).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting recent events: %w", err)
	}
	return count, nil
}
