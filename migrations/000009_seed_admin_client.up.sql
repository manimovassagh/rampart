INSERT INTO oauth_clients (id, org_id, name, client_type, redirect_uris)
SELECT
    'rampart-admin',
    o.id,
    'Rampart Admin Console',
    'public',
    ARRAY['http://localhost:8080/admin/callback']
FROM organizations o
WHERE o.slug = 'default'
ON CONFLICT (id) DO NOTHING;
