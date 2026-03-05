ALTER TABLE organization_settings
    ADD COLUMN self_registration_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN email_verification_required BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN forgot_password_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN remember_me_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN login_page_title VARCHAR(255) DEFAULT '',
    ADD COLUMN login_page_message TEXT DEFAULT '';
