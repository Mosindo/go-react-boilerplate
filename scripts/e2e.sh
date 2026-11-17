#!/usr/bin/env bash
# Runs the browser end-to-end test against a throw-away stack:
#   fresh PostgreSQL database -> API -> exported web app -> Playwright (two real users).
#
# Requirements: Go, Node, Python 3 (static file server), a reachable PostgreSQL server
# with CREATE DATABASE rights, and the `playwright` npm package with Chromium
# (set CHROMIUM_PATH to use an already installed browser).
#
# Environment:
#   E2E_PG_URL     base connection URL without database (default postgres://postgres@localhost:5432)
#   E2E_API_PORT   default 18090
#   E2E_WEB_PORT   default 8091
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PG_URL="${E2E_PG_URL:-postgres://postgres@localhost:5432}"
API_PORT="${E2E_API_PORT:-18090}"
WEB_PORT="${E2E_WEB_PORT:-8091}"
DB_NAME="lueur_e2e_$(date +%s)"
WORK="$(mktemp -d)"
API_PID=""
WEB_PID=""

cleanup() {
  [[ -n "$API_PID" ]] && kill "$API_PID" 2>/dev/null || true
  [[ -n "$WEB_PID" ]] && kill "$WEB_PID" 2>/dev/null || true
  psql "$PG_URL/postgres" -qc "DROP DATABASE IF EXISTS $DB_NAME" >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

echo "› creating database $DB_NAME"
psql "$PG_URL/postgres" -qc "CREATE DATABASE $DB_NAME"

echo "› building API"
(cd "$ROOT/services/api" && go build -o "$WORK/api" ./cmd/api)
DATABASE_URL="$PG_URL/$DB_NAME?sslmode=disable" \
JWT_SECRET="$(head -c 32 /dev/urandom | base64)" \
UPLOAD_DIR="$WORK/uploads" \
PORT="$API_PORT" \
APP_ENV=test \
ALLOWED_ORIGINS="http://localhost:$WEB_PORT" \
  "$WORK/api" >"$WORK/api.log" 2>&1 &
API_PID=$!

for _ in $(seq 1 30); do
  curl -fsS "http://localhost:$API_PORT/health" >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS "http://localhost:$API_PORT/health" >/dev/null || { cat "$WORK/api.log"; exit 1; }

echo "› exporting web app"
(cd "$ROOT/apps/mobile" && EXPO_PUBLIC_API_URL="http://localhost:$API_PORT" npx expo export --clear --platform web --output-dir "$WORK/web" >/dev/null 2>&1)
python3 -m http.server "$WEB_PORT" --directory "$WORK/web" >/dev/null 2>&1 &
WEB_PID=$!
sleep 1

echo "› running Playwright scenario"
(cd "$ROOT/apps/mobile" && E2E_WEB_URL="http://localhost:$WEB_PORT" E2E_API_URL="http://localhost:$API_PORT" node e2e/run.mjs) || {
  echo "--- API log (errors) ---"
  grep -v '"status":2' "$WORK/api.log" | grep -v GIN-debug | tail -40
  exit 1
}
