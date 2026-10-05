import { defineConfig } from "@playwright/test";

// The suite drives the exported web build against a real API + database. Start both first:
//   (services/api)  go run ./cmd/api            with ALLOWED_ORIGINS=http://localhost:8081
//   (apps/mobile)   npm run build:web && npm run serve:web
export default defineConfig({
  testDir: "./e2e/web",
  timeout: 120_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: process.env.E2E_WEB_URL ?? "http://localhost:8081",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH } : {}
  }
});
