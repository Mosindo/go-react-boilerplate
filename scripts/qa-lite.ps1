param([string]$ApiBaseUrl = "http://localhost:18080")
# Thin wrapper: the end-to-end API smoke test is a cross-platform Node script.
$env:API_BASE_URL = $ApiBaseUrl
Push-Location (Join-Path $PSScriptRoot "../apps/mobile")
try { node ./scripts/e2e-smoke.js; exit $LASTEXITCODE } finally { Pop-Location }
