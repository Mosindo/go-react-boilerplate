#!/usr/bin/env bash
# Runs the browser E2E (register -> onboarding -> discover -> match -> realtime chat)
# against a freshly started API and the Expo web build.
#
# Requires: a reachable PostgreSQL (DATABASE_URL), Go, Node >= 22 and `playwright`
# with a Chromium browser installed (npx playwright install chromium).
set -euo pipefail
cd "$(dirname "$0")/.."

: "${DATABASE_URL:?set DATABASE_URL to a PostgreSQL database dedicated to this test}"
API_PORT="${E2E_API_PORT:-18080}"
WEB_PORT="${E2E_WEB_PORT:-19006}"
export JWT_SECRET="${JWT_SECRET:-$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')}"
export PORT="$API_PORT" ALLOWED_ORIGINS="http://localhost:$WEB_PORT"
WORK="$(mktemp -d)"
pids=()
cleanup() { for p in "${pids[@]:-}"; do kill "$p" 2>/dev/null || true; done; rm -rf "$WORK"; }
trap cleanup EXIT

echo "[e2e] building API"
(cd services/api && go build -o "$WORK/api" ./cmd/api)
"$WORK/api" >"$WORK/api.log" 2>&1 & pids+=($!)

echo "[e2e] exporting web build"
(cd apps/mobile && CI=1 EXPO_PUBLIC_API_URL="http://localhost:$API_PORT" npx expo export --clear --platform web --output-dir "$WORK/web" >/dev/null)
(cd "$WORK/web" && exec python3 -m http.server "$WEB_PORT" >/dev/null 2>&1) & pids+=($!)

for _ in $(seq 1 30); do
  curl -fs "http://localhost:$API_PORT/health" >/dev/null 2>&1 && break
  sleep 1
done
for _ in $(seq 1 30); do
  curl -fs "http://localhost:$WEB_PORT/" >/dev/null 2>&1 && break
  sleep 1
done
curl -fs "http://localhost:$API_PORT/health" >/dev/null || { echo "API did not start"; cat "$WORK/api.log"; exit 1; }

E2E_APP_URL="http://localhost:$WEB_PORT" node apps/mobile/e2e/web-flow.mjs
