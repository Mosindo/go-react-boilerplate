#!/usr/bin/env bash
# Runs the Maestro critical flow on macOS/Linux. Usage: scripts/run-maestro.sh [appId]
set -euo pipefail
cd "$(dirname "$0")/.."

APP_ID="${1:-host.exp.exponent}"
command -v maestro >/dev/null || { echo "Install the Maestro CLI first: https://maestro.mobile.dev" >&2; exit 1; }

npm run --silent e2e:ui:setup
# shellcheck disable=SC1091
source .e2e/maestro-env.sh
APP_ID="$APP_ID" maestro test e2e/maestro/critical-flow.yaml
