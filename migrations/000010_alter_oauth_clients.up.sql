ALTER TABLE oauth_clients
    ADD COLUMN IF NOT EXISTS client_secret_hash BYTEA,
    ADD COLUMN IF NOT EXISTS description VARCHAR(500) DEFAULT '',
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_oauth_clients_org_enabled ON oauth_clients(org_id, enabled);
