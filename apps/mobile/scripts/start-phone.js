// Starts Expo with EXPO_PUBLIC_API_URL pointing at this computer's LAN address, so a physical phone
// can reach the API. (`localhost` on a phone is the phone itself.)
// Usage: API_PORT=18080 npm run start:phone
const os = require("os");
const { spawn } = require("child_process");

function lanAddress() {
  for (const addresses of Object.values(os.networkInterfaces())) {
    for (const a of addresses ?? []) {
      if (
        a.family === "IPv4" &&
        !a.internal &&
        /^(10\.|192\.168\.|172\.(1[6-9]|2\d|3[01])\.)/.test(a.address)
      ) {
        return a.address;
      }
    }
  }
  return null;
}

const ip = process.env.LAN_IP || lanAddress();
if (!ip) {
  console.error("No LAN address found. Set LAN_IP=<your computer's IP> and retry.");
  process.exit(1);
}
const port = process.env.API_PORT || "18080";
const apiUrl = `http://${ip}:${port}`;
console.log(`API url for the phone: ${apiUrl}`);
console.log(`Check it first:  curl ${apiUrl}/health`);

const child = spawn("npx", ["expo", "start"], {
  stdio: "inherit",
  shell: process.platform === "win32",
  env: { ...process.env, EXPO_PUBLIC_API_URL: apiUrl }
});
child.on("exit", (code) => process.exit(code ?? 0));
