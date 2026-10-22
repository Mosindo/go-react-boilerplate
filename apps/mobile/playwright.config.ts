import { defineConfig } from "@playwright/test";

const webPort = Number(process.env.E2E_WEB_PORT ?? 19006);

// The API must already be running (see README "End-to-end tests") and the web bundle exported
// with `npm run build:web` using the same EXPO_PUBLIC_API_URL.
export default defineConfig({
  testDir: "./e2e/web",
  timeout: 120_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: `http://localhost:${webPort}`,
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH } : {},
    trace: "retain-on-failure",
    screenshot: "only-on-failure"
  },
  webServer: {
    command: `node e2e/web/static-server.js ${process.env.E2E_WEB_DIST ?? "dist"} ${webPort}`,
    url: `http://localhost:${webPort}`,
    reuseExistingServer: true
  }
});
