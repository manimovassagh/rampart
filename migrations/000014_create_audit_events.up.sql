CREATE TABLE audit_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_type  VARCHAR(50) NOT NULL,
    actor_id    UUID,
    actor_name  VARCHAR(255) DEFAULT '',
    target_type VARCHAR(50) DEFAULT '',
    target_id   VARCHAR(255) DEFAULT '',
    target_name VARCHAR(255) DEFAULT '',
    ip_address  VARCHAR(45) DEFAULT '',
    user_agent  VARCHAR(500) DEFAULT '',
    details     JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_events_org_created ON audit_events(org_id, created_at DESC);
CREATE INDEX idx_audit_events_type ON audit_events(event_type);
CREATE INDEX idx_audit_events_actor ON audit_events(actor_id);
