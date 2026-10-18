param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Runs the API smoke test (register -> profile -> photo -> discover -> match -> chat -> safety).
# The scenario lives in apps/mobile/scripts/e2e-smoke.js so Windows and Linux share one implementation.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$env:MOBILE_E2E_API_URL = $ApiBaseUrl
node "$PSScriptRoot\..\apps\mobile\scripts\e2e-smoke.js"
if ($LASTEXITCODE -ne 0) { throw "API smoke test failed" }
