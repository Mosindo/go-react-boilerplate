import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.ts',
  timeout: 180_000,
  expect: { timeout: 15_000 },
  workers: 1,
  outputDir: process.env.PW_OUTPUT_DIR ?? '/tmp/e2e-results',
  reporter: [['list']],
  use: {
    actionTimeout: 15_000,
    baseURL: process.env.WEB_URL ?? 'http://localhost:8099',
    viewport: { width: 420, height: 860 },
    locale: 'fr-FR',
    launchOptions: process.env.CHROMIUM_PATH ? { executablePath: process.env.CHROMIUM_PATH } : {},
  },
});
