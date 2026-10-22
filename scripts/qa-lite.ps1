param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Local quality gate: formatting, vet, build, tests, typecheck, lint, then the API smoke flow.
# Backend integration tests need DATABASE_URL_TEST (an empty throwaway database).
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $repoRoot ".smoke"
New-Item -ItemType Directory -Path $reportDir -Force | Out-Null
$reportPath = Join-Path $reportDir "qa-lite-report.txt"
$results = New-Object System.Collections.Generic.List[string]
$failed = $false

function Run-Step([string]$name, [scriptblock]$action) {
  try {
    & $action
    if ($LASTEXITCODE -ne 0) { throw "exit code $LASTEXITCODE" }
    $results.Add("PASS - $name")
  }
  catch {
    $results.Add("FAIL - $name - $($_.Exception.Message)")
    $script:failed = $true
  }
}

Push-Location $repoRoot
try {
  Run-Step "gofmt" {
    Push-Location ".\services\api"
    try {
      $unformatted = gofmt -l .
      if ($unformatted) { throw "unformatted files:`n$unformatted" }
      $global:LASTEXITCODE = 0
    }
    finally { Pop-Location }
  }
  Run-Step "go vet" { Push-Location ".\services\api"; try { go vet ./... } finally { Pop-Location } }
  Run-Step "go build" { Push-Location ".\services\api"; try { go build ./cmd/api } finally { Pop-Location } }
  Run-Step "go test" { Push-Location ".\services\api"; try { go test ./... } finally { Pop-Location } }
  Run-Step "mobile typecheck" { Push-Location ".\apps\mobile"; try { npx tsc --noEmit } finally { Pop-Location } }
  Run-Step "mobile lint" { Push-Location ".\apps\mobile"; try { npm run lint } finally { Pop-Location } }
  Run-Step "mobile unit tests" { Push-Location ".\apps\mobile"; try { npm test } finally { Pop-Location } }
  Run-Step "API smoke" { & (Join-Path $PSScriptRoot "smoke-api.ps1") -ApiBaseUrl $ApiBaseUrl }
}
finally {
  Pop-Location
  $results | Set-Content -Path $reportPath
  $results | ForEach-Object { Write-Host $_ }
}

if ($failed) { exit 1 }
