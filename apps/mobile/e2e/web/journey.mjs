// Browser end-to-end test of the critical journey, driven through the real UI (web build):
// sign up → onboarding (profile, preferences, location, photo) → discover → like → match → chat,
// including live delivery of a message sent by the other member.
//
// Prerequisites: a running API (MOBILE_E2E_API_URL, default http://localhost:18080) started with
// ALLOWED_ORIGINS=http://localhost:19006, and a web export built with
// EXPO_PUBLIC_API_URL=<same API url> npx expo export --platform web --output-dir <WEB_DIST>.
import { createServer } from "node:http";
import { existsSync, mkdtempSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { extname, join, normalize } from "node:path";
import { createRequire } from "node:module";
import { chromium } from "playwright-core";

const require = createRequire(import.meta.url);
const { PASSWORD, assert, createMember, request, samplePng, uniqueEmail } = require("../../scripts/lib/api.js");

const WEB_DIST = process.env.WEB_DIST || "/tmp/amora-web";
const PORT = Number(process.env.WEB_PORT || 19006);
const MIME = { ".html": "text/html", ".js": "text/javascript", ".json": "application/json", ".png": "image/png", ".ico": "image/x-icon", ".css": "text/css" };

function serveStatic() {
  const server = createServer((req, res) => {
    const url = new URL(req.url, "http://localhost");
    let file = normalize(join(WEB_DIST, decodeURIComponent(url.pathname)));
    if (!file.startsWith(WEB_DIST) || !existsSync(file) || !statSync(file).isFile()) {
      file = join(WEB_DIST, "index.html"); // single-page app fallback
    }
    res.writeHead(200, { "Content-Type": MIME[extname(file)] || "application/octet-stream" });
    res.end(readFileSync(file));
  });
  return new Promise((resolve) => server.listen(PORT, () => resolve(server)));
}

const step = (name) => console.log(`[e2e-web] ${name}`);

async function main() {
  assert(existsSync(join(WEB_DIST, "index.html")), `web export not found in ${WEB_DIST}`);
  const server = await serveStatic();
  const browser = await chromium.launch({ executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH || undefined, args: ["--no-sandbox"] });
  const origin = { latitude: 46.2 + Math.random(), longitude: 6.1 + Math.random() };
  const photoPath = join(mkdtempSync(join(tmpdir(), "amora-")), "me.png");
  writeFileSync(photoPath, samplePng(96));

  try {
    // Casey exists beforehand and is near the new member.
    const casey = await createMember("web_casey", { name: "Casey", gender: "woman", interestedIn: ["man"], origin });

    const context = await browser.newContext({
      baseURL: `http://localhost:${PORT}`,
      permissions: ["geolocation"],
      geolocation: { latitude: origin.latitude, longitude: origin.longitude },
      viewport: { width: 420, height: 860 }
    });
    const page = await context.newPage();
    page.setDefaultTimeout(15_000);
    page.on("pageerror", (e) => console.error(`[browser error] ${e.message}`));

    let alexId = null;
    page.on("response", async (response) => {
      if (response.url().endsWith("/auth/register") && response.status() === 201) {
        alexId = (await response.json()).user.id;
      }
    });

    const email = uniqueEmail("web_alex");
    await page.goto("/");
    await page.getByTestId("auth-screen").waitFor();
    step("auth screen visible");

    // Client-side validation and server-side rejection are both reachable from the UI.
    await page.getByTestId("auth-to-register").click();
    await page.getByTestId("auth-email").fill("not-an-email");
    await page.getByTestId("auth-password").fill("Password123");
    await page.getByTestId("auth-submit").click();
    await page.getByText("That email address does not look right.").waitFor();

    await page.getByTestId("auth-email").fill(email);
    await page.getByTestId("auth-password").fill(PASSWORD);
    await page.getByTestId("auth-submit").click();
    await page.getByTestId("onboarding-screen").waitFor();
    step("registered, onboarding started");

    // Step 1: basics (an under-18 birth date is refused before anything is sent).
    await page.getByTestId("profile-first-name").fill("Alex");
    await page.getByTestId("profile-day").fill("14");
    await page.getByTestId("profile-month").fill("3");
    await page.getByTestId("profile-year").fill(String(new Date().getFullYear() - 12));
    await page.getByTestId("profile-gender-man").click();
    await page.getByTestId("onboarding-next").click();
    await page.getByText("You must be at least 18 years old to use this app.").waitFor();
    await page.getByTestId("profile-year").fill(String(new Date().getFullYear() - 30));
    await page.getByTestId("onboarding-next").click();

    // Step 2: preferences.
    await page.getByTestId("pref-gender-woman").waitFor();
    await page.getByTestId("pref-gender-man").click();
    await page.getByTestId("pref-gender-nonbinary").click();
    await page.getByTestId("onboarding-next").click();

    // Step 3: story + location.
    await page.getByTestId("profile-bio").fill("Climber, coffee fan.");
    await page.getByText("Travel", { exact: true }).click();
    await page.getByTestId("onboarding-location").click();
    await page.getByText("Location saved ✓").waitFor();
    await page.getByTestId("onboarding-next").click();

    // Step 4: photo, through a real file chooser.
    await page.getByTestId("photo-add").waitFor();
    const [chooser] = await Promise.all([page.waitForEvent("filechooser"), page.getByTestId("photo-add").click()]);
    await chooser.setFiles(photoPath);
    await page.getByTestId("photo-0").waitFor();
    step("profile created with a photo");
    await page.getByTestId("onboarding-finish").click();
    await page.getByTestId("discover-screen").waitFor();
    assert(alexId, "register response was not captured");

    // Casey likes Alex first, so Casey is the top card and the like below completes a match.
    let res = await request("/swipes", { token: casey.token, json: { targetId: alexId, action: "like" } });
    assert(res.status === 200 && res.payload.matched === false, `casey like: ${res.status}`);

    await page.reload();
    // Memory-only session on web: sign back in to prove the login path too.
    await page.getByTestId("auth-email").fill(email);
    await page.getByTestId("auth-password").fill(PASSWORD);
    await page.getByTestId("auth-submit").click();
    await page.getByTestId("discover-screen").waitFor();
    await page.getByText("Casey, 30").first().waitFor();
    step("discovery shows Casey");

    await page.getByTestId("discover-like").click();
    await page.getByTestId("match-modal").waitFor();
    await page.getByText("You and Casey liked each other.").waitFor();
    step("match modal");

    await page.getByTestId("match-message").click();
    await page.getByTestId("chat-input").waitFor();
    await page.getByTestId("chat-input").fill("Hello Casey!");
    await page.getByTestId("chat-send").click();
    await page.getByText("Hello Casey!").waitFor();
    step("message sent from the UI");

    // Casey answers from another client: it must arrive live, without a reload.
    const matchRes = await request("/conversations", { token: casey.token });
    const matchId = matchRes.payload.conversations[0].matchId;
    const seen = await request(`/conversations/${matchId}/messages`, { token: casey.token });
    assert(seen.payload.messages.some((m) => m.body === "Hello Casey!"), "casey should receive alex's message");
    res = await request(`/conversations/${matchId}/messages`, { token: casey.token, json: { body: "Hi Alex, great to meet you" } });
    assert(res.status === 201, `casey reply: ${res.status}`);
    await page.getByText("Hi Alex, great to meet you").waitFor();
    step("live message received");

    // Matches tab, profile tab, and account deletion through the UI.
    await page.getByRole("button", { name: "Back" }).click().catch(() => undefined);
    await page.getByTestId("tab-profile").click();
    await page.getByTestId("profile-screen").waitFor();
    await page.getByTestId("menu-settings").click();
    await page.getByTestId("settings-screen").waitFor();
    await page.getByTestId("settings-delete").click();
    await page.getByTestId("dialog-Continue").click();
    await page.getByTestId("delete-password").fill(PASSWORD);
    await page.getByTestId("delete-confirm").click();
    await page.getByTestId("auth-screen").waitFor();
    res = await request("/auth/login", { json: { email, password: PASSWORD } });
    assert(res.status === 401, "deleted account must not log in");
    step("account deleted");

    await request("/me", { token: casey.token, method: "DELETE", json: { password: PASSWORD } });
    console.log("[e2e-web] PASS");
  } finally {
    await browser.close();
    server.close();
  }
}

main().catch((err) => {
  console.error(`[e2e-web] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
