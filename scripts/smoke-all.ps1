param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Checks the runtime prerequisites called out in AGENTS.md (containers up, API reachable
# from the network, port not hijacked), then runs the API smoke journey.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Output "[smoke-all] docker compose ps"
docker compose ps

Write-Output "[smoke-all] GET $ApiBaseUrl/health"
$health = Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl/health"
if ($health.status -ne "ok") {
  throw "health check failed: $($health | ConvertTo-Json -Compress)"
}

& (Join-Path $PSScriptRoot "smoke-api.ps1") -ApiBaseUrl $ApiBaseUrl
Write-Output "[smoke-all] PASS"
