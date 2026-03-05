DROP INDEX IF EXISTS idx_oauth_clients_org_enabled;

ALTER TABLE oauth_clients
    DROP COLUMN IF EXISTS enabled,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS client_secret_hash;
