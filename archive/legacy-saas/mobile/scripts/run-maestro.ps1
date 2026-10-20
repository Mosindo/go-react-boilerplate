param(
  [string]$AppId = "host.exp.exponent",
  [switch]$SkipSetup
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Require-Command {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$Hint
  )

  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Missing command '$Name'. $Hint"
  }
}

function Require-Env {
  param(
    [Parameter(Mandatory = $true)][string]$Name
  )

  $value = (Get-Item -Path "Env:$Name" -ErrorAction SilentlyContinue).Value
  if ([string]::IsNullOrWhiteSpace($value)) {
    throw "Missing environment variable '$Name'. Run setup first."
  }
}

Push-Location $PSScriptRoot\..
try {
  if (-not $SkipSetup) {
    Write-Host "[maestro-run] Running setup fixtures..."
    npm run e2e:ui:setup | Out-Host
  }

  $envFile = ".\.e2e\maestro-env.ps1"
  if (-not (Test-Path $envFile)) {
    throw "Missing $envFile. Run npm run e2e:ui:setup."
  }

  . $envFile

  Require-Env -Name "TEST_EMAIL"
  Require-Env -Name "TEST_PASSWORD"
  Require-Env -Name "CONTACT_EMAIL"
  Require-Env -Name "NOTIFICATION_TITLE"

  Require-Command -Name "maestro" -Hint "Install Maestro CLI and ensure it is in PATH."

  if (-not (Get-Command adb -ErrorAction SilentlyContinue)) {
    Write-Warning "adb not found in PATH. If using a physical device, ensure Android platform-tools are installed."
  }

  $env:APP_ID = $AppId
  Write-Host "[maestro-run] Running flow with APP_ID=$AppId"
  maestro test .\e2e\maestro\critical-flow.yaml
}
finally {
  Pop-Location
}
