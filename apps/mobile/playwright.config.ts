import { defineConfig, devices } from "@playwright/test";

// Web E2E of the critical journey. Requires the API running at E2E_API_URL
// with ALLOWED_ORIGINS including the web origin, and a web export in dist/
// built with EXPO_PUBLIC_API_URL=E2E_API_URL (see README "Tests").
export default defineConfig({
  testDir: "./e2e/web",
  timeout: 90_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  reporter: [["list"]],
  outputDir: "./e2e/web/test-results",
  use: {
    baseURL: process.env.E2E_WEB_URL ?? "http://localhost:8099",
    ...devices["Pixel 7"],
    trace: "retain-on-failure",
    screenshot: "only-on-failure"
  },
  webServer: {
    command: "node ./scripts/serve-dist.js dist",
    url: process.env.E2E_WEB_URL ?? "http://localhost:8099",
    reuseExistingServer: true
  }
});
