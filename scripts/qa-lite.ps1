param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# QA lite: static checks for the mobile app plus the runtime smoke journey, with a report
# written to .smoke/qa-lite-report.txt for CI to upload.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Join-Path $PSScriptRoot ".."
$reportDir = Join-Path $root ".smoke"
New-Item -ItemType Directory -Force -Path $reportDir | Out-Null
$report = Join-Path $reportDir "qa-lite-report.txt"
Set-Content -Path $report -Value "QA lite $(Get-Date -Format o)"

function Invoke-Step([string]$Name, [scriptblock]$Action) {
  Write-Output "[qa-lite] $Name"
  Add-Content -Path $report -Value "STEP $Name"
  & $Action
  if ($LASTEXITCODE -ne 0) {
    Add-Content -Path $report -Value "FAIL $Name"
    throw "qa-lite step failed: $Name"
  }
  Add-Content -Path $report -Value "PASS $Name"
}

Push-Location (Join-Path $root "apps\mobile")
try {
  Invoke-Step "typecheck" { npm run --silent typecheck }
  Invoke-Step "lint" { npm run --silent lint }
  Invoke-Step "unit tests" { npm run --silent test }
}
finally {
  Pop-Location
}

Invoke-Step "api smoke" { & (Join-Path $PSScriptRoot "smoke-api.ps1") -ApiBaseUrl $ApiBaseUrl }
Write-Output "[qa-lite] PASS (report: $report)"
