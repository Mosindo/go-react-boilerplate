#!/usr/bin/env bash
# Repository quality gate (Linux/macOS). Mirrors CI.
#   ./scripts/qa.sh                       # static checks + unit tests
#   API_URL=http://localhost:18080 ./scripts/qa.sh   # + API end-to-end smoke
# Set DATABASE_URL_TEST to a disposable database to run the API integration tests.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"

step() { printf '\n\033[1;35m==> %s\033[0m\n' "$1"; }

step "API: gofmt"
cd "$root/services/api"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then echo "Unformatted files:"; echo "$unformatted"; exit 1; fi
step "API: go vet"
go vet ./...
step "API: go test"
go test -p 1 ./...
step "API: go build"
go build -o /dev/null ./cmd/api
go build -o /dev/null ./cmd/seed

cd "$root/apps/mobile"
step "Mobile: typecheck"
npx tsc --noEmit
step "Mobile: lint"
npx eslint .
step "Mobile: format"
npx prettier --check .
step "Mobile: unit tests"
npx jest --ci

if [ -n "${API_URL:-}" ]; then
  step "End-to-end API smoke ($API_URL)"
  E2E_API_URL="$API_URL" node scripts/e2e-smoke.js
fi

printf '\n\033[1;32mAll checks passed.\033[0m\n'
