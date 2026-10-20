param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $repoRoot ".smoke"
$reportPath = Join-Path $reportDir "report.txt"

if (-not (Test-Path $reportDir)) {
  New-Item -ItemType Directory -Path $reportDir -Force | Out-Null
}

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
  Run-Step "Smoke API automatise (checklist MVP courte)" {
    & (Join-Path $repoRoot "scripts/smoke-api.ps1") -ApiBaseUrl $ApiBaseUrl
    if ($LASTEXITCODE -ne 0) { throw "smoke-api exited with code $LASTEXITCODE" }
  }

  Run-Step "Smoke mobile e2e (parcours critique)" {
    Push-Location ".\apps\mobile"
    try {
      $previousMobileApi = $env:MOBILE_E2E_API_URL
      $previousExpoApi = $env:EXPO_PUBLIC_API_URL
      $env:MOBILE_E2E_API_URL = $ApiBaseUrl
      $env:EXPO_PUBLIC_API_URL = $ApiBaseUrl

      npm run e2e:smoke | Out-Host
      if ($LASTEXITCODE -ne 0) { throw "npm run e2e:smoke exited with code $LASTEXITCODE" }
    }
    finally {
      if ($null -eq $previousMobileApi) {
        Remove-Item Env:\MOBILE_E2E_API_URL -ErrorAction SilentlyContinue
      }
      else {
        $env:MOBILE_E2E_API_URL = $previousMobileApi
      }

      if ($null -eq $previousExpoApi) {
        Remove-Item Env:\EXPO_PUBLIC_API_URL -ErrorAction SilentlyContinue
      }
      else {
        $env:EXPO_PUBLIC_API_URL = $previousExpoApi
      }

      Pop-Location
    }
  }
}
finally {
  Pop-Location
}

$stamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$header = @(
  "go-react-saas Smoke Report",
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
