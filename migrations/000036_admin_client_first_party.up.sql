-- Mark the admin console client as first-party so it skips the consent screen.
-- First-party clients are trusted apps that don't need user consent for OAuth flows.
UPDATE oauth_clients SET first_party = true WHERE id = 'rampart-admin';
