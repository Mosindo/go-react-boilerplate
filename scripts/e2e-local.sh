#!/usr/bin/env bash
# Runs the whole verification suite locally against a real Postgres:
# Go checks + integration tests, mobile typecheck/lint/format, API smoke test
# and the browser E2E of the critical journey.
#
#   DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable ./scripts/e2e-local.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
: "${DATABASE_URL:?set DATABASE_URL to a disposable Postgres database}"
API_PORT="${API_PORT:-18090}"
WEB_PORT="${WEB_PORT:-8091}"
WORK="$(mktemp -d)"
export JWT_SECRET="${JWT_SECRET:-$(head -c 48 /dev/urandom | base64 | tr -d '\n')}"

cleanup() {
  [[ -n "${API_PID:-}" ]] && kill "$API_PID" 2>/dev/null || true
  [[ -n "${WEB_PID:-}" ]] && kill "$WEB_PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

echo "== Go: gofmt, vet, tests, build"
cd "$ROOT/services/api"
test -z "$(gofmt -l .)" || { gofmt -l .; echo "gofmt needed"; exit 1; }
go vet ./...
DATABASE_URL_TEST="$DATABASE_URL" go test -count=1 ./...
go build -o "$WORK/api" ./cmd/api

echo "== Mobile: typecheck, lint, format"
cd "$ROOT/apps/mobile"
npm run --silent check

echo "== Start API on :$API_PORT"
APP_ENV=test PORT="$API_PORT" UPLOAD_DIR="$WORK/uploads" ALLOWED_ORIGINS="http://localhost:$WEB_PORT" \
  "$WORK/api" >"$WORK/api.log" 2>&1 &
API_PID=$!
for _ in $(seq 1 30); do curl -fsS "http://localhost:$API_PORT/health" >/dev/null 2>&1 && break; sleep 1; done

echo "== Build and serve web app on :$WEB_PORT"
EXPO_PUBLIC_API_URL="http://localhost:$API_PORT" npx expo export --clear --platform web --output-dir "$WORK/web" >/dev/null
node e2e/serve.mjs "$WORK/web" "$WEB_PORT" >"$WORK/web.log" 2>&1 &
WEB_PID=$!
sleep 1

echo "== API smoke test"
E2E_API_URL="http://localhost:$API_PORT" node e2e/api-smoke.mjs

echo "== Browser E2E (critical journey)"
E2E_API_URL="http://localhost:$API_PORT" E2E_WEB_URL="http://localhost:$WEB_PORT" node e2e/critical-flow.mjs

echo "All checks passed."
