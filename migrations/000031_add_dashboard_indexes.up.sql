-- Composite index for CountRecentUsers: filters by org_id + created_at range.
CREATE INDEX IF NOT EXISTS idx_users_org_created ON users (org_id, created_at DESC);

-- Composite index for LoginCountsByDay: filters by org_id, event_type, and created_at.
CREATE INDEX IF NOT EXISTS idx_audit_events_org_type_created ON audit_events (org_id, event_type, created_at DESC);
