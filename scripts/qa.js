#!/usr/bin/env node
/**
 * Full local quality gate, mirroring CI:
 *   node scripts/qa.js            # static checks, unit + integration tests, builds
 *   node scripts/qa.js --smoke    # + API smoke against a running API
 *
 * Integration tests run when DATABASE_URL (or DATABASE_URL_TEST) points to a
 * reachable PostgreSQL; otherwise they are skipped by the Go test suite.
 */
const { spawnSync } = require("child_process");
const path = require("path");

const root = path.resolve(__dirname, "..");
const api = path.join(root, "services/api");
const mobile = path.join(root, "apps/mobile");
const npx = process.platform === "win32" ? "npx.cmd" : "npx";

const steps = [
  ["gofmt", "gofmt", ["-l", "."], api, (out) => out.trim() === "" || `unformatted Go files:\n${out}`],
  ["go vet", "go", ["vet", "./..."], api],
  ["go test", "go", ["test", "-count=1", "./..."], api],
  ["go build", "go", ["build", "-o", process.platform === "win32" ? "NUL" : "/dev/null", "./cmd/api"], api],
  ["typecheck", npx, ["tsc", "--noEmit"], mobile],
  ["lint", npx, ["eslint", ".", "--max-warnings", "0"], mobile],
  ["format", npx, ["prettier", "--check", "."], mobile]
];
if (process.argv.includes("--smoke")) {
  steps.push(["api smoke", "node", ["scripts/e2e-smoke.js"], mobile]);
}

let failed = false;
for (const [name, cmd, args, cwd, check] of steps) {
  process.stdout.write(`▶ ${name} … `);
  const res = spawnSync(cmd, args, { cwd, encoding: "utf8", shell: process.platform === "win32" });
  const output = `${res.stdout ?? ""}${res.stderr ?? ""}`;
  const verdict = res.status !== 0 ? output || res.error?.message : check ? check(res.stdout ?? "") : true;
  if (verdict === true) {
    console.log("ok");
  } else {
    failed = true;
    console.log("FAILED");
    console.log(String(verdict).split("\n").slice(-40).join("\n"));
  }
}
process.exit(failed ? 1 : 0);
