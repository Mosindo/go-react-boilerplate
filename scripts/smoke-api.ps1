param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Short API checklist: health, register, profile, me, account deletion.
# The full journey (match, chat, block) is covered by apps/mobile/scripts/e2e-smoke.js.

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$password = "Password123"
$email = "smoke_$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())_$(Get-Random -Maximum 100000)@smoke.test"

function Invoke-Api([string]$Method, [string]$Path, [object]$Body = $null, [string]$Token = "") {
  $headers = @{}
  if ($Token) { $headers["Authorization"] = "Bearer $Token" }
  $params = @{ Method = $Method; Uri = "$ApiBaseUrl$Path"; Headers = $headers }
  if ($null -ne $Body) {
    $params["Body"] = ($Body | ConvertTo-Json -Depth 5)
    $params["ContentType"] = "application/json"
  }
  return Invoke-RestMethod @params
}

Write-Host "[smoke-api] $ApiBaseUrl"
$health = Invoke-Api GET "/health"
if ($health.status -ne "ok") { throw "health check failed" }

$auth = Invoke-Api POST "/auth/register" @{ email = $email; password = $password }
if (-not $auth.accessToken) { throw "register did not return an access token" }

$profile = Invoke-Api PATCH "/profile" @{ firstName = "Smoke"; birthdate = "1990-01-01"; gender = "woman" } $auth.accessToken
if ($profile.age -lt 18) { throw "unexpected profile age" }

$me = Invoke-Api GET "/me" $null $auth.accessToken
if ($me.email -ne $email) { throw "unexpected /me email" }

Invoke-Api DELETE "/me" @{ password = $password } $auth.accessToken | Out-Null
Write-Host "[smoke-api] PASS"
