param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Surface checks only. The full member journey (register -> match -> chat -> delete account)
# runs in apps/mobile/scripts/e2e-smoke.js, which smoke-all.ps1 executes next.

function Get-StatusCode([string]$Method, [string]$Path) {
  try {
    $response = Invoke-WebRequest -Method $Method -Uri "$ApiBaseUrl$Path" -UseBasicParsing
    return [int]$response.StatusCode
  }
  catch {
    if ($_.Exception.Response) { return [int]$_.Exception.Response.StatusCode }
    throw
  }
}

try {
  Write-Output "[smoke-api] API=$ApiBaseUrl"
  $health = Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl/health"
  if ($health.status -ne "ok") { throw "health check failed" }

  foreach ($protected in @("/me", "/discover", "/conversations", "/notifications", "/me/profile")) {
    $code = Get-StatusCode "GET" $protected
    if ($code -ne 401) { throw "$protected should require authentication (got $code)" }
  }

  Write-Output "[smoke-api] PASS"
  exit 0
}
catch {
  Write-Error "[smoke-api] FAIL $($_.Exception.Message)"
  exit 1
}
