-- Track SAML AuthnRequest IDs so we can validate InResponseTo in the ACS callback.
CREATE TABLE IF NOT EXISTS saml_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id  TEXT NOT NULL UNIQUE,
    provider_id UUID NOT NULL REFERENCES saml_providers(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Track consumed SAML assertion IDs to prevent replay attacks.
CREATE TABLE IF NOT EXISTS saml_consumed_assertions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assertion_id TEXT NOT NULL,
    provider_id  UUID NOT NULL REFERENCES saml_providers(id) ON DELETE CASCADE,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(assertion_id, provider_id)
);

CREATE INDEX idx_saml_requests_expires ON saml_requests (expires_at);
CREATE INDEX idx_saml_consumed_assertions_expires ON saml_consumed_assertions (expires_at);
