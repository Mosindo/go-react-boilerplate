param(
  [string]$AppId = "host.exp.exponent",
  [switch]$SkipSetup
)

# Thin Windows wrapper around the cross-platform runner (scripts/run-maestro.js).
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Push-Location $PSScriptRoot\..
try {
  if (-not $SkipSetup) {
    npm run e2e:ui:setup | Out-Host
  }
  $env:APP_ID = $AppId
  node .\scripts\run-maestro.js
  if ($LASTEXITCODE -ne 0) { throw "Maestro flow failed." }
}
finally {
  Pop-Location
}
