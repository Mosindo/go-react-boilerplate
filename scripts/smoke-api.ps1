param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# API smoke test: register, login, profile + photo, discover, like/match, chat send/read, notifications.
# The flow itself lives in apps/mobile/scripts/e2e-smoke.js so it runs the same on every OS.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$env:MOBILE_E2E_API_URL = $ApiBaseUrl
Push-Location (Join-Path $repoRoot "apps\mobile")
try {
  node .\scripts\e2e-smoke.js
  if ($LASTEXITCODE -ne 0) { throw "API smoke test failed" }
}
finally {
  Pop-Location
}
