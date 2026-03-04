INSERT INTO oauth_clients (id, org_id, name, client_type, redirect_uris)
SELECT
    'sample-react-app',
    o.id,
    'Sample React App',
    'public',
    ARRAY['http://localhost:3002/callback']
FROM organizations o
WHERE o.slug = 'default'
ON CONFLICT (id) DO NOTHING;
