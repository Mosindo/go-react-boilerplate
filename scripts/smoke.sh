#!/usr/bin/env bash
# Runtime smoke test against a running API (local, staging...).
#   ./scripts/smoke.sh [API_BASE_URL]
# Checks health, registration, login, session, profile creation and
# discovery gating, then deletes the temporary account.
set -euo pipefail

API="${1:-http://localhost:18080}"
EMAIL="smoke-$(date +%s)-$RANDOM@smoke.test"
PASSWORD="SmokeTest2026"

json() { python3 -c "import sys, json; print(json.load(sys.stdin)$1)"; }
req() { curl -sS -f -H "Content-Type: application/json" "$@"; }

echo "[smoke] API=$API"
req "$API/health" | grep -q '"ok"' && echo "[smoke] health ok"

TOKEN=$(req -X POST "$API/auth/register" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | json "['accessToken']")
echo "[smoke] register ok"
req -X POST "$API/auth/login" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" >/dev/null && echo "[smoke] login ok"
req -H "Authorization: Bearer $TOKEN" "$API/me" | grep -q "$EMAIL" && echo "[smoke] /me ok"

req -X PUT -H "Authorization: Bearer $TOKEN" "$API/profile" \
  -d '{"firstName":"Smoke","birthdate":"1990-01-01","gender":"woman"}' >/dev/null && echo "[smoke] profile ok"
code=$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" "$API/discovery")
[ "$code" = "409" ] && echo "[smoke] discovery gated until profile complete ok"
req -H "Authorization: Bearer $TOKEN" "$API/matches" >/dev/null && echo "[smoke] matches ok"
req -H "Authorization: Bearer $TOKEN" "$API/notifications" >/dev/null && echo "[smoke] notifications ok"

curl -sS -f -X DELETE -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" "$API/me" -d "{\"password\":\"$PASSWORD\"}" >/dev/null
echo "[smoke] account deleted — all checks passed"
