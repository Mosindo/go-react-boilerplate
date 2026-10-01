param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# The API smoke test is a dependency-free Node script shared with Linux/macOS/CI.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $repoRoot "apps\mobile")
try {
  $env:MOBILE_E2E_API_URL = $ApiBaseUrl
  node .\scripts\e2e-smoke.js
  if ($LASTEXITCODE -ne 0) { throw "e2e-smoke exited with code $LASTEXITCODE" }
}
finally {
  Pop-Location
}
