#!/usr/bin/env bash
# Full local quality gate: backend (format, vet, tests, build) + mobile (typecheck, lint, format, unit tests).
# The Go integration tests need PostgreSQL (TEST_DATABASE_URL, default postgres://postgres:postgres@localhost:5432/app_test).
# Set REQUIRE_DB=1 to fail instead of skipping them when the database is unreachable.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"

echo "== backend =="
cd "$root/services/api"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "gofmt needed on:" && echo "$unformatted" && exit 1
fi
go vet ./...
go test ./...
go build -o /dev/null ./cmd/api

echo "== mobile =="
cd "$root/apps/mobile"
npm ci --no-audit --no-fund
npx tsc --noEmit
npx eslint . --max-warnings 0
npx prettier --check "src/**/*.{ts,tsx}" App.tsx
npm test

echo "All checks passed."
