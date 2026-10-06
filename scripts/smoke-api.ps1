param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Runs the register -> profile -> discover -> match -> chat -> realtime smoke test.
$env:MOBILE_E2E_API_URL = $ApiBaseUrl
Push-Location (Join-Path (Split-Path -Parent $PSScriptRoot) "apps\mobile")
try {
  npm run e2e:smoke
  exit $LASTEXITCODE
}
finally {
  Pop-Location
}
