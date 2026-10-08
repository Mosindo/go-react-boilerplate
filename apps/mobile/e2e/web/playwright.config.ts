import { defineConfig, devices } from "@playwright/test";

/**
 * Browser E2E against the Expo web export. Prerequisites (see README "Tests"):
 *   - the API running at E2E_API_URL (default http://localhost:18080) with ALLOWED_ORIGINS including the web origin
 *   - the app exported with EXPO_PUBLIC_API_URL pointing to that API (`npm run web:export`)
 * The static server for the export is started automatically.
 */
const webUrl = process.env.E2E_WEB_URL ?? "http://localhost:8081";

export default defineConfig({
  testDir: ".",
  timeout: 120_000,
  expect: { timeout: 15_000 },
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: webUrl,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    ...devices["Pixel 7"],
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE } : {}
  },
  webServer: {
    command: "node ../../scripts/serve-web.js ../../dist",
    url: webUrl,
    reuseExistingServer: true,
    env: { PORT: new URL(webUrl).port || "8081" }
  }
});
