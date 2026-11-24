# Local quality gate (Windows / PowerShell). Mirrors scripts/qa.sh.
param(
  [string]$ApiBaseUrl = "http://localhost:18080",
  [switch]$SkipSmoke
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $repoRoot ".smoke"
$reportPath = Join-Path $reportDir "qa-lite-report.txt"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir -Force | Out-Null }

$results = New-Object System.Collections.Generic.List[string]
$failed = $false

function Run-Step([string]$name, [string]$dir, [scriptblock]$action) {
  Push-Location (Join-Path $repoRoot $dir)
  try {
    & $action
    if ($LASTEXITCODE -ne 0) { throw "exit code $LASTEXITCODE" }
    $results.Add("PASS - $name")
  }
  catch {
    $results.Add("FAIL - $name - $($_.Exception.Message)")
    $script:failed = $true
  }
  finally {
    Pop-Location
  }
}

Run-Step "gofmt" "services/api" {
  $unformatted = gofmt -l .
  if ($unformatted) { throw "unformatted files: $unformatted" }
}
Run-Step "go vet" "services/api" { go vet ./... }
Run-Step "go test" "services/api" { go test ./... }
Run-Step "go build" "services/api" { go build -o $env:TEMP/lueur-api.exe ./cmd/api }
Run-Step "npm ci" "apps/mobile" { npm ci --no-audit --no-fund }
Run-Step "typecheck" "apps/mobile" { npx tsc --noEmit }
Run-Step "lint" "apps/mobile" { npx eslint . }
Run-Step "unit tests" "apps/mobile" { npx jest --ci }
if (-not $SkipSmoke) {
  Run-Step "API smoke ($ApiBaseUrl)" "apps/mobile" {
    $env:E2E_API_URL = $ApiBaseUrl
    node ./scripts/e2e-smoke.js
  }
}

$results | Set-Content -Path $reportPath -Encoding UTF8
$results | ForEach-Object { Write-Host $_ }
Write-Host "Report: $reportPath"
if ($failed) { exit 1 }
