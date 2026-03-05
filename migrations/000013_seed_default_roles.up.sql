-- Seed default roles in the default organization
INSERT INTO roles (org_id, name, description, builtin)
SELECT o.id, 'admin', 'Full administrative access', true
FROM organizations o WHERE o.slug = 'default'
ON CONFLICT (name, org_id) DO NOTHING;

INSERT INTO roles (org_id, name, description, builtin)
SELECT o.id, 'user', 'Standard user access', true
FROM organizations o WHERE o.slug = 'default'
ON CONFLICT (name, org_id) DO NOTHING;

-- Assign admin role to the seeded admin user
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN organizations o ON u.org_id = o.id
JOIN roles r ON r.org_id = o.id AND r.name = 'admin'
WHERE o.slug = 'default' AND u.username = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;
