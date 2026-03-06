CREATE TABLE IF NOT EXISTS social_provider_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    client_id   TEXT NOT NULL DEFAULT '',
    client_secret TEXT NOT NULL DEFAULT '',
    extra_config JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, provider)
);
