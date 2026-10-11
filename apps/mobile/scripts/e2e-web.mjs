#!/usr/bin/env node
/**
 * Runs the browser end-to-end test against an already running stack. It starts nothing itself.
 *
 *   1. API:  PORT=18090 DATABASE_URL=... JWT_SECRET=... ALLOWED_ORIGINS=http://localhost:8099 \
 *            APP_ENV=development ./api            (use a throw-away database)
 *   2. Web:  EXPO_PUBLIC_API_URL=http://localhost:18090 npx expo export --platform web
 *            then serve dist/ on WEB_URL with an SPA fallback (any static server).
 *   3. Run:  API_URL=http://localhost:18090 WEB_URL=http://localhost:8099 npm run e2e:web
 *
 * Optional: CHROMIUM_PATH (use an existing Chromium), SHOTS_DIR (default /tmp/e2e-shots).
 */
import { spawnSync } from 'node:child_process';

const missing = ['API_URL', 'WEB_URL'].filter((name) => !process.env[name]);
if (missing.length > 0) {
  console.error(`Missing required env: ${missing.join(', ')} (see header of scripts/e2e-web.mjs).`);
  process.exit(2);
}

const result = spawnSync(
  'npx',
  ['playwright', 'test', '--config', 'e2e/web/playwright.config.ts', ...process.argv.slice(2)],
  { stdio: 'inherit', shell: process.platform === 'win32' },
);
process.exit(result.status ?? 1);
