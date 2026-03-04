CREATE TABLE oauth_clients (
    id              VARCHAR(128) PRIMARY KEY,
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    client_type     VARCHAR(20) NOT NULL DEFAULT 'public'
                    CHECK (client_type IN ('public', 'confidential')),
    redirect_uris   TEXT[] NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_clients_org_id ON oauth_clients(org_id);
