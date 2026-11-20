// UI end-to-end test of the critical journey on the web build, driven by Playwright.
//   E2E_WEB_URL (default http://localhost:8081) must serve `expo export --platform web`
//   E2E_API_URL (default http://localhost:18080) must point to an API whose ALLOWED_ORIGINS includes the web URL.
// Optional: E2E_SCREENSHOTS=<dir> to save screenshots of each step.
import { createRequire } from "node:module";
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { PASSWORD, api, assert, createMember, expectStatus, pngBuffer, uniqueEmail } from "./lib.mjs";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const WEB_URL = process.env.E2E_WEB_URL || "http://localhost:8081";
const SHOTS = process.env.E2E_SCREENSHOTS;

async function main() {
  const bob = await createMember({ name: "Bob", gender: "man", interestedIn: ["woman"] });
  const email = uniqueEmail("alice-ui");
  const photoPath = join(tmpdir(), `lueur-e2e-${Date.now()}.png`);
  writeFileSync(photoPath, pngBuffer(600, 800, [240, 130, 95]));
  if (SHOTS) mkdirSync(SHOTS, { recursive: true });

  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 400, height: 860 }, colorScheme: process.env.E2E_COLOR_SCHEME === "dark" ? "dark" : "light" });
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  const shot = async (name) => {
    if (SHOTS) {
      await page.waitForTimeout(800); // let image fade-ins finish
      await page.screenshot({ path: join(SHOTS, `${name}.png`) });
    }
  };
  const step = (label) => console.log(`[e2e-web] ${label}`);

  try {
    await page.goto(WEB_URL);
    await page.getByTestId("welcome-register").waitFor();
    await shot("01-welcome");

    step("register");
    await page.getByTestId("welcome-register").click();
    await page.getByTestId("register-email").fill(email);
    await page.getByTestId("register-password").fill(PASSWORD);
    await page.getByTestId("register-adult").click();
    await page.getByTestId("register-submit").click();

    step("onboarding: basics");
    await page.getByTestId("field-firstname").waitFor();
    await page.getByTestId("field-firstname").fill("Alice");
    await page.getByTestId("field-day").fill("12");
    await page.getByTestId("field-month").fill("04");
    await page.getByTestId("field-year").fill("2010");
    await page.getByTestId("gender-woman").click();
    await page.getByTestId("onboarding-next").click();
    await page.getByText("Lueur est réservée aux personnes majeures").first().waitFor();
    await shot("02-underage-error");
    await page.getByTestId("field-year").fill("1994");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: preferences");
    await page.getByTestId("interested-man").click();
    await shot("03-preferences");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: about");
    await page.getByTestId("field-bio").fill("Amatrice de randonnées et de vieux films.");
    await page.getByTestId("interest-randonnee").click();
    await page.getByTestId("interest-cinema").click();
    await shot("04-about");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: photo upload");
    const chooser = page.waitForEvent("filechooser");
    await page.getByTestId("photo-add-0").click();
    await (await chooser).setFiles(photoPath);
    await page.getByTestId("photo-slot-0").waitFor({ timeout: 20_000 });
    await shot("05-photos");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: location (skipped)");
    await page.getByText("Continuer sans position").waitFor();
    await page.getByTestId("onboarding-next").click();

    step("discovery");
    const alice = await expectStatus(api("/auth/login", { method: "POST", body: { email, password: PASSWORD } }), 200, "alice login");
    await expectStatus(api("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.user.id, action: "like" } }), 200, "bob likes alice");
    await page.getByTestId("discovery-card").waitFor({ timeout: 20_000 });
    await page.getByTestId("discovery-card").getByText("Bob").first().waitFor();
    await shot("06-discovery");

    step("like -> match");
    await page.getByTestId("discovery-like").click();
    await page.getByTestId("match-title").waitFor({ timeout: 10_000 });
    await shot("07-match");
    await page.getByTestId("match-message").click();

    step("chat + realtime");
    await page.getByTestId("chat-input").waitFor();
    await page.getByTestId("chat-input").fill("Bonjour Bob, ravie de ce match !");
    await page.getByTestId("chat-send").click();
    await page.getByText("Bonjour Bob, ravie de ce match !").waitFor();
    await page.getByTestId("message-mine").first().waitFor();
    const conversations = await expectStatus(api("/conversations?filter=active", { token: bob.token }), 200, "bob conversations");
    const conversationId = conversations.conversations[0]?.id;
    assert(conversationId, "Bob sees the conversation");
    await expectStatus(
      api(`/conversations/${conversationId}/messages`, { method: "POST", token: bob.token, body: { body: "Salut Alice ! Une randonnée ce week-end ?" } }),
      201,
      "bob replies"
    );
    await page.getByText("Une randonnée ce week-end").waitFor({ timeout: 10_000 });
    await shot("08-chat");

    step("matches list");
    await page.getByLabel("Retour").first().click();
    await page.getByTestId("tab-matches").click();
    await page.getByTestId("conversation-row").first().waitFor();
    await shot("09-matches");

    step("profile + account deletion");
    await page.getByTestId("tab-me").click();
    await page.getByTestId("me-settings").waitFor();
    await shot("10-me");
    await page.getByTestId("me-settings").click();
    await page.getByTestId("settings-delete").click();
    await page.getByTestId("delete-password").fill(PASSWORD);
    await page.getByTestId("delete-confirm").click();
    await page.getByTestId("welcome-register").waitFor({ timeout: 10_000 });
    await expectStatus(api("/auth/login", { method: "POST", body: { email, password: PASSWORD } }), 401, "deleted account cannot log in");

    assert(errors.length === 0, `page errors: ${errors.join(" | ")}`);
    await expectStatus(api("/me", { method: "DELETE", token: bob.token, body: { password: PASSWORD } }), 204, "cleanup bob");
    step("OK");
  } catch (error) {
    await shot("failure");
    const ids = await page.locator("[data-testid]").evaluateAll((els) => els.map((e) => e.getAttribute("data-testid")));
    console.error(`[e2e-web] test ids on screen: ${ids.join(", ")}`);
    throw error;
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(`[e2e-web] FAILED: ${error.message}`);
  process.exit(1);
});
