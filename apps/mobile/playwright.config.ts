import { defineConfig, devices } from "@playwright/test";

/**
 * UI end-to-end tests run the exported web build against a real API.
 * Prerequisites: API on E2E_API_URL (default http://localhost:18080) with
 * ALLOWED_ORIGINS including http://localhost:8081, and `npm run build:web`.
 */
export default defineConfig({
  testDir: "./e2e",
  timeout: 90_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: "http://localhost:8081",
    trace: "off",
    screenshot: "only-on-failure",
    geolocation: { latitude: 48.8566, longitude: 2.3522 },
    permissions: ["geolocation"],
    locale: "fr-FR"
  },
  projects: [{ name: "mobile-chromium", use: { ...devices["Pixel 7"] } }],
  webServer: {
    command: "node ./scripts/serve-web.js",
    url: "http://localhost:8081",
    reuseExistingServer: !process.env.CI
  }
});
