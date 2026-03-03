-- Seed default admin user (admin/admin) in the default organization.
-- The admin should change this password immediately after first login.
INSERT INTO users (
    org_id,
    username,
    email,
    email_verified,
    given_name,
    family_name,
    password_hash,
    enabled
)
SELECT
    o.id,
    'admin',
    'admin@rampart.local',
    true,
    'Admin',
    'User',
    convert_to('$argon2id$v=19$m=65536,t=3,p=4$1b9kW9M2rZ5NS3GN7rfChg$+BwS3tHBUjmH8+cFLjqxyRabRphKuzD7UTKnftXhBp8', 'UTF8'),
    true
FROM organizations o
WHERE o.slug = 'default'
ON CONFLICT DO NOTHING;
