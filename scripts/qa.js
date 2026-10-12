/* eslint-disable no-console */
// Cross-platform quality gate. Run from the repository root: node scripts/qa.js
// Needs DATABASE_URL (or DATABASE_URL_TEST) to run the Go integration tests.
const { spawnSync } = require("child_process");
const path = require("path");

const root = path.join(__dirname, "..");
const api = path.join(root, "services", "api");
const mobile = path.join(root, "apps", "mobile");
const win = process.platform === "win32";

function run(name, cwd, cmd, args) {
  console.log(`\n▶ ${name}`);
  const result = spawnSync(win && cmd === "npx" ? "npx.cmd" : win && cmd === "npm" ? "npm.cmd" : cmd, args, { cwd, stdio: "inherit" });
  if (result.status !== 0) {
    console.error(`✗ ${name} failed`);
    process.exit(result.status ?? 1);
  }
}

function gofmtCheck() {
  console.log("\n▶ gofmt");
  const result = spawnSync("gofmt", ["-l", "."], { cwd: api, encoding: "utf8" });
  if (result.status !== 0 || result.stdout.trim()) {
    console.error(`✗ gofmt: unformatted files:\n${result.stdout}${result.stderr}`);
    process.exit(1);
  }
}

if (!process.env.DATABASE_URL && !process.env.DATABASE_URL_TEST) {
  console.warn("! DATABASE_URL is not set: Go integration tests will be skipped.");
}

gofmtCheck();
run("go vet", api, "go", ["vet", "./..."]);
run("go test", api, "go", ["test", "-count=1", "./..."]);
run("go build", api, "go", ["build", "-o", path.join(root, ".cache", win ? "api.exe" : "api"), "./cmd/api"]);
run("mobile typecheck", mobile, "npm", ["run", "typecheck"]);
run("mobile lint", mobile, "npm", ["run", "lint"]);
run("mobile format", mobile, "npm", ["run", "format:check"]);
run("mobile tests", mobile, "npm", ["test", "--", "--silent"]);
console.log("\n✓ all checks passed");
