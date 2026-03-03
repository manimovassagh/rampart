CREATE TABLE organization_settings (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                      UUID NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    password_min_length         INT NOT NULL DEFAULT 8,
    password_require_uppercase  BOOLEAN NOT NULL DEFAULT true,
    password_require_lowercase  BOOLEAN NOT NULL DEFAULT true,
    password_require_numbers    BOOLEAN NOT NULL DEFAULT true,
    password_require_symbols    BOOLEAN NOT NULL DEFAULT true,
    mfa_enforcement             VARCHAR(20) NOT NULL DEFAULT 'off'
                                CHECK (mfa_enforcement IN ('off', 'optional', 'required')),
    access_token_ttl            INTERVAL NOT NULL DEFAULT '15 minutes',
    refresh_token_ttl           INTERVAL NOT NULL DEFAULT '7 days',
    logo_url                    VARCHAR(2048),
    primary_color               VARCHAR(7),
    background_color            VARCHAR(7),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO organization_settings (org_id)
SELECT id FROM organizations WHERE slug = 'default'
ON CONFLICT (org_id) DO NOTHING;
