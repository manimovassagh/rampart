package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateAuditEvent inserts a new audit event and populates the ID and CreatedAt fields.
func (db *DB) CreateAuditEvent(ctx context.Context, event *model.AuditEvent) error {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_events (org_id, event_type, actor_id, actor_name, target_type, target_id, target_name, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	err = db.Pool.QueryRow(ctx, query,
		event.OrgID, event.EventType, event.ActorID, event.ActorName,
		event.TargetType, event.TargetID, event.TargetName,
		event.IPAddress, event.UserAgent, detailsJSON).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting audit event: %w", err)
	}
	return nil
}

// GetAuditEventByID returns a single audit event by ID.
func (db *DB) GetAuditEventByID(ctx context.Context, id uuid.UUID) (*model.AuditEvent, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	var e model.AuditEvent
	var detailsJSON []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT id, org_id, event_type, actor_id, actor_name, target_type, target_id, target_name, ip_address, user_agent, details, created_at
		 FROM audit_events WHERE id = $1`, id,
	).Scan(&e.ID, &e.OrgID, &e.EventType, &e.ActorID, &e.ActorName,
		&e.TargetType, &e.TargetID, &e.TargetName, &e.IPAddress, &e.UserAgent,
		&detailsJSON, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting audit event: %w", err)
	}
	if len(detailsJSON) > 0 {
		if unmarshalErr := json.Unmarshal(detailsJSON, &e.Details); unmarshalErr != nil {
			slog.Warn("failed to unmarshal audit event details", "event_id", e.ID, "error", unmarshalErr)
		}
	}
	return &e, nil
}

// ListAuditEvents returns a paginated, filterable list of audit events.
func (db *DB) ListAuditEvents(ctx context.Context, orgID uuid.UUID, eventType, search string, limit, offset int) ([]*model.AuditEvent, int, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

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
			if unmarshalErr := json.Unmarshal(detailsJSON, &e.Details); unmarshalErr != nil {
				slog.Warn("failed to unmarshal audit event details", "event_id", e.ID, "error", unmarshalErr)
			}
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating audit event rows: %w", err)
	}

	return events, total, nil
}

// LoginCountsByDay returns login event counts per day for the last N days.
func (db *DB) LoginCountsByDay(ctx context.Context, orgID uuid.UUID, days int) ([]model.DayCount, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	query := `
		SELECT d::date AS day, COALESCE(c.cnt, 0) AS count
		FROM generate_series(
			(now() - make_interval(days => $2))::date,
			now()::date,
			'1 day'::interval
		) AS d
		LEFT JOIN (
			SELECT created_at::date AS day, COUNT(*) AS cnt
			FROM audit_events
			WHERE org_id = $1
			  AND event_type IN ('user.login', 'user.login_failed')
			  AND created_at > now() - make_interval(days => $2)
			GROUP BY created_at::date
		) c ON c.day = d::date
		ORDER BY d`

	rows, err := db.Pool.Query(ctx, query, orgID, days)
	if err != nil {
		return nil, fmt.Errorf("counting logins by day: %w", err)
	}
	defer rows.Close()

	var counts []model.DayCount
	for rows.Next() {
		var dc model.DayCount
		var t time.Time
		if err := rows.Scan(&t, &dc.Count); err != nil {
			return nil, fmt.Errorf("scanning login day count: %w", err)
		}
		dc.Day = t.Format("Mon")
		counts = append(counts, dc)
	}
	return counts, rows.Err()
}

// CountRecentEvents returns the number of audit events in the last N hours.
func (db *DB) CountRecentEvents(ctx context.Context, orgID uuid.UUID, hours int) (int, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	var count int
	err := db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM audit_events WHERE org_id = $1 AND created_at > now() - make_interval(hours => $2)",
		orgID, hours).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting recent events: %w", err)
	}
	return count, nil
}
