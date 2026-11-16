#!/usr/bin/env bash
# Full local quality gate: formatting, static checks, unit + integration tests,
# builds, and (when an API is reachable) the API smoke test.
#
#   DATABASE_URL_TEST=postgresql://app:app@localhost:5432/app_test?sslmode=disable ./scripts/qa.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_URL="${MOBILE_E2E_API_URL:-http://localhost:18080}"

step() { printf '\n\033[1;33m==> %s\033[0m\n' "$1"; }

step "API: gofmt"
cd "$ROOT/services/api"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "gofmt needed on:"; echo "$unformatted"; exit 1
fi

step "API: go vet"
go vet ./...

step "API: go test (integration tests run when DATABASE_URL_TEST is set)"
go test ./...

step "API: go build"
go build -o /dev/null ./cmd/api
go build -o /dev/null ./cmd/seed

step "Mobile: npm ci"
cd "$ROOT/apps/mobile"
npm ci --no-audit --no-fund

step "Mobile: typecheck, lint, format"
npm run -s typecheck
npm run -s lint
npm run -s format:check

if curl -fsS "$API_URL/health" >/dev/null 2>&1; then
  step "API smoke against $API_URL"
  MOBILE_E2E_API_URL="$API_URL" npm run -s e2e:smoke
else
  echo "(skipping API smoke: $API_URL/health not reachable)"
fi

printf '\n\033[1;32mQA passed.\033[0m\n'
