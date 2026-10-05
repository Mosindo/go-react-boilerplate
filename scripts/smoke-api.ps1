param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# The API smoke test lives in one place (Node, shared with CI): it registers users, completes onboarding,
# likes, matches, chats, blocks and deletes the accounts again.
Push-Location (Join-Path (Split-Path -Parent $PSScriptRoot) "apps/mobile")
try {
  $env:MOBILE_E2E_API_URL = $ApiBaseUrl
  node ./scripts/e2e-smoke.js
  if ($LASTEXITCODE -ne 0) { throw "e2e-smoke exited with code $LASTEXITCODE" }
}
finally {
  Remove-Item Env:\MOBILE_E2E_API_URL -ErrorAction SilentlyContinue
  Pop-Location
}
