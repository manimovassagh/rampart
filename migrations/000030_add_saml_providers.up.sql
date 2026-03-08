-- SAML Identity Provider configurations for SP-initiated SSO
CREATE TABLE IF NOT EXISTS saml_providers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    metadata_url    TEXT NOT NULL DEFAULT '',
    metadata_xml    TEXT NOT NULL DEFAULT '',
    sso_url         TEXT NOT NULL,
    slo_url         TEXT NOT NULL DEFAULT '',
    certificate     TEXT NOT NULL,
    name_id_format  TEXT NOT NULL DEFAULT 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
    attribute_mapping JSONB NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, entity_id)
);

CREATE INDEX idx_saml_providers_org_id ON saml_providers(org_id);
