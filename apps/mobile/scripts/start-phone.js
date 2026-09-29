const os = require("os");
const { spawn } = require("child_process");

function getLanIp() {
  const interfaces = os.networkInterfaces();
  for (const entries of Object.values(interfaces)) {
    if (!entries) continue;
    for (const entry of entries) {
      if (entry.family !== "IPv4" || entry.internal) continue;
      if (
        entry.address.startsWith("192.168.") ||
        entry.address.startsWith("10.") ||
        entry.address.startsWith("172.")
      ) {
        return entry.address;
      }
    }
  }
  throw new Error("No LAN IPv4 found. Connect to Wi-Fi/Ethernet and retry.");
}

const ip = getLanIp();
const apiUrl = `http://${ip}:18080`;

console.log(`[mobile] Using API: ${apiUrl}`);

const child = spawn(
  process.platform === "win32" ? "npx.cmd" : "npx",
  ["expo", "start", "--clear"],
  {
    stdio: "inherit",
    env: {
      ...process.env,
      EXPO_PUBLIC_API_URL: apiUrl
    }
  }
);

child.on("exit", (code) => {
  process.exit(code ?? 0);
});
