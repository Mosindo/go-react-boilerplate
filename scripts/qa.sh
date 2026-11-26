#!/usr/bin/env bash
# Runs every quality gate: backend format/vet/tests/build and mobile typecheck/lint/format/tests.
# Integration tests run when DATABASE_URL_TEST points to a disposable PostgreSQL database.
# The API smoke test runs when an API answers on API_URL (default http://localhost:18080).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
API_URL="${API_URL:-http://localhost:18080}"

step() { printf '\n\033[1;35m==> %s\033[0m\n' "$1"; }

step "Backend: gofmt"
cd "$ROOT/services/api"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then echo "$unformatted"; exit 1; fi

step "Backend: go vet"
go vet ./...

step "Backend: tests${DATABASE_URL_TEST:+ (with integration tests)}"
go test -count=1 ./...

step "Backend: build"
go build -o /dev/null ./cmd/api
go build -o /dev/null ./cmd/seed

cd "$ROOT/apps/mobile"
step "Mobile: typecheck"
npm run --silent typecheck
step "Mobile: lint"
npm run --silent lint
step "Mobile: format"
npm run --silent format:check
step "Mobile: tests"
npm test --silent -- --ci

if curl -fsS "$API_URL/health" >/dev/null 2>&1; then
  step "API smoke test against $API_URL"
  MOBILE_E2E_API_URL="$API_URL" npm run --silent e2e:smoke
else
  step "API smoke test skipped (no API on $API_URL)"
fi

printf '\n\033[1;32mAll checks passed.\033[0m\n'
