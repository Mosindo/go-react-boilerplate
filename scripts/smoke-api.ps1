param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# The critical-path API smoke test is a Node script shared with CI (no dependency).
$repoRoot = Split-Path -Parent $PSScriptRoot
$env:MOBILE_E2E_API_URL = $ApiBaseUrl
node (Join-Path $repoRoot "apps/mobile/scripts/e2e-smoke.js")
if ($LASTEXITCODE -ne 0) {
  Write-Output "[smoke-api] FAIL"
  exit 1
}
Write-Output "[smoke-api] PASS"
exit 0
