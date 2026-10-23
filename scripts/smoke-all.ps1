param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

& (Join-Path $PSScriptRoot "smoke-api.ps1") -ApiBaseUrl $ApiBaseUrl
exit $LASTEXITCODE
