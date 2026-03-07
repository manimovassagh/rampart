ALTER TABLE organization_settings
    DROP COLUMN IF EXISTS lockout_duration_seconds,
    DROP COLUMN IF EXISTS max_failed_login_attempts;
