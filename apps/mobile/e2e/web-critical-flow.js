/**
 * Browser E2E of the critical journey on the web build, with two real users:
 * register → onboarding (photo upload, location) → discovery → like → match →
 * conversation in realtime.
 *
 * Prerequisites: a running API (E2E_API_URL, default http://localhost:18080,
 * with ALLOWED_ORIGINS including the web origin) and a web export built with
 * EXPO_PUBLIC_API_URL pointing at it:
 *
 *   EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
 *   npm run e2e:web
 */
const fs = require("fs");
const http = require("http");
const os = require("os");
const path = require("path");
const { chromium } = require("playwright");
const { png } = require("../scripts/lib/png");

const DIST = path.resolve(__dirname, "../dist");
const PORT = Number(process.env.E2E_WEB_PORT || 8081);
const ORIGIN = `http://localhost:${PORT}`;
const API = (process.env.E2E_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const ARTIFACTS = path.resolve(__dirname, ".artifacts");
const PASSWORD = "E2eTest1234";

function serveDist() {
  const types = {
    ".html": "text/html",
    ".js": "application/javascript",
    ".png": "image/png",
    ".json": "application/json",
    ".ico": "image/x-icon"
  };
  const server = http.createServer((req, res) => {
    const urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
    let file = path.join(DIST, urlPath);
    if (!file.startsWith(DIST) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
      file = path.join(DIST, "index.html");
    }
    res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(PORT, () => resolve(server)));
}

async function shot(page, name) {
  await page.waitForLoadState("networkidle", { timeout: 5000 }).catch(() => undefined);
  await page.waitForTimeout(400);
  await page.screenshot({ path: path.join(ARTIFACTS, `${name}.png`) });
}

async function onboard(page, { email, name, gender, audience, birth, photoPath, label }) {
  await page.goto(ORIGIN);
  await page.getByTestId("welcome-register").click();
  await page.getByTestId("register-email").fill(email);
  await page.getByTestId("register-password").fill(PASSWORD);
  await page.getByTestId("register-confirm").fill(PASSWORD);
  await page.getByTestId("register-submit").click();

  await page.getByTestId("onboarding-basics").waitFor();
  await page.getByTestId("onboarding-name").fill(name);
  await page.getByTestId("birth-day").fill(birth[0]);
  await page.getByTestId("birth-month").fill(birth[1]);
  await page.getByTestId("birth-year").fill(birth[2]);
  await page.getByTestId(`gender-${gender}`).click();
  if (label === "alice") await shot(page, "01-onboarding-basics");
  await page.getByTestId("onboarding-next").click();

  await page.getByTestId("onboarding-looking").waitFor();
  await page.getByTestId(`audience-${audience}`).click();
  await page.getByTestId("onboarding-next").click();

  await page.getByTestId("onboarding-photos").waitFor();
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("photo-add").click();
  await (await chooser).setFiles(photoPath);
  await page.getByTestId("photo-slot-0").waitFor({ timeout: 20_000 });
  if (label === "alice") await shot(page, "02-onboarding-photos");
  await page.getByTestId("onboarding-next").click();

  await page.getByTestId("onboarding-about").waitFor();
  await page.getByTestId("onboarding-bio").fill(`Bonjour, je suis ${name}. Profil de test E2E.`);
  await page.getByTestId("interest-hiking").click();
  await page.getByTestId("onboarding-next").click();

  await page.getByTestId("onboarding-location").waitFor();
  await page.getByTestId("onboarding-locate").click();
  await page.getByTestId("onboarding-finish").click({ timeout: 20_000 });
  await page.getByTestId("discover-screen").waitFor();
}

async function main() {
  if (!fs.existsSync(path.join(DIST, "index.html"))) {
    throw new Error("dist/ is missing: run `EXPO_PUBLIC_API_URL=... npm run build:web` first");
  }
  fs.mkdirSync(ARTIFACTS, { recursive: true });
  const health = await fetch(`${API}/health`);
  if (!health.ok) {
    throw new Error(`API not healthy at ${API}`);
  }

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "hearth-e2e-"));
  const photoA = path.join(tmp, "alice.png");
  const photoB = path.join(tmp, "bruno.png");
  fs.writeFileSync(photoA, png(480, 600, [222, 128, 98]));
  fs.writeFileSync(photoB, png(480, 600, [96, 128, 214]));

  // A unique, remote area keeps this run isolated from other data.
  const base = { latitude: -45 + Math.random() * 10, longitude: -140 + Math.random() * 10 };
  const run = Date.now();
  const server = await serveDist();
  const browser = await chromium.launch();
  const contextOptions = {
    viewport: { width: 420, height: 860 },
    permissions: ["geolocation"],
    locale: "fr-FR"
  };
  const ctxA = await browser.newContext({ ...contextOptions, geolocation: base });
  const ctxB = await browser.newContext({ ...contextOptions, geolocation: { latitude: base.latitude + 0.03, longitude: base.longitude } });
  const alice = await ctxA.newPage();
  const bruno = await ctxB.newPage();
  for (const [label, page] of [
    ["alice", alice],
    ["bruno", bruno]
  ]) {
    page.on("pageerror", (err) => console.error(`[${label}] page error: ${err.message}`));
  }
  const accounts = [];

  try {
    const aliceEmail = `e2e_alice_${run}@hearth.test`;
    const brunoEmail = `e2e_bruno_${run}@hearth.test`;
    accounts.push(aliceEmail, brunoEmail);
    await onboard(alice, {
      email: aliceEmail,
      name: "Alice",
      gender: "woman",
      audience: "man",
      birth: ["12", "05", "1994"],
      photoPath: photoA,
      label: "alice"
    });
    console.log("[e2e] alice onboarded");
    await onboard(bruno, {
      email: brunoEmail,
      name: "Bruno",
      gender: "man",
      audience: "woman",
      birth: ["02", "03", "1992"],
      photoPath: photoB,
      label: "bruno"
    });
    console.log("[e2e] bruno onboarded");

    // Alice sees Bruno and likes him.
    // Sessions are memory-only on web, so navigate instead of reloading:
    // returning to an empty deck looks for new people.
    await alice.getByTestId("tab-matches").click();
    await alice.getByTestId("tab-discover").click();
    await alice.getByText("Bruno", { exact: false }).first().waitFor({ timeout: 20_000 });
    await shot(alice, "03-discover");
    await alice.getByTestId("swipe-like").click();
    await alice.waitForTimeout(800);

    // Bruno sees Alice, opens her profile, likes back: it's a match.
    await bruno.getByTestId("tab-matches").click();
    await bruno.getByTestId("tab-discover").click();
    await bruno.getByText("Alice", { exact: false }).first().waitFor({ timeout: 20_000 });
    await bruno.getByTestId("card-open").click();
    await bruno.getByTestId("profile-detail").waitFor();
    await shot(bruno, "04-profile-detail");
    await bruno.getByTestId("profile-close").click();
    await bruno.getByTestId("swipe-like").click();
    await bruno.getByTestId("match-celebration").waitFor({ timeout: 15_000 });
    await shot(bruno, "05-match");
    console.log("[e2e] match created");

    // Bruno writes; Alice receives in realtime.
    await bruno.getByTestId("match-message").click();
    await bruno.getByTestId("chat-screen").waitFor();
    await bruno.getByTestId("chat-input").fill("Salut Alice ! On se fait une rando ?");
    await bruno.getByTestId("chat-send").click();
    await bruno.getByTestId("chat-screen").getByText("Salut Alice ! On se fait une rando ?").waitFor();

    await alice.getByTestId("tab-matches").click();
    await alice.getByTestId("matches-screen").waitFor();
    await alice.getByText("Salut Alice ! On se fait une rando ?").first().waitFor({ timeout: 15_000 });
    await shot(alice, "06-matches");
    await alice.getByText("Bruno").first().click();
    await alice.getByTestId("chat-screen").waitFor();
    await alice.getByTestId("chat-input").fill("Avec plaisir 🙂");
    await alice.getByTestId("chat-send").click();

    // Realtime delivery + read receipt on Bruno's side, no reload.
    await bruno.getByTestId("chat-screen").getByText("Avec plaisir 🙂").waitFor({ timeout: 15_000 });
    await bruno
      .getByTestId("read-receipt")
      .filter({ hasText: "Lu" })
      .waitFor({ timeout: 15_000 })
      .catch(() => undefined);
    await shot(bruno, "07-chat");
    console.log("[e2e] realtime chat OK");

    // Activity tab shows the match notification.
    await alice.getByTestId("chat-screen").getByLabel("Retour").click();
    await alice.getByTestId("tab-activity").click();
    await alice.getByTestId("notification-match").first().waitFor({ timeout: 10_000 });
    await shot(alice, "08-activity");
    console.log("[e2e] notifications OK");

    console.log(`[e2e] PASSED — screenshots in ${ARTIFACTS}`);
  } catch (error) {
    await shot(alice, "failure-alice").catch(() => undefined);
    await shot(bruno, "failure-bruno").catch(() => undefined);
    throw error;
  } finally {
    await browser.close();
    server.close();
    // Remove the accounts through the public API.
    for (const email of accounts) {
      const login = await fetch(`${API}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password: PASSWORD })
      });
      if (login.ok) {
        const { accessToken } = await login.json();
        await fetch(`${API}/me`, {
          method: "DELETE",
          headers: { "Content-Type": "application/json", Authorization: `Bearer ${accessToken}` },
          body: JSON.stringify({ password: PASSWORD })
        });
      }
    }
  }
}

main().catch((error) => {
  console.error(`[e2e] FAILED: ${error.message}`);
  process.exit(1);
});
