import { defineConfig } from "@playwright/test";

/**
 * UI E2E suite running against the exported web build.
 *
 * Prerequisites (see README "Tests"):
 *   1. API running (default http://localhost:18080) with ALLOWED_ORIGINS=http://localhost:8081
 *   2. EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
 */
export default defineConfig({
  testDir: ".",
  timeout: 120_000,
  expect: { timeout: 15_000 },
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: process.env.E2E_WEB_URL ?? "http://localhost:8081",
    viewport: { width: 390, height: 844 },
    trace: "off",
    screenshot: "only-on-failure",
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH } : undefined
  },
  outputDir: "../test-results",
  webServer: process.env.E2E_WEB_URL ? undefined : { command: "node e2e/serve.js", url: "http://localhost:8081", reuseExistingServer: true, cwd: ".." }
});
