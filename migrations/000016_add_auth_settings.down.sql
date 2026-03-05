ALTER TABLE organization_settings
    DROP COLUMN IF EXISTS self_registration_enabled,
    DROP COLUMN IF EXISTS email_verification_required,
    DROP COLUMN IF EXISTS forgot_password_enabled,
    DROP COLUMN IF EXISTS remember_me_enabled,
    DROP COLUMN IF EXISTS login_page_title,
    DROP COLUMN IF EXISTS login_page_message;
