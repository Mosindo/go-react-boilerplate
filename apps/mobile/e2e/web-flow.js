/**
 * Browser E2E of the critical journey on the web build (React Native Web):
 *   register -> onboarding (photo upload) -> discovery -> like -> match
 *   -> realtime conversation -> read receipt -> account deletion
 *
 * Requirements: a running API (E2E_API_URL, default http://localhost:18080)
 * with ALLOWED_ORIGINS including the web origin, a web build in ./dist
 * (npm run build:web with EXPO_PUBLIC_API_URL pointing to the API), and
 * Playwright with Chromium available (`npm i -g playwright && npx playwright install chromium`).
 */
const fs = require("fs");
const os = require("os");
const path = require("path");
const zlib = require("zlib");
const { chromium } = require("playwright");
const { serve } = require("./serve-static");

const API = (process.env.E2E_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const WEB_PORT = Number(process.env.E2E_WEB_PORT || 8081);
const ARTIFACTS = path.join(__dirname, ".artifacts");
const PASSWORD = "Password123";

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function api(pathname, { method = "GET", body, token, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) payload = form;
  else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(`${API}${pathname}`, { method, headers, body: payload });
  const text = await res.text();
  return { status: res.status, json: text ? JSON.parse(text) : null };
}

function png(width, height, rgb) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    for (let x = 0; x < width; x++) {
      raw[row + 1 + x * 3] = (rgb[0] + x / 4) & 0xff;
      raw[row + 2 + x * 3] = (rgb[1] + y / 4) & 0xff;
      raw[row + 3 + x * 3] = rgb[2];
    }
  }
  const table = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (buf) => {
    let c = 0xffffffff;
    for (const b of buf) c = table[(c ^ b) & 0xff] ^ (c >>> 8);
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
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function createPartner() {
  const email = `e2e_partner_${Date.now()}@e2e.lueur.invalid`;
  const reg = await api("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  assert(reg.status === 201, `partner register ${reg.status}`);
  const token = reg.json.accessToken;
  await api("/me/profile", {
    method: "PUT",
    token,
    body: {
      firstName: "Bastien",
      birthdate: "1992-05-14",
      gender: "man",
      bio: "Grimpeur et amateur de cafés.",
      city: "Paris",
      jobTitle: "Architecte"
    }
  });
  await api("/me/preferences", { method: "PUT", token, body: { interestedIn: ["woman"], minAge: 18, maxAge: 99, maxDistanceKm: 100 } });
  await api("/me/location", { method: "PUT", token, body: { latitude: 48.86, longitude: 2.34 } });
  const form = new FormData();
  form.append("photo", new Blob([png(480, 600, [60, 90, 110])], { type: "image/png" }), "p.png");
  const photo = await api("/me/photos", { method: "POST", token, form });
  assert(photo.status === 201, `partner photo ${photo.status}`);
  return { email, token, id: reg.json.user.id };
}

async function main() {
  fs.mkdirSync(ARTIFACTS, { recursive: true });
  const server = await serve(path.join(__dirname, "..", "dist"), WEB_PORT);
  const browser = await chromium.launch();
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, locale: "fr-FR" });
  const page = await context.newPage();
  page.on("pageerror", (err) => console.error("[browser]", err.message));
  const shot = async (name) => {
    await page.waitForLoadState("networkidle").catch(() => undefined);
    await page.waitForTimeout(400);
    await page.screenshot({ path: path.join(ARTIFACTS, `${name}.png`) });
  };
  const byTest = (id) => page.getByTestId(id);
  const email = `e2e_user_${Date.now()}@e2e.lueur.invalid`;
  const photoFile = path.join(os.tmpdir(), `lueur-e2e-${Date.now()}.png`);
  fs.writeFileSync(photoFile, png(640, 800, [200, 120, 90]));
  let partner;

  try {
    partner = await createPartner();
    await page.goto(`http://localhost:${WEB_PORT}/`);
    await byTest("welcome-register").click();
    await shot("01-register");
    await byTest("register-email").fill(email);
    await byTest("register-password").fill(PASSWORD);
    await byTest("register-submit").click();

    // Onboarding
    await byTest("onboarding-firstname").fill("Juliette");
    await byTest("birth-day").fill("12");
    await byTest("birth-month").fill("06");
    await byTest("birth-year").fill("1995");
    await shot("02-onboarding-identity");
    await byTest("onboarding-next").click();
    await byTest("gender-woman").click();
    await byTest("seeking-man").click();
    await byTest("onboarding-next").click();

    const chooser = page.waitForEvent("filechooser");
    await byTest("photo-add-0").click();
    await (await chooser).setFiles(photoFile);
    await byTest("photo-slot-0").waitFor({ timeout: 20_000 });
    await shot("03-onboarding-photos");
    await byTest("onboarding-next").click();
    await byTest("onboarding-bio").fill("Lectrice, randonneuse, toujours partante pour un bon café.");
    await byTest("interest-hiking").click();
    await byTest("interest-coffee").click();
    await byTest("onboarding-next").click();

    // The partner likes the new member before she reaches discovery.
    const login = await api("/auth/login", { method: "POST", body: { email, password: PASSWORD } });
    const memberId = login.json.user.id;
    const liked = await api("/swipes", { method: "POST", token: partner.token, body: { targetUserId: memberId, action: "like" } });
    assert(liked.status === 201, `partner like ${liked.status} ${JSON.stringify(liked.json)}`);

    await byTest("onboarding-skip-location").click();

    // Discovery: the partner who liked her comes first.
    await byTest("discovery-card").waitFor({ timeout: 20_000 });
    await page.getByText("Bastien, ", { exact: false }).first().waitFor();
    await page.getByText("VOUS A LIKÉ").first().waitFor();
    await shot("04-discovery");
    await byTest("discovery-like").click();

    // Match
    await byTest("match-message").waitFor({ timeout: 10_000 });
    await page.waitForTimeout(600);
    await shot("05-match");
    await byTest("match-message").click();

    // Conversation + realtime
    await byTest("conversation-input").fill("Bonjour Bastien ! Tu grimpes où ?");
    await byTest("conversation-send").click();
    await page.getByText("Bonjour Bastien ! Tu grimpes où ?").waitFor();
    const partnerMessages = await api(`/chats/${memberId}/messages`, { token: partner.token });
    assert(
      partnerMessages.json.messages.some((m) => m.content.includes("Tu grimpes")),
      "partner should receive the message"
    );
    const reply = await api(`/chats/${memberId}/messages`, {
      method: "POST",
      token: partner.token,
      body: { content: "À Arkose, et toi ?" }
    });
    assert(reply.status === 201, `partner reply ${reply.status}`);
    await page.getByText("À Arkose, et toi ?").waitFor({ timeout: 10_000 });
    await api(`/chats/${memberId}/messages`, { method: "POST", token: partner.token, body: { content: "On pourrait y aller ensemble." } });
    await page.getByText("On pourrait y aller ensemble.").waitFor({ timeout: 10_000 });
    await byTest("conversation-input").fill("Avec plaisir 🙂");
    await byTest("conversation-send").click();
    await api(`/chats/${memberId}/read`, { method: "POST", token: partner.token });
    await byTest("message-read-state").getByText("Lu").waitFor({ timeout: 10_000 });
    await shot("06-conversation");

    // Matches list then notifications.
    await page.getByRole("button", { name: "Retour" }).click();
    await byTest("tab-matches").click();
    await byTest(`chat-row-${partner.id}`).waitFor();
    await shot("07-matches");
    await byTest("tab-notifications").click();
    await page.getByText("Nouveau match").first().waitFor();
    await shot("08-notifications");

    // Account deletion.
    await byTest("tab-me").click();
    await shot("09-profile");
    await byTest("me-settings").click();
    await shot("10-settings");
    await byTest("settings-delete").click();
    await byTest("delete-password").fill(PASSWORD);
    await byTest("delete-confirm").click();
    await byTest("welcome-register").waitFor({ timeout: 10_000 });
    const relogin = await api("/auth/login", { method: "POST", body: { email, password: PASSWORD } });
    assert(relogin.status === 401, "deleted account must not be able to log in");
    console.log("[e2e-web] PASS register/onboarding/photo/discovery/like/match/realtime chat/read receipt/delete account");
  } catch (err) {
    await shot("failure").catch(() => undefined);
    throw err;
  } finally {
    if (partner) await api("/me", { method: "DELETE", token: partner.token, body: { password: PASSWORD } }).catch(() => undefined);
    await browser.close();
    server.close();
    fs.rmSync(photoFile, { force: true });
  }
}

main().catch((err) => {
  console.error(`[e2e-web] FAIL ${err.message}`);
  process.exit(1);
});
