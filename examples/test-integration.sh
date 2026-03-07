#!/usr/bin/env bash
# Integration test: Rampart + React app + Node.js API
# Tests registration, login, JWT validation, role-based access, token revocation, and lockout.
#
# Prerequisites:
#   - Docker and docker-compose
#   - Node.js >= 18
#   - curl, jq
#
# Usage: ./test-integration.sh
set -euo pipefail

RAMPART_URL="http://localhost:8080"
NODE_API_URL="http://localhost:3001"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PASS_COUNT=0
FAIL_COUNT=0

pass() { echo "PASS: $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
fail() { echo "FAIL: $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }

cleanup() {
  echo ""
  echo "=== Cleaning up ==="
  if [ -n "${NODE_PID:-}" ]; then
    kill "$NODE_PID" 2>/dev/null || true
    wait "$NODE_PID" 2>/dev/null || true
  fi
  cd "$ROOT_DIR"
  docker compose down -v --remove-orphans 2>/dev/null || true
  rm -rf "$SCRIPT_DIR/node-api/node_modules" "$SCRIPT_DIR/react-app/node_modules"
  rm -f /tmp/rampart-test-*.json
  echo "=== Cleanup complete ==="
}
trap cleanup EXIT

# --- Start Rampart ---
echo "=== Starting Rampart stack ==="
cd "$ROOT_DIR"
RAMPART_ALLOWED_ORIGINS="http://localhost:5173,http://localhost:3001" \
  docker compose up -d --build

echo "Waiting for Rampart..."
for i in $(seq 1 30); do
  if curl -sf "$RAMPART_URL/healthz" > /dev/null 2>&1; then
    echo "Rampart is ready."
    break
  fi
  [ "$i" -eq 30 ] && { echo "ERROR: Rampart failed to start"; exit 1; }
  sleep 1
done

# --- OIDC Discovery ---
echo ""
echo "=== Test: OIDC Discovery ==="
DISCOVERY=$(curl -sf "$RAMPART_URL/.well-known/openid-configuration")
echo "$DISCOVERY" | jq -r '.issuer, .authorization_endpoint, .token_endpoint, .revocation_endpoint'
pass "OIDC Discovery"

# --- JWKS ---
echo ""
echo "=== Test: JWKS ==="
JWKS=$(curl -sf "$RAMPART_URL/.well-known/jwks.json")
KEY_COUNT=$(echo "$JWKS" | jq '.keys | length')
[ "$KEY_COUNT" -ge 1 ] && pass "JWKS has $KEY_COUNT key(s)" || fail "JWKS has no keys"

# --- Register users (use heredoc files to avoid shell escaping issues) ---
echo ""
echo "=== Setup: Register users ==="
cat <<'EOF' > /tmp/rampart-test-admin.json
{"username":"testadmin","email":"testadmin@test.com","password":"Admin1234!@","given_name":"Admin","family_name":"User"}
EOF
cat <<'EOF' > /tmp/rampart-test-viewer.json
{"username":"testviewer","email":"testviewer@test.com","password":"Viewer1234!@","given_name":"View","family_name":"User"}
EOF

ADMIN_REG=$(curl -s -X POST "$RAMPART_URL/register" -H 'Content-Type: application/json' -d @/tmp/rampart-test-admin.json)
ADMIN_ID=$(echo "$ADMIN_REG" | jq -r '.id // empty')
[ -n "$ADMIN_ID" ] && pass "Admin registered: $ADMIN_ID" || fail "Admin registration: $(echo "$ADMIN_REG" | jq -r '.error_description // .error')"

VIEWER_REG=$(curl -s -X POST "$RAMPART_URL/register" -H 'Content-Type: application/json' -d @/tmp/rampart-test-viewer.json)
VIEWER_ID=$(echo "$VIEWER_REG" | jq -r '.id // empty')
[ -n "$VIEWER_ID" ] && pass "Viewer registered: $VIEWER_ID" || fail "Viewer registration"

ORG_ID=$(echo "$ADMIN_REG" | jq -r '.org_id')

# --- Create admin role and assign to admin user via DB ---
echo ""
echo "=== Setup: Create admin role ==="
docker compose exec -T postgres psql -U rampart -d rampart -q -c \
  "INSERT INTO roles (id, org_id, name, description) VALUES (gen_random_uuid(), '$ORG_ID', 'admin', 'Administrator') ON CONFLICT DO NOTHING;"
ROLE_ID=$(docker compose exec -T postgres psql -U rampart -d rampart -t -A -c \
  "SELECT id FROM roles WHERE org_id = '$ORG_ID' AND name = 'admin' LIMIT 1;")
docker compose exec -T postgres psql -U rampart -d rampart -q -c \
  "INSERT INTO user_roles (user_id, role_id) VALUES ('$ADMIN_ID', '$ROLE_ID') ON CONFLICT DO NOTHING;"
pass "Admin role assigned"

# --- Login ---
echo ""
echo "=== Test: Login ==="
cat <<'EOF' > /tmp/rampart-test-admin-login.json
{"identifier":"testadmin@test.com","password":"Admin1234!@"}
EOF
cat <<'EOF' > /tmp/rampart-test-viewer-login.json
{"identifier":"testviewer@test.com","password":"Viewer1234!@"}
EOF

ADMIN_LOGIN=$(curl -s -X POST "$RAMPART_URL/login" -H 'Content-Type: application/json' -d @/tmp/rampart-test-admin-login.json)
ADMIN_TOKEN=$(echo "$ADMIN_LOGIN" | jq -r '.access_token // empty')
ADMIN_REFRESH=$(echo "$ADMIN_LOGIN" | jq -r '.refresh_token // empty')
[ -n "$ADMIN_TOKEN" ] && pass "Admin login" || fail "Admin login: $(echo "$ADMIN_LOGIN" | jq -r '.error_description')"

VIEWER_LOGIN=$(curl -s -X POST "$RAMPART_URL/login" -H 'Content-Type: application/json' -d @/tmp/rampart-test-viewer-login.json)
VIEWER_TOKEN=$(echo "$VIEWER_LOGIN" | jq -r '.access_token // empty')
[ -n "$VIEWER_TOKEN" ] && pass "Viewer login" || fail "Viewer login"

# --- Verify admin JWT has roles ---
echo ""
echo "=== Test: JWT claims ==="
ADMIN_ROLES=$(echo "$ADMIN_TOKEN" | python3 -c "
import sys, json, base64
t = sys.stdin.read().strip().split('.')[1]
t += '=' * (4 - len(t) % 4)
print(json.dumps(json.loads(base64.urlsafe_b64decode(t)).get('roles', [])))
")
echo "Admin roles: $ADMIN_ROLES"
echo "$ADMIN_ROLES" | grep -q '"admin"' && pass "Admin JWT has admin role" || fail "Admin JWT missing admin role"

# --- /me endpoint ---
echo ""
echo "=== Test: /me endpoint ==="
ME_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$RAMPART_URL/me" -H "Authorization: Bearer $ADMIN_TOKEN")
[ "$ME_STATUS" = "200" ] && pass "/me returns 200" || fail "/me returns $ME_STATUS"

# --- Token refresh ---
echo ""
echo "=== Test: Token refresh ==="
REFRESH_BODY=$(printf '{"refresh_token":"%s"}' "$ADMIN_REFRESH")
REFRESH_RESP=$(curl -s -X POST "$RAMPART_URL/token/refresh" -H 'Content-Type: application/json' -d "$REFRESH_BODY")
NEW_TOKEN=$(echo "$REFRESH_RESP" | jq -r '.access_token // empty')
[ -n "$NEW_TOKEN" ] && pass "Token refresh" || fail "Token refresh"
ADMIN_TOKEN="$NEW_TOKEN"

# --- Start Node.js API ---
echo ""
echo "=== Starting Node.js API ==="
cd "$SCRIPT_DIR/node-api"
npm install --silent 2>&1
node server.js &
NODE_PID=$!

for i in $(seq 1 10); do
  curl -sf "$NODE_API_URL/api/health" > /dev/null 2>&1 && break
  [ "$i" -eq 10 ] && { echo "ERROR: Node.js API failed to start"; exit 1; }
  sleep 1
done
pass "Node.js API started"

# --- Node.js API tests ---
echo ""
echo "=== Test: Node.js API - JWT validation ==="

# Admin profile
PROFILE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$NODE_API_URL/api/user/profile" -H "Authorization: Bearer $ADMIN_TOKEN")
[ "$PROFILE_STATUS" = "200" ] && pass "Admin profile: 200" || fail "Admin profile: $PROFILE_STATUS"

PROFILE_ROLES=$(curl -s "$NODE_API_URL/api/user/profile" -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.roles[]' 2>/dev/null)
echo "$PROFILE_ROLES" | grep -q "admin" && pass "Backend sees admin role" || fail "Backend missing admin role"

# Viewer profile
VIEWER_PROFILE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$NODE_API_URL/api/user/profile" -H "Authorization: Bearer $VIEWER_TOKEN")
[ "$VIEWER_PROFILE_STATUS" = "200" ] && pass "Viewer profile: 200" || fail "Viewer profile: $VIEWER_PROFILE_STATUS"

echo ""
echo "=== Test: Node.js API - Role enforcement ==="

# Admin accessing admin endpoint
ADMIN_DATA_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$NODE_API_URL/api/admin/data" -H "Authorization: Bearer $ADMIN_TOKEN")
[ "$ADMIN_DATA_STATUS" = "200" ] && pass "Admin data: 200 (admin role)" || fail "Admin data: $ADMIN_DATA_STATUS"

# Viewer accessing admin endpoint (should be 403)
VIEWER_ADMIN_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$NODE_API_URL/api/admin/data" -H "Authorization: Bearer $VIEWER_TOKEN")
[ "$VIEWER_ADMIN_STATUS" = "403" ] && pass "Viewer admin data: 403 (no admin role)" || fail "Viewer admin data: $VIEWER_ADMIN_STATUS"

# No auth (should be 401)
NO_AUTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$NODE_API_URL/api/user/profile")
[ "$NO_AUTH_STATUS" = "401" ] && pass "No auth: 401" || fail "No auth: $NO_AUTH_STATUS"

# --- Token revocation ---
echo ""
echo "=== Test: Token revocation ==="
REVOKE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$RAMPART_URL/oauth/revoke" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "token=$ADMIN_REFRESH")
[ "$REVOKE_STATUS" = "200" ] && pass "Token revocation: 200" || fail "Token revocation: $REVOKE_STATUS"

# --- Account lockout ---
echo ""
echo "=== Test: Account lockout ==="
cat <<'EOF' > /tmp/rampart-test-bad-login.json
{"identifier":"testviewer@test.com","password":"WrongPassword"}
EOF
for i in $(seq 1 6); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$RAMPART_URL/login" \
    -H 'Content-Type: application/json' -d @/tmp/rampart-test-bad-login.json)
  echo "  Attempt $i: HTTP $STATUS"
done
# The 6th attempt should show the account is locked
[ "$STATUS" != "200" ] && pass "Account lockout (blocked after failed attempts)" || fail "Account not locked"

# --- Summary ---
echo ""
echo "============================================"
echo "  Results: $PASS_COUNT passed, $FAIL_COUNT failed"
echo "============================================"
[ "$FAIL_COUNT" -eq 0 ] && exit 0 || exit 1
