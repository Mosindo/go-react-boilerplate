param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# API smoke test of the whole dating journey (sign up, onboarding, discover, like, match,
# chat, notifications, block/report, account deletion). The scenario lives in
# apps/mobile/scripts/e2e-smoke.js so Windows, macOS and Linux share one implementation.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$previous = $env:MOBILE_E2E_API_URL
$env:MOBILE_E2E_API_URL = $ApiBaseUrl
try {
  node (Join-Path $repoRoot "apps/mobile/scripts/e2e-smoke.js")
  if ($LASTEXITCODE -ne 0) { throw "API smoke failed with exit code $LASTEXITCODE" }
}
finally {
  if ($null -eq $previous) { Remove-Item Env:\MOBILE_E2E_API_URL -ErrorAction SilentlyContinue } else { $env:MOBILE_E2E_API_URL = $previous }
}
