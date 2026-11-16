/**
 * UI end-to-end test of the critical path on the web build, with Playwright:
 *
 *   Inscription -> création du profil (photo réelle, géolocalisation)
 *   -> découverte -> like -> match -> conversation temps réel
 *   -> signalement -> suppression du compte
 *
 * Prerequisites: a running API (with ALLOWED_ORIGINS including the web
 * origin) and a web export in ./dist (`npm run build:web`).
 *
 *   E2E_API_URL=http://localhost:18080 npm run e2e:web
 */
const fs = require("fs");
const http = require("http");
const path = require("path");
const { chromium } = require("playwright");
const { gradientPng: png } = require("../scripts/lib/png");

const API = (process.env.E2E_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PORT = Number(process.env.E2E_WEB_PORT || 8081);
const DIST = path.join(__dirname, "..", "dist");
const OUT = path.join(__dirname, "report");
const PASSWORD = "Password123";
// A remote spot so demo/seed users never interfere with the deck.
const SPOT = { latitude: -45.87, longitude: 170.5 };

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

function serveDist() {
  const types = {
    ".html": "text/html",
    ".js": "application/javascript",
    ".css": "text/css",
    ".ttf": "font/ttf",
    ".png": "image/png",
    ".json": "application/json",
    ".ico": "image/x-icon"
  };
  const server = http.createServer((req, res) => {
    const urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
    let file = path.normalize(path.join(DIST, urlPath));
    if (!file.startsWith(DIST) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
      file = path.join(DIST, "index.html");
    }
    res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(PORT, () => resolve(server)));
}

async function api(method, p, token, body) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (body instanceof FormData) payload = body;
  else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(`${API}${p}`, { method, headers, body: payload });
  const text = await res.text();
  return { status: res.status, data: text ? JSON.parse(text) : null };
}

async function createCounterpart() {
  const email = `e2e.bruno.${Date.now()}@e2e.test`;
  const auth = await api("POST", "/auth/register", null, { email, password: PASSWORD });
  assert(auth.status === 201, `counterpart register: ${auth.status}`);
  const t = auth.data.accessToken;
  await api("PUT", "/me/profile", t, {
    firstName: "Bruno",
    birthdate: "1991-08-20",
    gender: "man",
    bio: "Grimpeur, cuisinier du dimanche.",
    jobTitle: "Ébéniste",
    lookingFor: "relationship",
    interestIds: [1, 2, 3]
  });
  await api("PUT", "/me/preferences", t, { interestedIn: ["woman"], ageMin: 18, ageMax: 99, maxDistanceKm: 50 });
  await api("PUT", "/me/location", t, SPOT);
  const form = new FormData();
  form.append("photo", new Blob([png(480, 600, [120, 160, 200])], { type: "image/png" }), "p.png");
  const up = await api("POST", "/me/photos", t, form);
  assert(up.status === 201, `counterpart photo: ${up.status}`);
  return { email, token: t, id: auth.data.user.id };
}

async function main() {
  fs.mkdirSync(OUT, { recursive: true });
  assert(fs.existsSync(path.join(DIST, "index.html")), "dist/ missing: run `npm run build:web` first");
  const server = await serveDist();
  const bruno = await createCounterpart();
  const aliceEmail = `e2e.alice.${Date.now()}@e2e.test`;
  const photoPath = path.join(OUT, "alice.png");
  fs.writeFileSync(photoPath, png(480, 600, [240, 140, 90]));

  const browser = await chromium.launch();
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, geolocation: SPOT, permissions: ["geolocation"], locale: "fr-FR" });
  const page = await context.newPage();
  page.on("pageerror", (err) => console.error("[page error]", err.message));
  const shot = (name) => page.screenshot({ path: path.join(OUT, `${name}.png`) });
  const step = (msg) => console.log(`[e2e] ${msg}`);

  try {
    await page.goto(`http://localhost:${PORT}/`);
    await page.getByTestId("welcome-register").click();
    step("register");
    await page.getByTestId("register-email").fill(aliceEmail);
    await page.getByTestId("register-password").fill(PASSWORD);
    await page.getByTestId("register-adult").click();
    await shot("01-register");
    await page.getByTestId("register-submit").click();

    step("onboarding: identity");
    await page.getByTestId("onboarding-firstname").fill("Alice");
    await page.getByTestId("onboarding-birthdate").pressSequentially("14031994");
    await shot("02-onboarding-about");
    await page.getByTestId("onboarding-next").click();
    await page.getByTestId("gender-woman").click();
    await page.getByTestId("interested-man").click();
    await page.getByTestId("looking-relationship").click();
    await page.getByTestId("onboarding-next").click();

    step("onboarding: photo upload");
    await page.getByTestId("onboarding-step-2").waitFor();
    const chooser = page.waitForEvent("filechooser");
    await page.getByTestId("photo-add").click();
    await (await chooser).setFiles(photoPath);
    await page.getByTestId("photo-slot-0").waitFor({ timeout: 20000 });
    await shot("03-onboarding-photos");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: interests + location");
    await page.getByTestId("interest-hiking").click();
    await page.getByTestId("interest-cooking").click();
    await page.getByTestId("onboarding-bio").fill("Randonnées le week-end et marchés le dimanche.");
    await page.getByTestId("onboarding-next").click();
    await page.getByTestId("onboarding-location").click();

    step("discover");
    await page.getByTestId("discover-screen").waitFor({ timeout: 20000 });
    const alice = await api("POST", "/auth/login", null, { email: aliceEmail, password: PASSWORD });
    const aliceToken = alice.data.accessToken;
    const meProfile = await api("GET", "/me/profile", aliceToken);
    assert(meProfile.data.completeness.isComplete && meProfile.data.completeness.hasLocation, "profile should be complete with location");
    const liked = await api("POST", "/swipes", bruno.token, { targetUserId: alice.data.user.id, action: "like" });
    assert(liked.status === 200 && !liked.data.matched, "bruno like");

    // The deck was fetched before Bruno's like; reload it from the tab.
    await page.reload();
    await page.getByTestId("deck-top-card").waitFor({ timeout: 20000 });
    const label = await page.getByTestId("deck-top-card").getAttribute("aria-label");
    assert(label && label.startsWith("Bruno"), `expected Bruno on top, got ${label}`);
    await shot("04-discover");

    step("like -> match");
    await page.getByTestId("deck-like").click();
    await page.getByTestId("match-celebration").waitFor({ timeout: 10000 });
    await shot("05-match");
    await page.getByTestId("match-send-message").click();

    step("chat");
    await page.getByTestId("conversation-screen").waitFor();
    await page.getByTestId("composer-input").fill("Salut Bruno ! Tu grimpes où ?");
    await page.getByTestId("composer-send").click();
    await page.getByText("Salut Bruno ! Tu grimpes où ?").waitFor();
    const matches = await api("GET", "/matches", bruno.token);
    const matchId = matches.data.matches[0].id;
    const received = await api("GET", `/matches/${matchId}/messages`, bruno.token);
    assert(received.data.messages[0].body === "Salut Bruno ! Tu grimpes où ?", "bruno must receive the message");

    await api("POST", `/matches/${matchId}/read`, bruno.token);
    await api("POST", `/matches/${matchId}/messages`, bruno.token, { body: "À la salle de Dunedin, et toi ?" });
    await page.getByText("À la salle de Dunedin, et toi ?").waitFor({ timeout: 10000 });
    step("realtime reply received");
    await page.waitForTimeout(600);
    await shot("06-conversation");

    step("report");
    await page.getByTestId("conversation-safety").click();
    await page.getByTestId("safety-report").click();
    await page.getByTestId("report-reason-harassment").click();
    await shot("07-report");
    await page.getByTestId("report-submit").click();
    await page.getByTestId("toast").waitFor();
    const afterReport = await api("GET", `/matches/${matchId}/messages`, bruno.token);
    assert(afterReport.status === 404, "reported user must lose access to the conversation");

    step("delete account");
    await page.getByTestId("tab-me").click();
    await page.getByTestId("me-screen").waitFor();
    await page.waitForTimeout(3500);
    await shot("08-me");
    await page.getByTestId("me-settings").click();
    await page.getByTestId("settings-delete").click();
    await page.getByTestId("delete-password").fill(PASSWORD);
    await page.getByTestId("delete-confirm").click();
    await page.getByTestId("welcome-screen").waitFor({ timeout: 10000 });
    const relogin = await api("POST", "/auth/login", null, { email: aliceEmail, password: PASSWORD });
    assert(relogin.status === 401, "deleted account must not log in");
    await shot("09-deleted");
    step("OK — critical flow passed");
  } catch (err) {
    await shot("failure").catch(() => undefined);
    throw err;
  } finally {
    await api("DELETE", "/me", bruno.token, { password: PASSWORD }).catch(() => undefined);
    await browser.close();
    server.close();
  }
}

main().catch((err) => {
  console.error(`[e2e] FAILED: ${err.stack || err.message}`);
  process.exit(1);
});
