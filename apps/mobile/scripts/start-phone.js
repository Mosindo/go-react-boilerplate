// Starts Expo for a physical phone. The phone cannot reach "localhost", so the API URL is built from this
// computer's LAN IP. Override with EXPO_PUBLIC_API_URL (full URL) or API_PORT (host port, default 18080).
const os = require("os");
const { spawn } = require("child_process");

const DEFAULT_HOST_PORT = "18080";

function getLanIp() {
  const interfaces = os.networkInterfaces();
  for (const entries of Object.values(interfaces)) {
    if (!entries) continue;
    for (const entry of entries) {
      if (entry.family !== "IPv4" || entry.internal) continue;
      if (
        entry.address.startsWith("192.168.") ||
        entry.address.startsWith("10.") ||
        /^172\.(1[6-9]|2\d|3[01])\./.test(entry.address)
      ) {
        return entry.address;
      }
    }
  }
  throw new Error("No LAN IPv4 found. Connect to Wi-Fi/Ethernet and retry.");
}

async function checkHealth(apiUrl) {
  try {
    const response = await fetch(`${apiUrl}/health`, { signal: AbortSignal.timeout(3000) });
    if (response.ok) {
      console.log(`[mobile] API health check OK (${apiUrl}/health)`);
      return;
    }
    console.warn(
      `[mobile] WARNING: ${apiUrl}/health answered HTTP ${response.status}. Is another service using this port?`
    );
  } catch {
    console.warn(
      `[mobile] WARNING: ${apiUrl}/health is unreachable. Start the API (docker compose -f infra/docker-compose.yml up) ` +
        "and check `docker compose ps` for port conflicts before testing on the phone."
    );
  }
}

async function main() {
  const configured = process.env.EXPO_PUBLIC_API_URL && process.env.EXPO_PUBLIC_API_URL.trim();
  if (configured && /\/\/(localhost|127\.0\.0\.1)(:|\/|$)/.test(configured)) {
    console.warn("[mobile] WARNING: localhost points at the phone itself. Use your computer's LAN IP.");
  }
  const apiUrl = (configured || `http://${getLanIp()}:${process.env.API_PORT || DEFAULT_HOST_PORT}`).replace(
    /\/+$/,
    ""
  );

  console.log(`[mobile] Using API: ${apiUrl}`);
  await checkHealth(apiUrl);

  const child = spawn(process.platform === "win32" ? "npx.cmd" : "npx", ["expo", "start", "--clear"], {
    stdio: "inherit",
    env: { ...process.env, EXPO_PUBLIC_API_URL: apiUrl }
  });
  child.on("exit", (code) => process.exit(code ?? 0));
}

main().catch((error) => {
  console.error(`[mobile] ${error.message}`);
  process.exit(1);
});
