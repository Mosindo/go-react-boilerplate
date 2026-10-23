param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Runs the real critical flow (register -> profile -> photo -> discover -> match -> chat -> realtime -> block -> delete)
# against a running API. The flow itself lives in apps/mobile/scripts/e2e-smoke.js so Linux, macOS and CI share it.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$env:API_URL = $ApiBaseUrl
node (Join-Path $repoRoot "apps/mobile/scripts/e2e-smoke.js")
if ($LASTEXITCODE -ne 0) { throw "e2e-smoke failed with exit code $LASTEXITCODE" }
