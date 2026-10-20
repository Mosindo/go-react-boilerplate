param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $repoRoot ".smoke"
$reportPath = Join-Path $reportDir "qa-lite-report.txt"
$goCacheDir = Join-Path $repoRoot ".cache\\go-build"

if (-not (Test-Path $reportDir)) {
  New-Item -ItemType Directory -Path $reportDir -Force | Out-Null
}

if (-not (Test-Path $goCacheDir)) {
  New-Item -ItemType Directory -Path $goCacheDir -Force | Out-Null
}

$env:GOCACHE = $goCacheDir

$results = New-Object System.Collections.Generic.List[string]
$failed = $false

function Run-Step([string]$name, [scriptblock]$action) {
  try {
    & $action
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
      $unformatted = @(Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -l $_.FullName } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
      if ($LASTEXITCODE -ne 0) { throw "gofmt check failed with exit code $LASTEXITCODE" }
      if ($unformatted.Count -gt 0) {
        throw ("gofmt check failed. Unformatted files:`n{0}" -f ($unformatted -join [Environment]::NewLine))
      }
    }
    finally {
      Pop-Location
    }
  }

  Run-Step "Backend tests (go test ./...)" {
    Push-Location ".\services\api"
    try {
      go test ./... | Out-Host
      if ($LASTEXITCODE -ne 0) { throw "go test exited with code $LASTEXITCODE" }
    }
    finally {
      Pop-Location
    }
  }

  Run-Step "Backend build (go build ./cmd/api)" {
    Push-Location ".\services\api"
    try {
      go build ./cmd/api
      if ($LASTEXITCODE -ne 0) { throw "go build exited with code $LASTEXITCODE" }
    }
    finally {
      Pop-Location
    }
  }

  Run-Step "Mobile TypeScript check (npx tsc --noEmit)" {
    Push-Location ".\apps\mobile"
    try {
      npx tsc --noEmit
      if ($LASTEXITCODE -ne 0) { throw "tsc exited with code $LASTEXITCODE" }
    }
    finally {
      Pop-Location
    }
  }

  Run-Step "Smoke API + mobile critique" {
    & (Join-Path $repoRoot "scripts/smoke-all.ps1") -ApiBaseUrl $ApiBaseUrl
    if ($LASTEXITCODE -ne 0) { throw "smoke-all exited with code $LASTEXITCODE" }
  }
}
finally {
  Pop-Location
}

$stamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$header = @(
  "go-react-saas QA Lite Report",
  "Generated: $stamp",
  "API Base URL: $ApiBaseUrl",
  ""
)

($header + $results) | Set-Content -Path $reportPath -Encoding UTF8
Get-Content $reportPath

if ($failed) {
  exit 1
}

exit 0
