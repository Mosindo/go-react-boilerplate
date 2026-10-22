param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Health check followed by the full API smoke flow. Writes .smoke/report.txt.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $repoRoot ".smoke"
New-Item -ItemType Directory -Path $reportDir -Force | Out-Null
$reportPath = Join-Path $reportDir "report.txt"

try {
  $health = Invoke-WebRequest -UseBasicParsing "$ApiBaseUrl/health"
  if ($health.StatusCode -ne 200) { throw "/health returned $($health.StatusCode)" }
  & (Join-Path $PSScriptRoot "smoke-api.ps1") -ApiBaseUrl $ApiBaseUrl | Tee-Object -FilePath $reportPath
}
catch {
  "FAIL - $($_.Exception.Message)" | Tee-Object -FilePath $reportPath -Append
  exit 1
}
