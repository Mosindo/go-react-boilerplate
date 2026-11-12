#!/usr/bin/env bash
# Full local quality gate. Mirrors CI.
#   ./scripts/qa.sh                                  # static checks, tests, builds
#   API_URL=http://localhost:18080 ./scripts/qa.sh   # + live API smoke test
# Go integration tests run when DATABASE_URL (or DATABASE_URL_TEST) points to Postgres; otherwise they are skipped.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"

step() { printf '\n\033[1;33m==> %s\033[0m\n' "$1"; }

step "Go: gofmt"
cd "$root/services/api"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then echo "Unformatted Go files:"; echo "$unformatted"; exit 1; fi

step "Go: vet"
go vet ./...

step "Go: tests (-p 1: integration packages share one database)"
go test -p 1 -count=1 ./...

step "Go: build"
go build -o /dev/null ./cmd/api
go build -o /dev/null ./cmd/seed

cd "$root/apps/mobile"
step "Mobile: install"
npm ci --no-audit --no-fund

step "Mobile: typecheck"
npm run typecheck

step "Mobile: lint"
npm run lint

step "Mobile: format"
npm run format:check

step "Mobile: unit tests"
npm test

step "Mobile: web build"
npm run build:web >/dev/null

if [ -n "${API_URL:-}" ]; then
  step "Live API smoke test ($API_URL)"
  E2E_API_URL="$API_URL" npm run e2e:api
fi

printf '\n\033[1;32mAll checks passed.\033[0m\n'
