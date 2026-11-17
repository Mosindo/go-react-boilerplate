// End-to-end test of the critical journey on the web build, with two real users:
// register -> onboarding (photo upload, location) -> discovery -> like -> match ->
// realtime chat -> read receipt -> notifications -> report/block -> account deletion.
//
// Requires a running API and the exported web app:
//   E2E_WEB_URL (default http://localhost:8081), E2E_API_URL (default http://localhost:18080)
// Uses the `playwright` package (CHROMIUM_PATH can point to a preinstalled browser).
import { createRequire } from "node:module";
import assert from "node:assert/strict";
import { mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const WEB_URL = process.env.E2E_WEB_URL ?? "http://localhost:8081";
const API_URL = process.env.E2E_API_URL ?? "http://localhost:18080";
const PASSWORD = "Password123";
const run = Date.now().toString(36);
const T = 20_000;
// Letters-only suffix so first names stay valid and unique per run.
const tag = run
  .replace(/[^a-z]/g, "")
  .slice(0, 4)
  .padEnd(4, "x");
const ALICE = `Alice${tag}`;
const BOB = `Bob${tag}`;

function log(step) {
  console.log(`✓ ${step}`);
}

async function byTestId(page, id) {
  const locator = page.getByTestId(id).first();
  await locator.waitFor({ state: "visible", timeout: T });
  return locator;
}

async function tap(page, id) {
  await (await byTestId(page, id)).click();
}

async function fill(page, id, value) {
  const field = await byTestId(page, id);
  await field.fill(value);
}

/** Presses the visible in-app header back button (browser history is not used by the app). */
async function back(page) {
  await page.locator('[aria-label^="Retour"]:visible').last().click();
}

async function newUserPage(browser) {
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, locale: "fr-FR" });
  const page = await context.newPage();
  page.on("dialog", (dialog) => void dialog.accept());
  page.on("pageerror", (error) => console.error("[pageerror]", error.message));
  await page.goto(WEB_URL);
  return page;
}

async function register(page, email) {
  await tap(page, "welcome-register");
  await fill(page, "register-email", email);
  await fill(page, "register-password", PASSWORD);
  await tap(page, "register-adult");
  await tap(page, "register-submit");
  await byTestId(page, "onboarding-identity");
}

async function onboard(page, { firstName, birthdate, gender, interestedIn, interest }) {
  // Underage birthdates are refused client-side (and server-side).
  await fill(page, "field-first-name", firstName);
  await fill(page, "field-birthdate", "01012015");
  await tap(page, "onboarding-next");
  await page.getByText("réservé aux personnes majeures").first().waitFor({ timeout: T });
  await fill(page, "field-birthdate", birthdate);
  await tap(page, "onboarding-next");

  await byTestId(page, "onboarding-gender");
  await tap(page, `gender-${gender}`);
  for (const g of ["woman", "man", "nonbinary"]) {
    const chip = page.getByTestId(`interested-${g}`);
    const checked = (await chip.getAttribute("aria-checked")) === "true";
    if (checked !== interestedIn.includes(g)) await chip.click();
  }
  await tap(page, "onboarding-next");

  await byTestId(page, "onboarding-photos");
  const photo = await page.screenshot({ type: "png" });
  const chooser = page.waitForEvent("filechooser", { timeout: T });
  await tap(page, "photo-add");
  await (await chooser).setFiles({ name: "portrait.png", mimeType: "image/png", buffer: photo });
  await page.getByRole("img", { name: "Photo 1" }).waitFor({ timeout: 30_000 });
  await tap(page, "onboarding-next");

  await byTestId(page, "onboarding-about");
  await fill(page, "field-bio", `Bonjour, je suis ${firstName}.`);
  await tap(page, `interest-${interest}`);
  await tap(page, "onboarding-next");

  await byTestId(page, "onboarding-location");
  await tap(page, "location-city-toggle");
  await tap(page, "city-Paris");
  await byTestId(page, "discover-screen");
}

/** Passes other profiles until `name` is on top of the deck, then likes it. */
async function findAndLike(page, name) {
  for (let i = 0; i < 40; i++) {
    const like = page.getByTestId("discover-like");
    if (await like.isVisible().catch(() => false)) {
      const label = (await like.getAttribute("aria-label")) ?? "";
      if (label === `Aimer ${name}`) {
        await like.click();
        return;
      }
      await page
        .getByTestId("discover-pass")
        .click({ timeout: 5000 })
        .catch(() => undefined);
      await page.waitForTimeout(400);
      continue;
    }
    const refresh = page.getByRole("button", { name: "Actualiser" });
    if (await refresh.isVisible().catch(() => false)) await refresh.click({ timeout: 2000 }).catch(() => undefined);
    await page.waitForTimeout(800);
  }
  throw new Error(`${name} never appeared in discovery`);
}

async function main() {
  const health = await fetch(`${API_URL}/health`).then((r) => r.json());
  assert.equal(health.status, "ok", "API must be healthy");

  const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
  try {
    const alice = await newUserPage(browser);
    const bob = await newUserPage(browser);

    await register(alice, `alice.${run}@e2e.test`);
    log("Alice registered");
    await onboard(alice, { firstName: ALICE, birthdate: "14021995", gender: "woman", interestedIn: ["man"], interest: "hiking" });
    log("Alice completed onboarding (photo + city)");

    await register(bob, `bob.${run}@e2e.test`);
    await onboard(bob, { firstName: BOB, birthdate: "03091993", gender: "man", interestedIn: ["woman"], interest: "hiking" });
    log("Bob registered and completed onboarding");

    await findAndLike(bob, ALICE);
    assert.equal(await bob.getByTestId("match-modal").isVisible(), false, "a one-sided like must not match");
    log("Bob liked Alice (no match yet)");

    await findAndLike(alice, BOB);
    await byTestId(alice, "match-modal");
    log("Mutual like: Alice sees the match celebration");

    await bob.getByTestId("toast").getByText(`Nouveau match avec ${ALICE}`).waitFor({ timeout: T });
    log("Bob received the match in real time");

    await tap(alice, "match-send-message");
    await byTestId(alice, "conversation-screen");
    await fill(alice, "message-input", "Salut Bob ! Randonnée ce week-end ?");
    await tap(alice, "message-send");
    await alice.getByTestId("message-bubble").getByText("Salut Bob ! Randonnée ce week-end ?").waitFor({ timeout: T });
    await alice.getByTestId("message-status").getByText("Envoyé").waitFor({ timeout: T });
    log("Alice sent the first message");

    await tap(bob, "tab-matches");
    await byTestId(bob, "matches-screen");
    await bob.getByRole("button", { name: new RegExp(`Conversation avec ${ALICE}`) }).click();
    await bob.getByTestId("message-bubble").getByText("Salut Bob ! Randonnée ce week-end ?").waitFor({ timeout: T });
    log("Bob opened the conversation and read the message");

    await alice.getByTestId("message-status").getByText("Lu").waitFor({ timeout: T });
    log("Alice sees the read receipt in real time");

    await fill(bob, "message-input", "Avec plaisir 🙂");
    await tap(bob, "message-send");
    await alice.getByTestId("message-bubble").getByText("Avec plaisir 🙂").waitFor({ timeout: T });
    log("Alice received Bob's reply in real time");

    await back(bob);
    await tap(bob, "tab-activity");
    await byTestId(bob, "notifications-screen");
    await byTestId(bob, "notification-match.new");
    log("Bob sees the match notification");

    // Bob reports Alice from the conversation: the match ends for both.
    await bob
      .getByRole("button", { name: new RegExp(`Conversation avec ${ALICE}`) })
      .click()
      .catch(async () => {
        await tap(bob, "tab-matches");
        await bob.getByRole("button", { name: new RegExp(`Conversation avec ${ALICE}`) }).click();
      });
    await tap(bob, "conversation-menu");
    await tap(bob, "menu-report");
    await tap(bob, "report-reason-fake_profile");
    await tap(bob, "report-submit");
    await bob.getByText("Le signalement est enregistré").waitFor({ timeout: T });
    log("Bob reported (and blocked) Alice");

    await alice.getByText("Conversation indisponible").waitFor({ timeout: T });
    log("Alice's conversation closed in real time");

    // Logout / login round trip for Bob.
    await tap(bob, "tab-me");
    await tap(bob, "me-logout");
    await tap(bob, "welcome-login");
    await fill(bob, "login-email", `bob.${run}@e2e.test`);
    await fill(bob, "login-password", "WrongPassword9");
    await tap(bob, "login-submit");
    await bob.getByTestId("login-error").waitFor({ timeout: T });
    await fill(bob, "login-password", PASSWORD);
    await tap(bob, "login-submit");
    await byTestId(bob, "discover-screen");
    log("Bob logged out and back in (wrong password rejected)");

    // Alice deletes her account.
    await back(alice);
    await tap(alice, "tab-me");
    await tap(alice, "me-delete");
    await fill(alice, "delete-password", PASSWORD);
    await tap(alice, "delete-confirm");
    await byTestId(alice, "welcome-screen");
    const login = await fetch(`${API_URL}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: `alice.${run}@e2e.test`, password: PASSWORD })
    });
    assert.equal(login.status, 401, "deleted account must not be able to log in");
    log("Alice deleted her account");

    console.log("\nE2E critical flow passed.");
  } catch (error) {
    for (const [i, ctx] of browser.contexts().entries()) {
      for (const page of ctx.pages()) {
        const dir = fileURLToPath(new URL("./test-results/", import.meta.url));
        mkdirSync(dir, { recursive: true });
        await page.screenshot({ path: `${dir}failure-${i}.png` }).catch(() => undefined);
        const buttons = await page
          .locator("[role=button]:visible, button:visible, a:visible")
          .evaluateAll((els) => els.map((e) => e.getAttribute("aria-label") || e.textContent?.trim()).filter(Boolean))
          .catch(() => []);
        console.error(`[user ${i}] visible controls:`, buttons.join(" | "));
      }
    }
    throw error;
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
