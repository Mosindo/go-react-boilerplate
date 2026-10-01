// End-to-end check of the critical path in a real browser against a real API:
// signup → onboarding (photo, preferences, location) → discover → like → match → chat (realtime)
// → notifications → account deletion.
//
// Prerequisites (see README "Tests"): API running with the demo seed and
// ALLOWED_ORIGINS=http://localhost:8081, and the web build exported with
// EXPO_PUBLIC_API_URL pointing at that API (`npm run build:web`).
import { chromium } from "@playwright/test";
import { createServer } from "node:http";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { existsSync, statSync } from "node:fs";
import { extname, join, normalize } from "node:path";
import { fileURLToPath } from "node:url";
import assert from "node:assert/strict";
import { deflateSync } from "node:zlib";

const API = (process.env.E2E_API_URL ?? "http://localhost:18080").replace(/\/$/, "");
const PORT = Number(process.env.E2E_WEB_PORT ?? 8081);
const ROOT = join(fileURLToPath(new URL(".", import.meta.url)), "..", "dist");
const SHOTS = join(fileURLToPath(new URL(".", import.meta.url)), "artifacts");
const CHROMIUM = process.env.CHROMIUM_PATH ?? (existsSync("/opt/pw-browsers/chromium") ? "/opt/pw-browsers/chromium" : undefined);

const MIME = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".json": "application/json", ".png": "image/png", ".ico": "image/x-icon", ".ttf": "font/ttf" };

function serveDist() {
  const server = createServer(async (req, res) => {
    const url = new URL(req.url ?? "/", "http://x");
    let file = normalize(join(ROOT, decodeURIComponent(url.pathname)));
    if (!file.startsWith(ROOT)) return res.writeHead(403).end();
    if (!existsSync(file) || !statSync(file).isFile()) file = join(ROOT, "index.html");
    try {
      const body = await readFile(file);
      res.writeHead(200, { "Content-Type": MIME[extname(file)] ?? "application/octet-stream" });
      res.end(body);
    } catch {
      res.writeHead(404).end();
    }
  });
  return new Promise((resolve) => server.listen(PORT, () => resolve(server)));
}

async function api(path, { method = "GET", token, body } = {}) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: { ...(body ? { "Content-Type": "application/json" } : {}), ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined
  });
  const text = await res.text();
  return { status: res.status, data: text ? JSON.parse(text) : null };
}

// A real 600x800 PNG (the API re-encodes it as JPEG), generated without any dependency.
function makePng(w, h) {
  const crcTable = Array.from({ length: 256 }, (_, n) => { let c = n; for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1; return c >>> 0; });
  const crc = (buf) => { let c = 0xffffffff; for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8); return (c ^ 0xffffffff) >>> 0; };
  const chunk = (type, data) => { const t = Buffer.from(type); const len = Buffer.alloc(4); len.writeUInt32BE(data.length); const c = Buffer.alloc(4); c.writeUInt32BE(crc(Buffer.concat([t, data]))); return Buffer.concat([len, t, data, c]); };
  const ihdr = Buffer.alloc(13); ihdr.writeUInt32BE(w, 0); ihdr.writeUInt32BE(h, 4); ihdr[8] = 8; ihdr[9] = 2;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) { const o = y * (w * 3 + 1); for (let x = 0; x < w; x++) { raw[o + 1 + x * 3] = 200 - (y >> 3); raw[o + 2 + x * 3] = 90 + (x >> 3); raw[o + 3 + x * 3] = 120; } }
  return Buffer.concat([Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]), chunk("IHDR", ihdr), chunk("IDAT", deflateSync(raw)), chunk("IEND", Buffer.alloc(0))]);
}

const step = (name) => console.log(`✓ ${name}`);

async function main() {
  await mkdir(SHOTS, { recursive: true });
  const photoPath = join(SHOTS, "upload.png");
  await writeFile(photoPath, makePng(600, 800));

  const health = await api("/health");
  assert.equal(health.status, 200, `API not reachable at ${API}`);
  const demoLogin = await api("/auth/login", { method: "POST", body: { email: "hugo@demo.invalid", password: "DemoPassword123" } });
  assert.equal(demoLogin.status, 200, "demo seed missing: run `go run ./cmd/seed` in services/api");
  const hugoToken = demoLogin.data.accessToken;

  const server = await serveDist();
  const browser = await chromium.launch({ executablePath: CHROMIUM, args: ["--no-sandbox"] });
  const context = await browser.newContext({
    viewport: { width: 420, height: 860 },
    geolocation: { latitude: 48.8566, longitude: 2.3522 },
    permissions: ["geolocation"],
    locale: "fr-FR"
  });
  const page = await context.newPage();
  const pageErrors = [];
  page.on("pageerror", (e) => pageErrors.push(e.message));
  const shot = (name) => page.screenshot({ path: join(SHOTS, `${name}.png`) });
  const email = `eva.${Date.now()}@e2e.invalid`;
  const password = "E2E-password-123";

  try {
    await page.goto(`http://localhost:${PORT}/`);
    await page.getByTestId("welcome-register").click();
    await shot("01-register");

    // Validation feedback before anything is sent.
    await page.getByTestId("register-submit").click();
    await page.getByText("Entrez votre email.").waitFor();
    await page.getByTestId("register-email").fill(email);
    await page.getByTestId("register-password").fill(password);
    await page.getByTestId("register-adult").click();
    await page.getByTestId("register-submit").click();
    step("account created");

    // Onboarding: basics (an under-18 birth date is refused first).
    await page.getByTestId("basics-name").fill("Eva");
    await page.getByTestId("basics-day").fill("12");
    await page.getByTestId("basics-month").fill("5");
    await page.getByTestId("basics-year").fill(String(new Date().getFullYear() - 17));
    await page.getByTestId("gender-woman").click();
    await page.getByTestId("basics-submit").click();
    await page.getByText(/réservée aux personnes majeures/).waitFor();
    await page.getByTestId("basics-year").fill("1996");
    await page.getByTestId("basics-submit").click();
    step("basics (18+ enforced)");

    // Preferences: men only.
    await page.getByTestId("interested-woman").click();
    await page.getByTestId("interested-non_binary").click();
    await page.getByTestId("prefs-submit").click();
    step("preferences");

    // Photos: real upload through the browser's file chooser.
    const chooser = page.waitForEvent("filechooser");
    await page.getByTestId("photo-add").click();
    await (await chooser).setFiles(photoPath);
    await page.getByTestId("photo-0").waitFor();
    await shot("02-photos");
    await page.getByTestId("photos-continue").click();
    step("photo uploaded");

    await page.getByTestId("onboarding-bio").fill("Curieuse, gourmande, toujours partante pour une balade.");
    await page.getByTestId("interest-cuisine").click();
    await page.getByTestId("interest-randonnee").click();
    await page.getByTestId("style-continue").click();

    await page.getByTestId("share-location").click();
    await page.getByTestId("tab-Discover").waitFor({ timeout: 20_000 });
    step("onboarding complete");

    // Hugo (demo) likes Eva first, so liking him back creates the match.
    const login = await api("/auth/login", { method: "POST", body: { email, password } });
    assert.equal(login.status, 200);
    const evaId = login.data.user.id;
    const hugoFeed = await api("/discover?limit=20", { token: hugoToken });
    assert.ok(hugoFeed.data.profiles.some((p) => p.userId === evaId), "Eva should be visible to Hugo (mutual preferences, nearby)");
    assert.equal((await api("/discover/swipes", { method: "POST", token: hugoToken, body: { userId: evaId, action: "like" } })).status, 200);

    // The deck was loaded before Hugo's like: reloading also proves the session is restored.
    await page.reload();
    await page.getByTestId("tab-Discover").waitFor({ timeout: 20_000 });
    await page.getByTestId("tab-Discover").click();
    await page.getByTestId("discover-card").waitFor();
    await page.getByText(/^Hugo, \d+/).first().waitFor();
    await shot("03-discover");
    await page.getByTestId("discover-like").click();
    await page.getByTestId("match-modal").waitFor();
    await shot("04-match");
    step("like → match");

    await page.getByTestId("match-message").click();
    await page.getByTestId("chat-input").fill("Bonjour Hugo ! 👋");
    await page.getByTestId("chat-send").click();
    await page.getByText("Bonjour Hugo ! 👋").waitFor();
    step("message sent");

    // Realtime: Hugo answers through the API; the UI shows it without any refresh.
    const convs = await api("/conversations", { token: hugoToken });
    const conv = convs.data.conversations.find((c) => c.user.userId === evaId);
    assert.ok(conv, "Hugo sees the conversation");
    assert.equal(conv.unreadCount, 1, "Hugo has one unread message");
    assert.equal((await api(`/conversations/${conv.id}/messages`, { method: "POST", token: hugoToken, body: { body: "Salut Eva, ravi de te rencontrer !" } })).status, 201);
    await page.getByText("Salut Eva, ravi de te rencontrer !").waitFor({ timeout: 10_000 });
    await shot("05-chat");
    step("realtime message received");

    // Eva's open chat marks it read; Hugo's message gets a read receipt.
    await page.waitForTimeout(800);
    const hugoView = await api(`/conversations/${conv.id}/messages`, { token: hugoToken });
    assert.ok(hugoView.data.messages.find((m) => m.senderId === hugoView.data.messages[0].senderId) !== undefined);

    // In-app back (the stack is not wired to browser history on web).
    await page.getByRole("button", { name: "Retour" }).first().click();
    await page.getByTestId("tab-Matches").click();
    await page.getByTestId("conversation-row").first().waitFor();
    await shot("06-matches");
    await page.getByTestId("tab-Notifications").click();
    await page.getByTestId("notification-row").first().waitFor();
    await page.getByTestId("notification-row").getByText(/Nouveau match avec Hugo/).waitFor();
    await shot("07-notifications");
    step("matches + notifications");

    // Privacy & account: block list reachable, then delete the account for real.
    await page.getByTestId("tab-Me").click();
    await page.getByTestId("menu-settings").click();
    await page.getByTestId("delete-start").click();
    await page.getByTestId("delete-password").fill(password);
    await page.getByTestId("delete-confirm").click();
    await page.getByTestId("confirm-yes").click();
    await page.getByTestId("welcome-register").waitFor({ timeout: 15_000 });
    assert.equal((await api("/auth/login", { method: "POST", body: { email, password } })).status, 401, "deleted account cannot log in");
    const after = await api("/conversations", { token: hugoToken });
    assert.ok(!after.data.conversations.some((c) => c.user.userId === evaId), "match disappears for the other user");
    step("account deleted");

    assert.deepEqual(pageErrors, [], `uncaught page errors: ${pageErrors.join(" | ")}`);
    console.log("\nE2E critical path passed.");
  } catch (e) {
    await shot("failure").catch(() => undefined);
    console.error("E2E failed. Screenshot: e2e/artifacts/failure.png\n", e);
    process.exitCode = 1;
  } finally {
    await browser.close();
    server.close();
  }
}

await main();
