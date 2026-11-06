import { defineConfig } from "@playwright/test";

// End-to-end tests drive the exported web build against a real API + PostgreSQL.
//   E2E_DATABASE_URL=postgres://user:pass@localhost:5432/e2e npm run build:web && npm run test:e2e
const dbUrl = process.env.E2E_DATABASE_URL ?? "postgres://postgres@localhost:5433/e2e?sslmode=disable";
const apiPort = process.env.E2E_API_PORT ?? "18080";
const webPort = process.env.WEB_PORT ?? "8081";

export default defineConfig({
  testDir: "./e2e",
  testMatch: /.*\.spec\.ts/,
  timeout: 120_000,
  expect: { timeout: 15_000 },
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: `http://localhost:${webPort}`,
    trace: "off",
    launchOptions: process.env.PW_CHROMIUM_PATH ? { executablePath: process.env.PW_CHROMIUM_PATH } : {}
  },
  webServer: [
    {
      command: "go run ./cmd/api",
      cwd: "../../services/api",
      url: `http://localhost:${apiPort}/health`,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        PORT: apiPort,
        DATABASE_URL: dbUrl,
        JWT_SECRET: "e2e-jwt-secret-4f9a1c7d2b8e63a05d94c1f7b2a8e6d3",
        ALLOWED_ORIGINS: `http://localhost:${webPort}`,
        UPLOAD_DIR: "/tmp/lueur-e2e-uploads",
        LOG_RESET_CODES: "true"
      }
    },
    {
      command: "node e2e/static-server.js",
      url: `http://localhost:${webPort}`,
      reuseExistingServer: !process.env.CI,
      env: { WEB_PORT: webPort }
    }
  ]
});
