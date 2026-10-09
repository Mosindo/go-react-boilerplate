// `npm run e2e:web` : export the web build against E2E_API_URL, then run the Playwright suite.
//   node e2e/run.mjs [--build-only] [--skip-build]
import { spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';

const apiUrl = (process.env.E2E_API_URL ?? 'http://localhost:18080').replace(/\/+$/, '');
const args = new Set(process.argv.slice(2));
const env = { ...process.env, E2E_API_URL: apiUrl, EXPO_PUBLIC_API_URL: apiUrl, CI: '1' };

function run(cmd, cmdArgs) {
  const r = spawnSync(cmd, cmdArgs, { stdio: 'inherit', env, shell: process.platform === 'win32' });
  if (r.status !== 0) process.exit(r.status ?? 1);
}

if (!args.has('--skip-build') || !existsSync(new URL('../dist/index.html', import.meta.url))) {
  run('npx', ['expo', 'export', '--platform', 'web']);
}
if (!args.has('--build-only')) {
  run('npx', ['playwright', 'test', '-c', 'e2e/playwright.config.ts']);
}
