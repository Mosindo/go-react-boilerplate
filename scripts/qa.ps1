# Windows equivalent of scripts/qa.sh: runs every backend and mobile quality gate.
# Integration tests run when $env:DATABASE_URL_TEST points to a disposable PostgreSQL database.
param(
  [string]$ApiUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

function Step([string]$name, [scriptblock]$action) {
  Write-Host "`n==> $name" -ForegroundColor Magenta
  & $action
  if ($LASTEXITCODE -ne 0) { throw "FAILED: $name" }
}

Push-Location (Join-Path $root "services/api")
try {
  Step "Backend: gofmt" {
    $unformatted = gofmt -l .
    if ($unformatted) { Write-Host $unformatted; $global:LASTEXITCODE = 1 } else { $global:LASTEXITCODE = 0 }
  }
  Step "Backend: go vet" { go vet ./... }
  Step "Backend: tests" { go test -count=1 ./... }
  Step "Backend: build" { go build -o $null ./cmd/api; if ($LASTEXITCODE -eq 0) { go build -o $null ./cmd/seed } }
}
finally { Pop-Location }

Push-Location (Join-Path $root "apps/mobile")
try {
  Step "Mobile: typecheck" { npm run --silent typecheck }
  Step "Mobile: lint" { npm run --silent lint }
  Step "Mobile: format" { npm run --silent format:check }
  Step "Mobile: tests" { npm test --silent -- --ci }

  $apiUp = $false
  try { $apiUp = (Invoke-WebRequest -UseBasicParsing "$ApiUrl/health" -TimeoutSec 3).StatusCode -eq 200 } catch { $apiUp = $false }
  if ($apiUp) {
    $env:MOBILE_E2E_API_URL = $ApiUrl
    Step "API smoke test against $ApiUrl" { npm run --silent e2e:smoke }
  }
  else {
    Write-Host "`nAPI smoke test skipped (no API on $ApiUrl)" -ForegroundColor Yellow
  }
}
finally { Pop-Location }

Write-Host "`nAll checks passed." -ForegroundColor Green
