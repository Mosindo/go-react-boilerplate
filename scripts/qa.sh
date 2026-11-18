#!/usr/bin/env bash
# Repository quality gate: formatting, static checks, tests and builds.
# Optional end-to-end suites run when E2E_API_URL points to a live API:
#   E2E_API_URL=http://localhost:18080 ./scripts/qa.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
step() { printf '\n==> %s\n' "$1"; }

cd "$ROOT/services/api"
step "gofmt"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "Unformatted Go files:"
  echo "$unformatted"
  exit 1
fi
step "go vet"
go vet ./...
step "go test (integration tests need DATABASE_URL)"
go test ./...
step "go build"
go build -o /dev/null ./cmd/api
go build -o /dev/null ./cmd/seed

cd "$ROOT/apps/mobile"
if [ ! -d node_modules ]; then
  step "npm ci"
  npm ci
fi
step "typecheck"
npm run typecheck
step "lint"
npm run lint
step "format"
npm run format:check

if [ -n "${E2E_API_URL:-}" ]; then
  step "API end-to-end smoke"
  npm run e2e:api
  step "web build + UI end-to-end"
  EXPO_PUBLIC_API_URL="$E2E_API_URL" npm run build:web
  npm run e2e:web
fi

printf '\nQA passed.\n'
