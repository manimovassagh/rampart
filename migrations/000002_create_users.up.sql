CREATE TABLE users (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    username              VARCHAR(255) NOT NULL,
    email                 VARCHAR(255) NOT NULL,
    email_verified        BOOLEAN NOT NULL DEFAULT false,
    given_name            VARCHAR(255),
    family_name           VARCHAR(255),
    picture               VARCHAR(2048),
    phone_number          VARCHAR(50),
    phone_number_verified BOOLEAN NOT NULL DEFAULT false,
    password_hash         BYTEA,
    enabled               BOOLEAN NOT NULL DEFAULT true,
    mfa_enabled           BOOLEAN NOT NULL DEFAULT false,
    last_login_at         TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Multi-tenant unique constraints
CREATE UNIQUE INDEX idx_users_email_org ON users (email, org_id);
CREATE UNIQUE INDEX idx_users_username_org ON users (username, org_id);
CREATE INDEX idx_users_org ON users (org_id);
