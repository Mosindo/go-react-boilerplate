// Browser E2E of the critical path against the exported web build and a live API:
// sign up -> onboarding (with a real photo upload) -> discovery -> like -> match
// -> conversation -> realtime reply -> notifications.
//
//   npm run build:web && npx serve dist -l 8099   (any static server works)
//   WEB_URL=http://localhost:8099 API_URL=http://localhost:18080 npm run e2e:web
import { createRequire } from "node:module";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { chromium } from "playwright-core";

const require = createRequire(import.meta.url);
const WEB_URL = process.env.WEB_URL ?? "http://localhost:8099";
process.env.MOBILE_E2E_API_URL = process.env.API_URL ?? process.env.MOBILE_E2E_API_URL ?? "http://localhost:18080";
const { PASSWORD, createMember, makePng, request, uniqueEmail } = require("../../scripts/lib/dating-api.js");

const shotsDir = process.env.SCREENSHOT_DIR;
if (shotsDir) fs.mkdirSync(shotsDir, { recursive: true });
const photoPath = path.join(os.tmpdir(), `lueur-e2e-${Date.now()}.png`);
fs.writeFileSync(photoPath, makePng(600, 800));

const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH || undefined,
  args: ["--no-sandbox"]
});
const page = await (await browser.newContext({ viewport: { width: 420, height: 860 }, locale: "fr-FR" })).newPage();
const problems = [];
page.on("pageerror", (error) => problems.push(`pageerror: ${error.message}`));

let shotIndex = 0;
const shot = async (name) => {
  if (shotsDir) await page.screenshot({ path: path.join(shotsDir, `${String(++shotIndex).padStart(2, "0")}-${name}.png`) });
};
const step = (message) => console.log(`[e2e-web] ${message}`);

try {
  const email = uniqueEmail("web_alice");
  await page.goto(WEB_URL);
  await page.getByTestId("auth-switch-register").click();
  await page.getByTestId("auth-email").fill(email);
  await page.getByTestId("auth-password").fill(PASSWORD);
  await shot("register");
  await page.getByTestId("auth-submit").click();

  // Onboarding: you -> wishes -> photos -> about -> location
  await page.getByTestId("onboarding-name").fill("Alice");
  await page.getByTestId("onboarding-birth").fill("14/03/1995");
  await page.getByTestId("gender-woman").click();
  await page.getByTestId("onboarding-next").click();
  await page.getByTestId("pref-man").click();
  await page.getByTestId("onboarding-next").click();
  const chooser = page.waitForEvent("filechooser");
  await page.getByLabel("Ajouter une photo").first().click();
  (await chooser).setFiles(photoPath);
  await page.getByTestId("photo-0").waitFor({ timeout: 20_000 });
  await page.getByTestId("onboarding-next").click();
  await page.getByTestId("onboarding-bio").fill("Chasseuse de bons cafés.");
  await page.getByRole("checkbox", { name: "Voyages" }).click();
  await shot("onboarding-about");
  await page.getByTestId("onboarding-next").click();
  await page.getByTestId("onboarding-skip-location").click();
  await page.getByTestId("tab-discover").waitFor({ timeout: 20_000 });
  step("onboarding complete");

  // Bob exists, is complete, and likes Alice.
  const login = await request("/auth/login", { method: "POST", json: { email, password: PASSWORD } });
  const aliceId = login.payload.user.id;
  const bob = await createMember({ prefix: "web_bob", name: "Bob", gender: "man", interestedIn: ["woman"] });
  await request("/likes", { method: "POST", token: bob.token, json: { userId: aliceId, action: "like" } });

  await page.reload();
  await page.getByTestId("swipe-like").waitFor({ timeout: 20_000 });
  await page.getByText("Bob, ").first().waitFor();
  await shot("discover");
  await page.getByTestId("swipe-card").getByTestId("profile-card-details").click();
  await page.getByTestId("profile-detail").waitFor();
  await page.waitForTimeout(500);
  await shot("profile-detail");
  await page.getByLabel("Fermer").first().click();
  await page.getByTestId("swipe-like").click();
  await page.getByTestId("match-modal").waitFor({ timeout: 10_000 });
  step("match announced");
  await page.waitForTimeout(400);
  await shot("match");
  await page.getByTestId("match-send-message").click();

  await page.getByTestId("chat-input").fill("Salut Bob ! Ravie de ce match.");
  await page.getByTestId("chat-send").click();
  await page.getByText("Salut Bob ! Ravie de ce match.").waitFor();

  const conversations = await request("/conversations", { token: bob.token });
  const conversationId = conversations.payload.conversations[0].id;
  await request(`/conversations/${conversationId}/read`, { method: "POST", token: bob.token });
  await request(`/conversations/${conversationId}/messages`, {
    method: "POST",
    token: bob.token,
    json: { body: "Salut Alice, moi aussi !" }
  });
  await page.getByText("Salut Alice, moi aussi !").waitFor({ timeout: 10_000 });
  step("realtime message received");
  await shot("chat");

  await page.getByTestId("chat-back").click();
  await page.getByTestId("tab-messages").click();
  await page.getByTestId("conversation-Bob").waitFor();
  await page.getByTestId("tab-notifications").click();
  await page.getByText("Nouveau match").waitFor();
  await shot("notifications");
  await page.getByTestId("tab-profile").click();
  await page.getByRole("heading", { name: "Mon profil" }).waitFor();
  await page.getByTestId("menu-settings").click();
  await page.getByTestId("settings-delete").waitFor();
  await shot("settings");

  // Account deletion removes the member for good.
  await page.getByTestId("settings-delete").click();
  await page.getByTestId("delete-password").fill(PASSWORD);
  await page.getByRole("button", { name: "Supprimer définitivement" }).click();
  await page.getByTestId("auth-submit").waitFor({ timeout: 15_000 });
  const relogin = await request("/auth/login", { method: "POST", json: { email, password: PASSWORD } });
  if (relogin.status !== 401) throw new Error(`deleted account can still sign in (${relogin.status})`);
  step("account deleted");

  if (problems.length > 0) throw new Error(problems.join("\n"));
  console.log("[e2e-web] PASS");
} catch (error) {
  await shot("failure").catch(() => undefined);
  console.error(`[e2e-web] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 1;
} finally {
  await browser.close();
  fs.rmSync(photoPath, { force: true });
}
