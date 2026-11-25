# Repository quality gate for Windows (PowerShell). Mirrors scripts/qa.sh.
#   powershell -ExecutionPolicy Bypass -File .\scripts\qa-lite.ps1
#   powershell -ExecutionPolicy Bypass -File .\scripts\qa-lite.ps1 -ApiBaseUrl http://localhost:18080
param(
  [string]$ApiBaseUrl = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot

function Step([string]$Name, [scriptblock]$Action) {
  Write-Host "`n==> $Name" -ForegroundColor Magenta
  & $Action
  if ($LASTEXITCODE -ne 0) { throw "$Name failed" }
}

Push-Location (Join-Path $repoRoot "services/api")
try {
  Step "API: gofmt" {
    $files = gofmt -l .
    if ($files) { Write-Host $files; $global:LASTEXITCODE = 1 } else { $global:LASTEXITCODE = 0 }
  }
  Step "API: go vet" { go vet ./... }
  Step "API: go test" { go test -p 1 ./... }
  Step "API: go build" { go build -o NUL ./cmd/api }
} finally {
  Pop-Location
}

Push-Location (Join-Path $repoRoot "apps/mobile")
try {
  Step "Mobile: typecheck" { npx tsc --noEmit }
  Step "Mobile: lint" { npx eslint . }
  Step "Mobile: format" { npx prettier --check . }
  Step "Mobile: unit tests" { npx jest --ci }
  if ($ApiBaseUrl) {
    $env:E2E_API_URL = $ApiBaseUrl
    Step "End-to-end API smoke ($ApiBaseUrl)" { node scripts/e2e-smoke.js }
  }
} finally {
  Pop-Location
}

Write-Host "`nAll checks passed." -ForegroundColor Green
