param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Quality gate for Windows machines: same checks as scripts/check.sh, followed by
# the end-to-end API smoke test against a running API (docker compose up -d).

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $repoRoot ".smoke"
$reportPath = Join-Path $reportDir "qa-lite-report.txt"
New-Item -ItemType Directory -Path $reportDir -Force | Out-Null

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
  Run-Step "Backend gofmt" {
    Push-Location ".\services\api"
    try {
      $unformatted = gofmt -l .
      if ($unformatted) { throw "gofmt needed on: $unformatted" }
    } finally { Pop-Location }
  }
  Run-Step "Backend go vet" { Push-Location ".\services\api"; try { go vet ./... } finally { Pop-Location } }
  Run-Step "Backend go test" { Push-Location ".\services\api"; try { go test ./... } finally { Pop-Location } }
  Run-Step "Backend go build" { Push-Location ".\services\api"; try { go build -o NUL ./cmd/api } finally { Pop-Location } }
  Run-Step "Mobile npm ci" { Push-Location ".\apps\mobile"; try { npm ci } finally { Pop-Location } }
  Run-Step "Mobile typecheck" { Push-Location ".\apps\mobile"; try { npx tsc --noEmit } finally { Pop-Location } }
  Run-Step "Mobile lint" { Push-Location ".\apps\mobile"; try { npx eslint . --max-warnings 0 } finally { Pop-Location } }
  Run-Step "Mobile unit tests" { Push-Location ".\apps\mobile"; try { npm test } finally { Pop-Location } }
  Run-Step "API smoke (register -> match -> chat)" {
    Push-Location ".\apps\mobile"
    try { $env:MOBILE_E2E_API_URL = $ApiBaseUrl; npm run e2e:smoke } finally { Pop-Location }
  }
}
finally {
  Pop-Location
}

$results | Tee-Object -FilePath $reportPath
if ($failed) { exit 1 }
