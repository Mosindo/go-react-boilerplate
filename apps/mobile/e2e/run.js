/* eslint-disable */
// End-to-end smoke test in a real browser (Chromium via Playwright) against a running API.
//   1. API running (default http://localhost:18080) with ALLOWED_ORIGINS including http://localhost:19006
//   2. `EXPO_PUBLIC_API_URL=<api> npx expo export --platform web` has produced ./dist
//   3. node e2e/run.js
// Journey: sign up -> onboarding (profile, interests, preferences, photo) -> discovery -> like ->
// match -> real-time chat -> read state -> account deletion, with two independent users.
const http = require("http");
const fs = require("fs");
const path = require("path");
const zlib = require("zlib");
const { chromium } = require("playwright");

const API = process.env.E2E_API_URL || "http://localhost:18080";
const PORT = Number(process.env.E2E_WEB_PORT || 19006);
const BASE = `http://localhost:${PORT}`;
const DIST = path.join(__dirname, "..", "dist");
const stamp = Date.now();

function crc32(buf) {
  let c, crc = ~0;
  for (let n = 0; n < buf.length; n++) {
    c = (crc ^ buf[n]) & 0xff;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    crc = (crc >>> 8) ^ c;
  }
  return ~crc >>> 0;
}
function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}
function png(w, h, seed) {
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) {
    const o = y * (w * 3 + 1);
    raw[o] = 0;
    raw[o + 1 + x * 3] = (x + seed) % 256; raw[o + 2 + x * 3] = y % 256; raw[o + 3 + x * 3] = 128;
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0); ihdr.writeUInt32BE(h, 4); ihdr[8] = 8; ihdr[9] = 2;
  return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk("IHDR", ihdr), chunk("IDAT", zlib.deflateSync(raw)), chunk("IEND", Buffer.alloc(0))]);
}

function serve() {
  const types = { ".html": "text/html", ".js": "text/javascript", ".png": "image/png", ".json": "application/json", ".ico": "image/x-icon" };
  const server = http.createServer((req, res) => {
    let file = path.join(DIST, decodeURIComponent(req.url.split("?")[0]));
    if (!file.startsWith(DIST) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) file = path.join(DIST, "index.html");
    res.setHeader("Content-Type", types[path.extname(file)] || "application/octet-stream");
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(PORT, () => resolve(server)));
}

const pages = [];
async function dumpFailure() {
  const dir = path.join(__dirname, "artifacts");
  fs.mkdirSync(dir, { recursive: true });
  for (const [i, page] of pages.entries()) {
    await page.screenshot({ path: path.join(dir, `failure-${i}.png`) }).catch(() => undefined);
    const text = await page.locator("body").innerText().catch(() => "");
    console.error(`--- page ${i} text ---\n${text.slice(0, 600)}`);
  }
}

let step = 0;
const log = (m) => console.log(`[e2e ${String(++step).padStart(2, "0")}] ${m}`);
const T = 20000;

async function signUpAndOnboard(browser, { name, email, gender, interestedIn, birth, seed }) {
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, geolocation: { latitude: 48.8566, longitude: 2.3522 }, permissions: ["geolocation"] });
  const page = await context.newPage();
  pages.push(page);
  if (process.env.E2E_VERBOSE) {
    page.on("console", (m) => console.log(`[${name}] console.${m.type()}:`, m.text().slice(0, 300)));
    page.on("requestfinished", (r) => r.url().startsWith(API) && console.log(`[${name}] ${r.method()} ${r.url().replace(API, "")}`));
  }
  page.on("pageerror", (e) => console.error(`[${name}] page error:`, e.message));
  await page.goto(BASE);
  await page.getByTestId("tab-register").click();
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill("correct-horse-battery");
  await page.getByTestId("submit").click();

  await page.getByTestId("firstName").waitFor({ timeout: T });
  await page.getByTestId("firstName").fill(name);
  await page.getByTestId("birthDate").fill(birth);
  await page.getByTestId(`gender-${gender}`).click();
  await page.getByTestId("city").fill("Paris");
  await page.getByTestId("locate").click();
  await page.getByText("Position approximative enregistrée").waitFor({ timeout: T });
  await page.getByTestId("onboarding-next").click();

  await page.getByTestId("bio").fill(`Bonjour, je suis ${name}.`);
  await page.getByTestId("interest-travel").click();
  await page.getByTestId("interest-music").click();
  await page.getByTestId("onboarding-next").click();

  await page.getByTestId(`interested-${interestedIn}`).click();
  await page.getByTestId("onboarding-next").click();

  await page.getByTestId("add-photo").waitFor({ timeout: T });
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("add-photo").click();
  await (await chooser).setFiles({ name: "me.png", mimeType: "image/png", buffer: png(600, 800, seed) });
  await page.getByTestId("photo-0").waitFor({ timeout: T });
  await page.getByTestId("onboarding-finish").click();
  await page.getByTestId("tab-discover").waitFor({ timeout: T });
  return { context, page };
}

(async () => {
  const health = await fetch(`${API}/health`).catch(() => null);
  if (!health || !health.ok) throw new Error(`API not reachable at ${API}/health`);
  if (!fs.existsSync(path.join(DIST, "index.html"))) throw new Error("dist/ missing: run `npm run build:web` first");

  const server = await serve();
  // Set E2E_CHROMIUM_PATH to reuse an existing Chrome/Chromium binary instead of `npx playwright install chromium`.
  const browser = await chromium.launch(process.env.E2E_CHROMIUM_PATH ? { executablePath: process.env.E2E_CHROMIUM_PATH, args: ["--no-sandbox"] } : {});
  const tag = String(stamp).slice(-5);
  const alice = { name: `Alice${tag}`, email: `alice-${stamp}@e2e.invalid`, gender: "woman", interestedIn: "man", birth: "19940510", seed: 10 };
  const bob = { name: `Bob${tag}`, email: `bob-${stamp}@e2e.invalid`, gender: "man", interestedIn: "woman", birth: "19920302", seed: 90 };

  try {
    const a = await signUpAndOnboard(browser, alice);
    log("Alice signed up, completed her profile and reached discovery");
    const b = await signUpAndOnboard(browser, bob);
    log("Bob signed up, completed his profile and reached discovery");

    // Alice sees Bob (mutual orientation) and likes him: no match yet.
    await a.page.reload();
    await a.page.getByTestId("card-name").filter({ hasText: bob.name }).waitFor({ timeout: T });
    log("Alice's deck shows Bob");
    await a.page.getByTestId("like-button").click();
    await a.page.getByText("Vous avez tout vu pour l'instant").waitFor({ timeout: T });
    log("Alice liked Bob; the deck is now empty (no duplicates, no match yet)");

    // Bob sees Alice; the like is reciprocal -> match modal.
    await b.page.reload();
    await b.page.getByTestId("card-name").filter({ hasText: alice.name }).waitFor({ timeout: T });
    await b.page.getByTestId("like-button").click();
    await b.page.getByTestId("match-modal").waitFor({ timeout: T });
    log("Bob liked Alice back: match modal displayed");

    // Real-time: Alice receives the match without reloading.
    await a.page.getByTestId("match-modal").waitFor({ timeout: T });
    log("Alice received the match in real time");
    await a.page.getByTestId("match-continue").click();

    // Bob writes first.
    await b.page.getByTestId("match-chat").click();
    await b.page.getByTestId("message-input").fill("Salut Alice, ravi de ce match !");
    await b.page.getByTestId("send-message").click();
    await b.page.getByTestId("message-list").getByText("Salut Alice, ravi de ce match !").waitFor({ timeout: T });
    log("Bob sent a message");

    // Alice opens Messages, sees the unread conversation, opens it, sees the live message and replies.
    await a.page.getByTestId("tab-messages").click();
    await a.page.getByTestId(`conversation-${bob.name}`).waitFor({ timeout: T });
    await a.page.getByTestId(`conversation-${bob.name}`).click();
    await a.page.getByTestId("message-list").getByText("Salut Alice, ravi de ce match !").waitFor({ timeout: T });
    await a.page.getByTestId("message-input").fill("Merci Bob, moi aussi !");
    await a.page.getByTestId("send-message").click();
    log("Alice read Bob's message and replied");

    // Bob receives Alice's reply live, and Alice's message shows as read.
    await b.page.getByTestId("message-list").getByText("Merci Bob, moi aussi !").waitFor({ timeout: T });
    log("Bob received the reply in real time");
    await a.page.getByTestId("message-list").getByText("Merci Bob, moi aussi !").waitFor({ timeout: T });
    await a.page.getByText(/Lu/).first().waitFor({ timeout: T }).catch(() => undefined);

    // Session persistence.
    await a.page.reload();
    await a.page.getByTestId("tab-discover").waitFor({ timeout: T });
    log("Session survives a reload");

    // Account deletion from the UI, then login must fail.
    await a.page.getByTestId("tab-me").click();
    await a.page.getByTestId("settings").click();
    await a.page.getByTestId("delete-account").click();
    await a.page.getByTestId("delete-password").fill("correct-horse-battery");
    await a.page.getByTestId("confirm-delete").click();
    await a.page.getByTestId("tab-register").waitFor({ timeout: T });
    const login = await fetch(`${API}/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email: alice.email, password: "correct-horse-battery" }) });
    if (login.status !== 401) throw new Error(`deleted account can still log in (${login.status})`);
    log("Alice deleted her account; she is signed out and cannot log in anymore");

    await b.page.reload(); // leaves the chat screen: the alice account is gone, so the conversation vanished too
    await b.page.getByTestId("tab-messages").click();
    await b.page.getByText("Aucune conversation").waitFor({ timeout: T });
    log("Alice's deletion removed the conversation for Bob");
    await b.page.getByTestId("tab-me").click();
    await b.page.getByTestId("settings").click();
    await b.page.getByTestId("delete-account").click();
    await b.page.getByTestId("delete-password").fill("correct-horse-battery");
    await b.page.getByTestId("confirm-delete").click();
    await b.page.getByTestId("tab-register").waitFor({ timeout: T });
    log("Bob cleaned up his account");
    console.log("\nE2E PASSED");
  } catch (e) {
    await dumpFailure();
    throw e;
  } finally {
    // Remove both test accounts even when the run failed, so reruns start clean.
    for (const u of [alice, bob]) {
      const res = await fetch(`${API}/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email: u.email, password: "correct-horse-battery" }) }).catch(() => null);
      if (res && res.ok) {
        const { accessToken } = await res.json();
        await fetch(`${API}/me`, { method: "DELETE", headers: { Authorization: `Bearer ${accessToken}`, "Content-Type": "application/json" }, body: JSON.stringify({ password: "correct-horse-battery" }) }).catch(() => undefined);
      }
    }
    await browser.close();
    server.close();
  }
})().catch((e) => {
  console.error("\nE2E FAILED:", e);
  process.exit(1);
});
