# Runs the black-box API journey (register -> profile -> match -> chat). See apps/mobile/scripts/e2e-smoke.js.
param([string]$ApiBaseUrl = "http://localhost:18080")
$env:MOBILE_E2E_API_URL = $ApiBaseUrl
node (Join-Path $PSScriptRoot "../apps/mobile/scripts/e2e-smoke.js")
exit $LASTEXITCODE
