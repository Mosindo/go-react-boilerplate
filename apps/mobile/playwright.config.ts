import { defineConfig } from "@playwright/test";

// Runs against a real API (EXPO_PUBLIC_API_URL baked into the web export) and the exported web build.
export default defineConfig({
  testDir: "./e2e",
  timeout: 120_000,
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:8081",
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH
      ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH }
      : {},
    trace: "retain-on-failure",
    viewport: { width: 420, height: 860 },
  },
  webServer: {
    command: "node e2e/serve.mjs",
    url: "http://localhost:8081",
    reuseExistingServer: true,
  },
});
