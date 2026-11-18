# Windows equivalent of scripts/qa.sh.
#   powershell -ExecutionPolicy Bypass -File .\scripts\qa.ps1
#   $env:E2E_API_URL = "http://localhost:18080"; .\scripts\qa.ps1   # with end-to-end suites
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

function Invoke-Step([string]$Name, [scriptblock]$Action) {
  Write-Host "`n==> $Name"
  $global:LASTEXITCODE = 0
  & $Action
  if ($LASTEXITCODE -ne 0) { throw "Step failed: $Name" }
}

Push-Location (Join-Path $root "services/api")
try {
  Write-Host "`n==> gofmt"
  $unformatted = gofmt -l .
  if ($unformatted) { Write-Host $unformatted; throw "Unformatted Go files" }
  Invoke-Step "go vet" { go vet ./... }
  Invoke-Step "go test" { go test ./... }
  Invoke-Step "go build api" { go build -o "$env:TEMP\lueur-api.exe" ./cmd/api }
  Invoke-Step "go build seed" { go build -o "$env:TEMP\lueur-seed.exe" ./cmd/seed }
} finally { Pop-Location }

Push-Location (Join-Path $root "apps/mobile")
try {
  if (-not (Test-Path node_modules)) { Invoke-Step "npm ci" { npm ci } }
  Invoke-Step "typecheck" { npm run typecheck }
  Invoke-Step "lint" { npm run lint }
  Invoke-Step "format" { npm run format:check }
  if ($env:E2E_API_URL) {
    Invoke-Step "API end-to-end smoke" { npm run e2e:api }
    $env:EXPO_PUBLIC_API_URL = $env:E2E_API_URL
    Invoke-Step "web build" { npm run build:web }
    Invoke-Step "UI end-to-end" { npm run e2e:web }
  }
} finally { Pop-Location }

Write-Host "`nQA passed."
