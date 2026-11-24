#!/usr/bin/env bash
# Local quality gate (Linux / macOS). Usage: scripts/qa.sh [--skip-smoke]
# Integration tests run when DATABASE_URL_TEST (or DATABASE_URL) is set.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
API_URL="${E2E_API_URL:-http://localhost:18080}"

step() { printf '\n==> %s\n' "$1"; }

cd "$ROOT/services/api"
step "gofmt"; test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; }
step "go vet"; go vet ./...
step "go test"; go test ./...
step "go build"; go build -o "${TMPDIR:-/tmp}/lueur-api" ./cmd/api

cd "$ROOT/apps/mobile"
step "npm ci"; npm ci --no-audit --no-fund
step "typecheck"; npx tsc --noEmit
step "lint"; npx eslint .
step "unit tests"; npx jest --ci

if [[ "${1:-}" != "--skip-smoke" ]]; then
  step "API smoke ($API_URL)"; E2E_API_URL="$API_URL" node ./scripts/e2e-smoke.js
fi
printf '\nAll checks passed.\n'
