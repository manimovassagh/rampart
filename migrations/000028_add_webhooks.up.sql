-- Webhook subscriptions for event delivery
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    secret TEXT NOT NULL,            -- HMAC-SHA256 signing secret
    description TEXT NOT NULL DEFAULT '',
    event_types TEXT[] NOT NULL,      -- filter: e.g. {"user.login","user.created"} or {"*"} for all
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhooks_org_id ON webhooks(org_id);

-- Delivery log for retry and observability
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES audit_events(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',   -- pending, success, failed
    attempts INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ,
    last_response_code INT,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_webhook_deliveries_pending ON webhook_deliveries(status, next_retry_at)
    WHERE status = 'pending';
CREATE INDEX idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id, created_at DESC);
