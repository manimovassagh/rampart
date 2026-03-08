-- WebAuthn/Passkey credentials
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id   BYTEA NOT NULL UNIQUE,
    public_key      BYTEA NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT '',
    transport       TEXT[] NOT NULL DEFAULT '{}',
    flags_raw       SMALLINT NOT NULL DEFAULT 0,
    aaguid          BYTEA NOT NULL DEFAULT '\x00000000000000000000000000000000',
    sign_count      BIGINT NOT NULL DEFAULT 0,
    name            TEXT NOT NULL DEFAULT 'Passkey',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);

-- Temporary session data for WebAuthn ceremonies (short-lived)
CREATE TABLE IF NOT EXISTS webauthn_session_data (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_data BYTEA NOT NULL,
    ceremony    TEXT NOT NULL, -- 'registration' or 'login'
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_webauthn_session_data_user_id ON webauthn_session_data(user_id);
CREATE INDEX idx_webauthn_session_data_expires ON webauthn_session_data(expires_at);
