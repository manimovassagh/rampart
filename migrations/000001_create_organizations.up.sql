CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the default organization
INSERT INTO organizations (name, slug, display_name)
VALUES ('default', 'default', 'Default Organization')
ON CONFLICT (slug) DO NOTHING;
