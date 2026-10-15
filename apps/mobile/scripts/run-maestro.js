/* eslint-disable no-console */
"use strict";
// Runs the critical Maestro flow with the fixtures created by e2e-ui-setup.js.
//
//   npm run e2e:ui:setup && EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080 npm run start:phone
//   APP_ID=host.exp.exponent npm run e2e:ui:run     # Expo Go; use your own app id for a dev build

const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const envFile = path.join(process.cwd(), ".e2e", "maestro-env.json");
if (!fs.existsSync(envFile)) {
  console.error("[maestro-run] Missing .e2e/maestro-env.json. Run `npm run e2e:ui:setup` first.");
  process.exit(1);
}

const env = JSON.parse(fs.readFileSync(envFile, "utf8"));
const appId = process.env.APP_ID || "host.exp.exponent";
const args = ["test", "-e", `APP_ID=${appId}`];
for (const [key, value] of Object.entries(env)) {
  args.push("-e", `${key}=${value}`);
}
args.push(path.join("e2e", "maestro", "critical-flow.yaml"));

console.log(`[maestro-run] APP_ID=${appId}`);
const result = spawnSync("maestro", args, { stdio: "inherit", shell: process.platform === "win32" });
if (result.error) {
  console.error("[maestro-run] Could not start Maestro. Install it from https://maestro.mobile.dev and make sure it is on PATH.");
  process.exit(1);
}
process.exit(result.status ?? 1);
