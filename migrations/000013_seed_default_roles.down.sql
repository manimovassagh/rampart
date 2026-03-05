-- Remove seeded role assignments and roles
DELETE FROM user_roles WHERE role_id IN (
    SELECT r.id FROM roles r
    JOIN organizations o ON r.org_id = o.id
    WHERE o.slug = 'default' AND r.builtin = true
);

DELETE FROM roles WHERE builtin = true AND org_id IN (
    SELECT id FROM organizations WHERE slug = 'default'
);
