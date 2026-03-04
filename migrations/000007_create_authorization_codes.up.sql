CREATE TABLE authorization_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code_hash       BYTEA NOT NULL UNIQUE,
    client_id       VARCHAR(128) NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    redirect_uri    TEXT NOT NULL,
    code_challenge  VARCHAR(128) NOT NULL,
    scope           VARCHAR(512) NOT NULL DEFAULT 'openid',
    used            BOOLEAN NOT NULL DEFAULT false,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_authorization_codes_expires_at ON authorization_codes(expires_at);
