param(
  [string]$ApiBaseUrl = "http://localhost:18080"
)

# Full smoke run: API journey. UI flow (Maestro) is run separately with
# `npm run e2e:ui:run` in apps/mobile because it needs a device or emulator.
& (Join-Path $PSScriptRoot "smoke-api.ps1") -ApiBaseUrl $ApiBaseUrl
exit $LASTEXITCODE
