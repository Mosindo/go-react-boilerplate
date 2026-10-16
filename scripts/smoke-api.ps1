param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# API smoke test: the full product journey (signup, profile, photos, discover, like, match,
# chat, notifications, block, account deletion). The scenario lives in Node so Windows,
# macOS, Linux and CI all run the exact same checks.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$env:MOBILE_E2E_API_URL = $ApiBaseUrl
Write-Output "[smoke-api] API=$ApiBaseUrl"
node (Join-Path $PSScriptRoot "..\apps\mobile\scripts\e2e-smoke.js")
if ($LASTEXITCODE -ne 0) {
  throw "smoke-api failed"
}
Write-Output "[smoke-api] PASS"
