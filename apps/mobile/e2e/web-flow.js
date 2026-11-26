// UI end-to-end test of the critical journey on the web build, driven by Playwright.
//
//   API_URL  (default http://localhost:18080) - running API
//   WEB_URL  (default http://localhost:8081)  - served `npm run build:web` output (or `expo start --web`)
//
// Journey: register -> onboarding (identity, preferences, photo upload) -> Likes -> like back
// -> match modal -> chat -> realtime message from the other member -> read receipt -> account deletion.
const fs = require("fs");
const path = require("path");
const zlib = require("zlib");
const { chromium } = require("playwright");

const API = (process.env.API_URL || "http://localhost:18080").replace(/\/+$/, "");
const WEB = (process.env.WEB_URL || "http://localhost:8081").replace(/\/+$/, "");
const OUT = path.join(__dirname, ".results");
const PASSWORD = "Password123";

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function call(p, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${p}`, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error(`${method} ${p} -> ${res.status} ${text}`);
  return data;
}

function png(width, height, rgb) {
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
    const body = Buffer.concat([Buffer.from(type), data]);
    const sum = Buffer.alloc(4);
    sum.writeUInt32BE(crc(body));
    return Buffer.concat([len, body, sum]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  const rows = [];
  for (let y = 0; y < height; y++) {
    const row = Buffer.alloc(1 + width * 3);
    for (let x = 0; x < width; x++) {
      const t = y / height;
      row[1 + x * 3] = Math.round(rgb[0] * (1 - t) + 250 * t);
      row[2 + x * 3] = Math.round(rgb[1] * (1 - t) + 200 * t);
      row[3 + x * 3] = Math.round(rgb[2] * (1 - t) + 150 * t);
    }
    rows.push(row);
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(Buffer.concat(rows))),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

function birthdate(age) {
  const d = new Date();
  d.setFullYear(d.getFullYear() - age);
  d.setDate(d.getDate() - 20);
  return d;
}

/** The other member is prepared through the API: complete profile, then likes the UI user. */
async function createPartner(likeUserId) {
  const email = `e2e_partner_${Date.now()}@e2e.test`;
  const auth = await call("/auth/register", { method: "POST", body: { email, password: PASSWORD, acceptedTerms: true } });
  const token = auth.accessToken;
  await call("/profile", {
    method: "PUT",
    token,
    body: { firstName: "Clara", birthdate: birthdate(29).toISOString().slice(0, 10), gender: "woman", bio: "Toujours partante pour un café.", jobTitle: "Illustratrice", city: "Paris 11e", interests: ["coffee", "drawing"] }
  });
  await call("/profile/preferences", { method: "PUT", token, body: { interestedIn: ["man"], minAge: 18, maxAge: 60, maxDistanceKm: 50 } });
  const form = new FormData();
  form.append("photo", new Blob([png(480, 600, [116, 80, 138])], { type: "image/png" }), "clara.png");
  await call("/photos", { method: "POST", token, form });
  await call("/swipes", { method: "POST", token, body: { targetUserId: likeUserId, action: "like" } });
  return { token, id: auth.user.id };
}

async function main() {
  fs.mkdirSync(OUT, { recursive: true });
  const browser = await chromium.launch();
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 1, locale: "fr-FR" });
  const page = await context.newPage();
  page.setDefaultTimeout(15_000);
  const shot = (name) => page.screenshot({ path: path.join(OUT, `${name}.png`) });
  const email = `e2e_ui_${Date.now()}@e2e.test`;
  let partner = null;
  let step = "start";

  try {
    step = "welcome";
    await page.goto(WEB);
    await page.getByTestId("welcome-screen").waitFor();
    await shot("01-welcome");

    step = "register";
    await page.getByTestId("welcome-register").click();
    await page.getByTestId("register-email").fill(email);
    await page.getByTestId("register-password").fill(PASSWORD);
    await page.getByTestId("register-terms").click();
    await shot("02-register");
    await page.getByTestId("register-submit").click();

    step = "onboarding-identity";
    await page.getByTestId("onboarding-screen").waitFor();
    await page.getByTestId("onboarding-first-name").fill("Julien");
    const b = birthdate(31);
    await page.getByTestId("birthdate-day").fill(String(b.getDate()).padStart(2, "0"));
    await page.getByTestId("birthdate-month").fill(String(b.getMonth() + 1).padStart(2, "0"));
    await page.getByTestId("birthdate-year").fill(String(b.getFullYear()));
    await shot("03-onboarding-identity");
    await page.getByTestId("onboarding-next").click();

    step = "onboarding-gender";
    await page.getByTestId("gender-man").click();
    await page.getByTestId("interested-woman").click();
    await page.getByTestId("onboarding-next").click();

    step = "onboarding-photos";
    await page.getByTestId("photo-add").waitFor();
    const photoFile = path.join(OUT, "julien.png");
    fs.writeFileSync(photoFile, png(600, 750, [212, 83, 61]));
    const [chooser] = await Promise.all([page.waitForEvent("filechooser"), page.getByTestId("photo-add").click()]);
    await chooser.setFiles(photoFile);
    await page.getByTestId("photo-slot-0").waitFor();
    await page.waitForTimeout(500);
    await shot("04-onboarding-photos");
    await page.getByTestId("onboarding-next").click();

    step = "onboarding-about";
    await page.getByTestId("onboarding-bio").fill("Cuisinier du dimanche, randonneur le samedi.");
    await page.getByTestId("interest-hiking").click();
    await page.getByTestId("interest-cooking").click();
    await shot("05-onboarding-about");
    await page.getByTestId("onboarding-next").click();

    step = "onboarding-location";
    await page.getByTestId("onboarding-skip-location").click();
    await page.getByTestId("discover-screen").waitFor();
    await page.waitForTimeout(800);
    await shot("06-discover");

    step = "partner-likes";
    const me = await call("/auth/login", { method: "POST", body: { email, password: PASSWORD } });
    partner = await createPartner(me.user.id);

    step = "deck-interaction";
    if (await page.getByTestId("swipe-card").count()) {
      await page.getByTestId("deck-pass").click();
      await page.waitForTimeout(500);
    }

    step = "likes-tab";
    await page.getByTestId("tab-likes").click();
    await page.getByTestId("likes-screen").waitFor();
    await page.getByTestId(`like-tile-${partner.id}`).waitFor();
    await page.waitForTimeout(400);
    await shot("07-likes");
    await page.getByTestId(`like-tile-${partner.id}`).click();
    await page.getByTestId("profile-detail").waitFor();
    await page.waitForTimeout(400);
    await shot("08-profile-detail");

    step = "match";
    await page.getByTestId("profile-like").click();
    await page.getByTestId("match-modal").waitFor();
    await page.waitForTimeout(600);
    await shot("09-match");
    await page.getByTestId("match-modal-message").click();

    step = "chat";
    await page.getByTestId("chat-screen").waitFor();
    await page.getByTestId("chat-input").fill("Salut Clara ! Un café cette semaine ?");
    await page.getByTestId("chat-send").click();
    await page.getByText("Salut Clara ! Un café cette semaine ?").waitFor();

    const conversations = await call("/conversations", { token: partner.token });
    assert(conversations.conversations.length === 1, "partner should see the conversation");
    const conversationId = conversations.conversations[0].id;
    const thread = await call(`/conversations/${conversationId}/messages`, { token: partner.token });
    assert(thread.messages[0].body === "Salut Clara ! Un café cette semaine ?", "partner should receive the message");

    step = "realtime";
    await call(`/conversations/${conversationId}/read`, { method: "POST", token: partner.token });
    await page.getByText("· Lu").waitFor();
    await call(`/conversations/${conversationId}/messages`, { method: "POST", token: partner.token, body: { body: "Avec plaisir ! Jeudi ?" } });
    await page.getByText("Avec plaisir ! Jeudi ?").waitFor();
    await page.waitForTimeout(300);
    await shot("10-chat");

    step = "messages-tab";
    await page.getByTestId("chat-back").click();
    await page.getByTestId("tab-messages").click();
    await page.getByTestId("messages-screen").waitFor();
    await page.waitForTimeout(500);
    await shot("11-messages");

    step = "notifications";
    await page.getByTestId("tab-discover").click();
    await page.getByTestId("open-notifications").click();
    await page.getByTestId("notifications-screen").waitFor();
    await page.getByTestId("notification-match").first().waitFor();
    await shot("12-notifications");
    await page.getByTestId("header-back").click();

    step = "profile";
    await page.getByTestId("tab-me").click();
    await page.getByTestId("me-screen").waitFor();
    await page.waitForTimeout(400);
    await shot("13-me");
    await page.getByTestId("me-settings").click();
    await page.getByTestId("settings-screen").waitFor();
    await shot("14-settings");

    step = "delete-account";
    await page.getByTestId("settings-delete").click();
    await page.getByTestId("delete-password").fill(PASSWORD);
    await page.getByTestId("delete-confirm").click();
    await page.getByTestId("welcome-screen").waitFor();
    const relogin = await fetch(`${API}/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password: PASSWORD }) });
    assert(relogin.status === 401, "deleted account must not log in");
    const after = await call("/conversations", { token: partner.token });
    assert(after.conversations.length === 0, "conversation must disappear after account deletion");

    console.log("[e2e-web] PASS");
  } catch (err) {
    await shot("failure").catch(() => undefined);
    console.error(`[e2e-web] FAIL at step "${step}": ${err.message}`);
    process.exitCode = 1;
  } finally {
    if (partner) await call("/me", { method: "DELETE", token: partner.token, body: { password: PASSWORD } }).catch(() => undefined);
    await browser.close();
  }
}

main();
