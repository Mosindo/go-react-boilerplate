import { existsSync } from 'node:fs';
import { chromium, defineConfig } from '@playwright/test';

const WEB_PORT = Number(process.env.E2E_WEB_PORT ?? 19081);

/** Use Playwright's own Chromium when installed, otherwise the preinstalled one. */
function executablePath(): string | undefined {
  try {
    if (existsSync(chromium.executablePath())) return undefined;
  } catch {
    // fall through
  }
  const fallback = process.env.PW_CHROMIUM_PATH ?? '/opt/pw-browsers/chromium';
  return existsSync(fallback) ? fallback : undefined;
}

export default defineConfig({
  testDir: '.',
  testMatch: /.*\.spec\.ts/,
  timeout: 240_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  outputDir: '../test-results',
  use: {
    baseURL: `http://localhost:${WEB_PORT}`,
    viewport: { width: 430, height: 900 },
    locale: 'fr-FR',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    launchOptions: { executablePath: executablePath(), args: ['--no-sandbox'] },
  },
  webServer: {
    command: 'node serve.mjs',
    url: `http://localhost:${WEB_PORT}`,
    reuseExistingServer: true,
    timeout: 30_000,
    env: { E2E_WEB_PORT: String(WEB_PORT) },
  },
});
