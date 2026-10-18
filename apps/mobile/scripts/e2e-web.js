#!/usr/bin/env node
/**
 * End-to-end run in a real browser: boots the Go API (against E2E_DATABASE_URL),
 * exports the web build pointed at it, serves it, and runs the Playwright suite.
 *
 *   E2E_DATABASE_URL=postgres://user:pass@localhost:5432/app_e2e?sslmode=disable npm run e2e:web
 */
const { spawn, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");

const appDir = path.resolve(__dirname, "..");
const apiDir = path.resolve(appDir, "../../services/api");
const databaseUrl = process.env.E2E_DATABASE_URL;
const apiPort = process.env.E2E_API_PORT ?? "18090";
const webPort = process.env.E2E_WEB_PORT ?? "18091";
const apiUrl = `http://localhost:${apiPort}`;
const webUrl = `http://localhost:${webPort}`;

if (!databaseUrl) {
  console.error("E2E_DATABASE_URL is required (a throwaway PostgreSQL database; the API migrates it).");
  process.exit(2);
}

const workDir = fs.mkdtempSync(path.join(os.tmpdir(), "dating-e2e-"));
const children = [];

function run(command, args, options) {
  const result = spawnSync(command, args, { stdio: "inherit", ...options });
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed (${result.status})`);
  }
}

function cleanup() {
  for (const child of children) child.kill("SIGTERM");
  fs.rmSync(workDir, { recursive: true, force: true });
}

async function waitFor(url, label) {
  for (let i = 0; i < 60; i++) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // not up yet
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error(`${label} did not become ready at ${url}`);
}

function serveStatic(dir, port) {
  const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".png": "image/png", ".json": "application/json", ".ttf": "font/ttf" };
  const server = http.createServer((req, res) => {
    const requested = path.normalize(decodeURIComponent(new URL(req.url, webUrl).pathname)).replace(/^(\.\.[/\\])+/, "");
    let file = path.join(dir, requested);
    if (!file.startsWith(dir) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
      file = path.join(dir, "index.html"); // SPA fallback
    }
    res.writeHead(200, { "Content-Type": types[path.extname(file)] ?? "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  });
  server.listen(port);
  return server;
}

async function main() {
  const apiBinary = path.join(workDir, process.platform === "win32" ? "api.exe" : "api");
  run("go", ["build", "-o", apiBinary, "./cmd/api"], { cwd: apiDir });

  const api = spawn(apiBinary, [], {
    stdio: "inherit",
    env: {
      ...process.env,
      PORT: apiPort,
      DATABASE_URL: databaseUrl,
      JWT_SECRET: process.env.JWT_SECRET ?? "e2e-only-secret-0123456789abcdef0123456789",
      UPLOAD_DIR: path.join(workDir, "uploads"),
      ALLOWED_ORIGINS: webUrl,
      RATE_LIMIT_ENABLED: "false",
      GIN_MODE: "release"
    }
  });
  children.push(api);
  await waitFor(`${apiUrl}/health`, "API");

  const webDir = path.join(appDir, ".e2e", "web");
  fs.rmSync(webDir, { recursive: true, force: true });
  run("npx", ["expo", "export", "--clear", "--platform", "web", "--output-dir", webDir], {
    cwd: appDir,
    shell: process.platform === "win32",
    env: { ...process.env, EXPO_PUBLIC_API_URL: apiUrl }
  });
  const web = serveStatic(webDir, Number(webPort));
  fs.mkdirSync(path.join(appDir, ".e2e", "screens"), { recursive: true });

  // Async spawn: a synchronous child would block the event loop that serves the web build.
  const playwright = await new Promise((resolve) => {
    const child = spawn("npx", ["playwright", "test", ...process.argv.slice(2)], {
      cwd: appDir,
      stdio: "inherit",
      shell: process.platform === "win32",
      env: { ...process.env, E2E_WEB_URL: webUrl }
    });
    child.on("exit", (code) => resolve({ status: code }));
  });
  web.close();
  return playwright.status ?? 1;
}

main()
  .then((code) => {
    cleanup();
    process.exit(code);
  })
  .catch((error) => {
    console.error(error);
    cleanup();
    process.exit(1);
  });
