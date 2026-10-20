param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$password = "Password123"

function New-UniqueEmail([string]$prefix) {
  $stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
  $rand = Get-Random -Maximum 100000
  return "${prefix}_${stamp}_${rand}@go-react-saas.test"
}

function Invoke-Api([string]$Method, [string]$Path, [object]$Body = $null, [hashtable]$Headers = @{}) {
  $uri = "$ApiBaseUrl$Path"
  if ($null -eq $Body) {
    return Invoke-RestMethod -Method $Method -Uri $uri -Headers $Headers
  }
  return Invoke-RestMethod -Method $Method -Uri $uri -Headers $Headers -ContentType "application/json" -Body ($Body | ConvertTo-Json -Depth 5)
}

try {
  Write-Output "[smoke-api] API=$ApiBaseUrl"
  $health = Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl/health"
  if ($health.status -ne "ok") {
    throw "health check failed"
  }

  $email1 = New-UniqueEmail "smoke_api_a"
  $email2 = New-UniqueEmail "smoke_api_b"

  $auth1 = Invoke-Api "Post" "/auth/register" @{ email = $email1; password = $password }
  $auth2 = Invoke-Api "Post" "/auth/register" @{ email = $email2; password = $password }

  $login1 = Invoke-Api "Post" "/auth/login" @{ email = $email1; password = $password }
  if ($login1.user.id -ne $auth1.user.id) {
    throw "login user mismatch"
  }

  $headers1 = @{ Authorization = "Bearer $($auth1.token)" }

  $users = Invoke-Api "Get" "/users" $null $headers1
  if (-not ($users.users | Where-Object { $_.id -eq $auth2.user.id })) {
    throw "users list should include second user"
  }

  $post = Invoke-Api "Post" "/posts" @{ title = "Smoke post"; body = "Validating the generic post feed." } $headers1
  if ([string]::IsNullOrWhiteSpace($post.id)) {
    throw "post create failed"
  }

  $posts = Invoke-Api "Get" "/posts" $null $headers1
  if (-not ($posts.posts | Where-Object { $_.id -eq $post.id })) {
    throw "created post missing from list"
  }

  $msg = Invoke-Api "Post" "/chats/$($auth2.user.id)/messages" @{ content = "hello from smoke-api" } $headers1
  if ([string]::IsNullOrWhiteSpace($msg.id)) {
    throw "message send failed"
  }

  $messages = Invoke-Api "Get" "/chats/$($auth2.user.id)/messages" $null $headers1
  if (-not $messages.messages -or $messages.messages.Count -lt 1) {
    throw "message list empty"
  }

  $notification = Invoke-Api "Post" "/notifications" @{
    type  = "system"
    title = "Smoke notification"
    body  = "Validating the generic notification flow."
  } $headers1
  if ([string]::IsNullOrWhiteSpace($notification.id)) {
    throw "notification create failed"
  }

  $notifications = Invoke-Api "Get" "/notifications" $null $headers1
  if (-not ($notifications.notifications | Where-Object { $_.id -eq $notification.id })) {
    throw "notification missing from list"
  }

  $readNotification = Invoke-Api "Post" "/notifications/$($notification.id)/read" $null $headers1
  if ($readNotification.isRead -ne $true) {
    throw "notification should be marked as read"
  }

  Write-Output "[smoke-api] PASS"
  exit 0
}
catch {
  Write-Output "[smoke-api] FAIL $($_.Exception.Message)"
  exit 1
}
