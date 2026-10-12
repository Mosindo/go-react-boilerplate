/* eslint-disable no-console */
// Browser end-to-end test of the critical journey, against a real API:
//   register -> onboarding (profile, photo, preferences) -> discover -> like -> match -> chat (realtime)
//
// Usage: E2E_API_URL=http://localhost:18080 npm run e2e:web
// The API must run with ALLOWED_ORIGINS including the web origin used here (default http://localhost:19006).
const fs = require("fs");
const http = require("http");
const path = require("path");
const { execFileSync } = require("child_process");
const { chromium } = require("playwright-core");

const API = process.env.E2E_API_URL || "http://localhost:18080";
const PORT = Number(process.env.E2E_WEB_PORT || 19006);
const DIST = process.env.E2E_WEB_DIST || path.join(__dirname, "..", "..", "dist-e2e");
const PASSWORD = "Password123";
const JPEG = Buffer.from(
  "/9j/2wCEAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9PjsBCgsLDg0OHBAQHDsoIig7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O//AABEIAFAAQAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/AGFaaVqUrTStbpn08mRFaaVqYrTStUmYSZCVppWpitNK1SZjJkJWmlamK00rVJmMmaJWmlamK00rXKmdsmQlaaVqYrTStUmYyZCVpCtSlaaVqkzCTIitNK1KVppWqTMZM0itNK1KVppWuVM7pMiK00rUxWmlapMwkyErTStTFaaVqkzGTIStNK1MVppWqTMZM0StNK1MVppWuRM7pMhK00rUxWmlapMwkyErSFalK00rVpmMmRFaaVqUrTStUmYSZpFaaVqUrTStciZ3SZEVppWpitNK1SZjJkJWmlamK00rVJmEmQlaaVqYrTStUmYyZ//Z",
  "base64"
);

const stamp = Date.now();
const anaEmail = `ana-${stamp}@e2e.invalid`;
const benEmail = `ben-${stamp}@e2e.invalid`;

async function api(method, route, token, body) {
  const res = await fetch(`${API}${route}`, {
    method,
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined
  });
  const text = await res.text();
  if (!res.ok) throw new Error(`${method} ${route} -> ${res.status} ${text}`);
  return text ? JSON.parse(text) : null;
}

async function createBen() {
  const reg = await api("POST", "/auth/register", null, { email: benEmail, password: PASSWORD });
  const token = reg.accessToken;
  await api("PUT", "/me/profile", token, {
    firstName: "Ben",
    birthDate: "1992-04-04",
    gender: "man",
    bio: "Salut !",
    occupation: "Chef",
    city: "Paris",
    interests: ["music"]
  });
  await api("PUT", "/me/preferences", token, { interestedIn: ["woman"], ageMin: 18, ageMax: 60, maxDistanceKm: 20000 });
  const form = new FormData();
  form.append("file", new Blob([JPEG], { type: "image/jpeg" }), "ben.jpg");
  const up = await fetch(`${API}/me/photos`, { method: "POST", headers: { Authorization: `Bearer ${token}` }, body: form });
  if (!up.ok) throw new Error(`photo upload failed: ${up.status}`);
  return { token, id: reg.user.id };
}

function serveStatic() {
  const types = {
    ".html": "text/html",
    ".js": "text/javascript",
    ".css": "text/css",
    ".ttf": "font/ttf",
    ".png": "image/png",
    ".json": "application/json"
  };
  const server = http.createServer((req, res) => {
    const url = decodeURIComponent((req.url || "/").split("?")[0]);
    let file = path.join(DIST, url);
    if (!file.startsWith(DIST) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) file = path.join(DIST, "index.html");
    res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(PORT, () => resolve(server)));
}

function step(name) {
  console.log(`  ✓ ${name}`);
}

async function main() {
  console.log("Building web bundle…");
  execFileSync("npx", ["expo", "export", "--platform", "web", "--output-dir", DIST], {
    stdio: "inherit",
    env: { ...process.env, EXPO_PUBLIC_API_URL: API },
    shell: process.platform === "win32"
  });

  const server = await serveStatic();
  const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || undefined;
  const browser = await chromium.launch({ executablePath, args: ["--no-sandbox"] });
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, locale: "fr-FR" });
  const page = await context.newPage();
  page.setDefaultTimeout(20_000);
  const problems = [];
  page.on("pageerror", (e) => problems.push(String(e)));

  const ben = await createBen();

  try {
    await page.goto(`http://localhost:${PORT}`);

    // 1. Registration
    await page.getByRole("button", { name: "Créer un compte" }).click();
    await page.getByLabel("E-mail").fill(anaEmail);
    await page.getByLabel("Mot de passe", { exact: true }).fill(PASSWORD);
    await page.getByLabel("Confirmer le mot de passe").fill(PASSWORD);
    await page.getByRole("button", { name: "Créer mon compte" }).click();
    step("registered");

    // 2. Onboarding: basics (a minor is refused first)
    await page.getByLabel("Prénom").fill("Ana");
    await page.getByLabel("Jour").fill("10");
    await page.getByLabel("Mois").fill("2");
    await page.getByLabel("Année").fill(String(new Date().getFullYear() - 15));
    await page.getByRole("button", { name: "Femme" }).click();
    await page.getByRole("button", { name: "Continuer" }).click();
    await page.getByText("au moins 18 ans").waitFor();
    step("minor refused");
    await page.getByLabel("Année").fill("1994");
    await page.getByRole("button", { name: "Continuer" }).click();

    // 3. About
    await page.getByLabel("Bio").fill("Amatrice de randonnées et de bons cafés.");
    await page.getByRole("button", { name: "Voyage" }).click();
    await page.getByRole("button", { name: "Continuer" }).click();

    // 4. Photos: continuing without a photo is refused, then upload through the file chooser
    await page.getByRole("button", { name: "Continuer" }).click();
    await page.getByText("Ajoutez au moins une photo").waitFor();
    const [chooser] = await Promise.all([
      page.waitForEvent("filechooser"),
      page.getByRole("button", { name: "Ajouter une photo" }).first().click()
    ]);
    await chooser.setFiles({ name: "ana.jpg", mimeType: "image/jpeg", buffer: JPEG });
    await page.getByText("1/6 photos").waitFor();
    step("photo uploaded");

    // Ben (already a member) likes Ana now that her profile exists, so he is ranked first in her feed.
    const anaLogin = await api("POST", "/auth/login", null, { email: anaEmail, password: PASSWORD });
    await api("POST", "/swipes", ben.token, { targetId: anaLogin.user.id, action: "like" });
    step("Ben liked Ana");
    await page.getByRole("button", { name: "Continuer" }).click();

    // 5. Preferences
    await page.getByRole("button", { name: "Homme" }).click();
    await page.getByRole("button", { name: "Partout" }).click();
    await page.getByRole("button", { name: "Terminer" }).click();
    step("onboarding complete");

    await page.getByRole("button", { name: "Aimer Ben" }).click({ timeout: 30_000 });
    await page.getByText("C'est un match !").waitFor();
    step("match celebrated");

    // 7. Conversation, with realtime delivery of Ben's reply
    await page.getByRole("button", { name: "Écrire à Ben" }).click();
    await page.getByLabel("Votre message").fill("Salut Ben, ravie de te rencontrer !");
    await page.getByRole("button", { name: "Envoyer" }).click();
    await page.getByText("Salut Ben, ravie de te rencontrer !").waitFor();
    step("message sent");

    const matches = await api("GET", "/matches", ben.token);
    await api("POST", `/matches/${matches.items[0].id}/messages`, ben.token, { body: "Moi aussi, quelle belle surprise 😊" });
    await page.getByText("Moi aussi, quelle belle surprise").waitFor({ timeout: 10_000 });
    step("realtime reply received");

    if (problems.length) throw new Error(`page errors: ${problems.join(" | ")}`);
    console.log("E2E OK");
  } catch (error) {
    await page.screenshot({ path: path.join(__dirname, "failure.png"), fullPage: true }).catch(() => undefined);
    console.error("E2E FAILED:", error.message);
    process.exitCode = 1;
  } finally {
    await browser.close();
    server.close();
  }
}

main();
