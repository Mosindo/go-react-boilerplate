import { defineConfig } from "@playwright/test";

const webPort = Number(process.env.WEB_PORT ?? 8081);

export default defineConfig({
  testDir: ".",
  testMatch: /.*\.spec\.ts/,
  timeout: 120_000,
  expect: { timeout: 15_000 },
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  outputDir: "./test-results",
  use: {
    baseURL: `http://localhost:${webPort}`,
    viewport: { width: 420, height: 860 },
    acceptDownloads: false,
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH } : {}
  },
  webServer: {
    command: "node scripts/serve-web.mjs",
    cwd: "..",
    url: `http://localhost:${webPort}`,
    reuseExistingServer: true
  }
});
