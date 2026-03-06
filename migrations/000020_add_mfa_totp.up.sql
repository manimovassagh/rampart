CREATE TABLE mfa_totp_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT 'Authenticator',
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    UNIQUE(user_id, name)
);
CREATE INDEX idx_mfa_totp_devices_user_id ON mfa_totp_devices(user_id);

CREATE TABLE mfa_recovery_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at TIMESTAMPTZ
);
CREATE INDEX idx_mfa_recovery_codes_user_id ON mfa_recovery_codes(user_id);
