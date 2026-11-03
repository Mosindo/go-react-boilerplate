// Browser E2E for the critical path:
// register -> onboarding -> discover -> like -> match -> chat (+ realtime reply).
//
// Prerequisites: API running (see README), web bundle served at WEB_URL
// (`npm run build:web` then any static server), and Playwright available
// (`npm i -D playwright` or a global install). Run: node e2e/web-flow.mjs
import { createRequire } from "node:module";
import { writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const WEB_URL = process.env.WEB_URL ?? "http://localhost:8099";
const API_URL = process.env.API_URL ?? "http://localhost:18080";
const SHOTS = process.env.SHOTS_DIR ?? tmpdir();
const email = `zoe_${Date.now()}@demo.invalid`;

async function api(path, { method = "GET", token, body } = {}) {
  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined
  });
  const text = await res.text();
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status} ${text}`);
  return text ? JSON.parse(text) : undefined;
}

function assert(condition, message) {
  if (!condition) throw new Error(`ASSERTION FAILED: ${message}`);
  console.log(`ok - ${message}`);
}

// Minimal valid PNG (solid colour) large enough for the 200x200 minimum.
async function makePhoto() {
  const { deflateSync } = await import("node:zlib");
  const w = 400,
    h = 500;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) {
    raw[y * (w * 3 + 1)] = 0;
    for (let x = 0; x < w; x++) {
      const o = y * (w * 3 + 1) + 1 + x * 3;
      raw[o] = 200;
      raw[o + 1] = 90 + (y % 120);
      raw[o + 2] = 80;
    }
  }
  const crcTable = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (buf) => {
    let c = 0xffffffff;
    for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const td = Buffer.concat([Buffer.from(type), data]);
    const c = Buffer.alloc(4);
    c.writeUInt32BE(crc(td));
    return Buffer.concat([len, td, c]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  const png = Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
  const file = join(tmpdir(), "braise-e2e-photo.png");
  writeFileSync(file, png);
  return file;
}

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 430, height: 900 } });
page.on("pageerror", (e) => console.log("pageerror:", e.message));
const byId = (id) => page.getByTestId(id);

try {
  await page.goto(WEB_URL);

  // 1. Register
  await byId("goto-register").click();
  await byId("register-email").fill(email);
  await byId("register-password").fill("Password123");
  await byId("register-submit").click();

  // 2. Onboarding: basics -> about -> preferences -> photo -> skip location
  await byId("field-firstName").fill("Zoé");
  await byId("field-birthDate").fill("19950614");
  await byId("gender-woman").click();
  await byId("onboarding-next").click();
  await byId("field-bio").fill("Test E2E : randonnées et cinéma.");
  await byId("field-city").fill("Paris");
  await byId("interest-hiking").click();
  await byId("interest-cinema").click();
  await byId("onboarding-next").click();
  await byId("interested-man").click();
  await byId("distance-500").click();
  await byId("onboarding-next").click();

  const chooser = page.waitForEvent("filechooser");
  await byId("add-photo").click();
  (await chooser).setFiles(await makePhoto());
  await byId("photo-0").waitFor();
  assert(true, "photo uploaded and listed during onboarding");
  await byId("onboarding-next").click();
  await page.screenshot({ path: join(SHOTS, "e2e-1-onboarding-location.png") });
  await byId("onboarding-skip").click();

  // 3. Discover shows a card; Hugo (demo) likes Zoé through the API first.
  await byId("swipe-card").waitFor({ timeout: 15000 });
  assert(true, "discover shows a profile card");
  await page.screenshot({ path: join(SHOTS, "e2e-2-discover.png") });

  const hugo = await api("/auth/login", { method: "POST", body: { email: "hugo@demo.invalid", password: "DemoPassword123" } });
  const me = await api("/me", { token: hugo.accessToken }).catch(() => null);
  assert(me !== null, "demo account can log in");
  const zoeLogin = await page.evaluate(() => localStorage.getItem("braise.session.v1"));
  const zoeToken = JSON.parse(zoeLogin).accessToken;
  const zoe = await api("/me", { token: zoeToken });
  const swipe = await api("/swipes", { method: "POST", token: hugo.accessToken, body: { targetId: zoe.id, action: "like" } });
  assert(swipe.matched === false, "demo like is one-sided");

  // 4. Likes tab lists Hugo; liking back creates the match overlay.
  await page.getByText("Likes", { exact: true }).click();
  await byId("like-back-Hugo").click();
  await byId("match-overlay").waitFor({ timeout: 10000 });
  assert(true, "mutual like shows the match overlay");
  await page.screenshot({ path: join(SHOTS, "e2e-3-match.png") });

  // 5. Chat: send a message, then receive a realtime reply from Hugo.
  await byId("match-send-message").click();
  await byId("message-input").fill("Salut Hugo, ravie de te rencontrer !");
  await byId("send-message").click();
  await page.getByText("Salut Hugo, ravie de te rencontrer !").waitFor();
  assert(true, "message appears in the conversation");

  const convs = await api("/conversations", { token: hugo.accessToken });
  const conversation = convs.conversations.find((c) => c.user.id === zoe.id);
  assert(conversation && conversation.unreadCount === 1, "recipient sees one unread message");
  await api(`/conversations/${conversation.id}/messages`, {
    method: "POST",
    token: hugo.accessToken,
    body: { body: "Salut Zoé ! Enchanté." }
  });
  await page.getByText("Salut Zoé ! Enchanté.").waitFor({ timeout: 10000 });
  assert(true, "realtime reply is delivered over WebSocket without refresh");
  await page.waitForTimeout(600);
  assert((await byId("match-overlay").count()) === 0, "match overlay is dismissed after opening the chat");
  await page.screenshot({ path: join(SHOTS, "e2e-4-chat.png") });

  // 6. Cleanup: delete the account through the UI.
  await page.goBack().catch(() => undefined);
  await api("/me", { method: "DELETE", token: zoeToken, body: { password: "Password123" } });
  console.log("E2E PASSED");
} catch (error) {
  await page.screenshot({ path: join(SHOTS, "e2e-failure.png") }).catch(() => undefined);
  console.error(error);
  process.exitCode = 1;
} finally {
  await browser.close();
}
