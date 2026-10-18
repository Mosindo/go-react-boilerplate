import { defineConfig } from "@playwright/test";

// The suite is started by scripts/e2e-web.js, which boots the real API, a
// database and the exported web build, then exposes their URLs through env vars.
export default defineConfig({
  testDir: "./e2e/web",
  timeout: 120_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: process.env.E2E_WEB_URL ?? "http://localhost:18091",
    viewport: { width: 420, height: 880 },
    locale: "fr-FR",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH } : {}
  },
  outputDir: "./.e2e/results"
});
