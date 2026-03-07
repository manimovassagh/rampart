ALTER TABLE organization_settings
    ADD COLUMN max_failed_login_attempts INT NOT NULL DEFAULT 5,
    ADD COLUMN lockout_duration_seconds INT NOT NULL DEFAULT 900;
