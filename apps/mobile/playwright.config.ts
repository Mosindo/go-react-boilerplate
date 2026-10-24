import { defineConfig, devices } from "@playwright/test";

const port = Number(process.env.WEB_PORT ?? 8099);

// Requires: the API running with ALLOWED_ORIGINS=http://localhost:<port>, and `npm run build:web`
// executed with EXPO_PUBLIC_API_URL pointing at that API.
export default defineConfig({
  testDir: "./e2e",
  timeout: 90_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: `http://localhost:${port}`,
    ...devices["Desktop Chrome"],
    viewport: { width: 420, height: 860 },
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH
      ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH }
      : {},
    trace: "retain-on-failure",
  },
  webServer: {
    command: "node scripts/serve-web.js",
    port,
    reuseExistingServer: true,
  },
});
