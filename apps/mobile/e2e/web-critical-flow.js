/**
 * End-to-end critical flow against the web build of the app and a running
 * API (see README "Tests"):
 *
 *   register → onboarding (photo upload, location) → discovery → like →
 *   match → real-time conversation → read receipt → notifications →
 *   block → account deletion
 *
 * The second member ("Bastien") is driven through the public API.
 * Usage: E2E_WEB_URL=http://localhost:8081 E2E_API_URL=http://localhost:18080 npm run e2e:web
 */
const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require("playwright");
const { call, createMember, gradientPng } = require("./api-helpers");

const WEB = process.env.E2E_WEB_URL || "http://localhost:8081";
const ARTIFACTS = path.join(__dirname, ".artifacts");
const stamp = Date.now();
const password = "Lueur2026e2e";

function step(name) {
  console.log(`\n▶ ${name}`);
}

async function main() {
  fs.mkdirSync(ARTIFACTS, { recursive: true });
  const photoPath = path.join(ARTIFACTS, "alice.png");
  fs.writeFileSync(photoPath, gradientPng(600, 750, 90));

  step("Create the other member through the API");
  const bastien = await createMember({
    email: `bastien.${stamp}@e2e.lueur.test`,
    password,
    firstName: "Bastien",
    gender: "man",
    interestedIn: ["woman"],
    lat: 45.76,
    lng: 4.84
  });

  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    geolocation: { latitude: 45.7579, longitude: 4.8351 },
    permissions: ["geolocation"],
    locale: "fr-FR"
  });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  const shot = (name) => page.screenshot({ path: path.join(ARTIFACTS, `${name}.png`) });
  const byTestId = (id) => page.locator(`[data-testid="${id}"]`);

  try {
    step("Register");
    await page.goto(WEB);
    await byTestId("welcome-register").click();
    const aliceEmail = `alice.${stamp}@e2e.lueur.test`;
    await byTestId("register-email").fill(aliceEmail);
    await byTestId("register-password").fill(password);
    await byTestId("register-consent").click();
    await byTestId("register-submit").click();

    step("Onboarding: identity");
    await byTestId("onboarding-firstname").fill("Alice");
    await byTestId("onboarding-next").click();
    await byTestId("birth-day").fill("12");
    await byTestId("birth-month").fill("05");
    await byTestId("birth-year").fill("1995");
    await page.getByText(/Vous avez \d+ ans/).waitFor();
    await byTestId("onboarding-next").click();
    await page.getByText("Femme", { exact: true }).click();
    await byTestId("onboarding-next").click();
    await page.getByText("Des hommes", { exact: true }).click();
    await page.getByText("Une relation sérieuse").click();
    await byTestId("onboarding-next").click();

    step("Onboarding: photo upload");
    await page.getByText("Vos photos").waitFor();
    const chooser = page.waitForEvent("filechooser");
    await byTestId("photo-add").click();
    await (await chooser).setFiles(photoPath);
    await byTestId("photo-slot-0").waitFor({ timeout: 20_000 });
    await page.getByText("Principale").waitFor();
    await shot("01-onboarding-photos");
    await byTestId("onboarding-next").click();

    step("Onboarding: about & interests");
    await byTestId("onboarding-bio").fill("Randonnées le week-end, cafés en terrasse la semaine.");
    await page.getByText("Randonnée", { exact: true }).click();
    await page.getByText("Café", { exact: true }).click();
    await byTestId("onboarding-next").click();

    step("Onboarding: location");
    await byTestId("onboarding-locate").click();
    await page.getByText(/Position approximative enregistrée|Position enregistrée/).waitFor();
    await shot("02-onboarding-location");

    // Bastien likes Alice before she sees him, so her like creates a match.
    const aliceSession = await call("POST", "/auth/login", { body: { email: aliceEmail, password } });
    await call("POST", "/swipes", { token: bastien.token, body: { targetUserId: aliceSession.user.id, action: "like" } });
    await byTestId("onboarding-next").click();

    step("Discovery shows Bastien first, like → match");
    await byTestId("deck-top-card").waitFor();
    await page.getByText("Bastien, 34").first().waitFor();
    await shot("03-discover");
    await byTestId("discover-like").click();
    await byTestId("match-modal").waitFor();
    await shot("04-match");
    await byTestId("match-message").click();

    step("Real-time conversation");
    await page.getByText("Dites bonjour à Bastien").waitFor();
    await byTestId("composer-input").fill("Salut Bastien ! Une rando ce week-end ?");
    await byTestId("composer-send").click();
    await page.getByText("Salut Bastien ! Une rando ce week-end ?").waitFor();
    await page.getByText(/Envoyé/).waitFor();

    const convs = await call("GET", "/conversations", { token: bastien.token });
    const conv = convs.conversations.find((c) => c.user.firstName === "Alice");
    if (!conv || conv.unreadCount !== 1 || conv.lastMessage.body !== "Salut Bastien ! Une rando ce week-end ?") {
      throw new Error(`Bastien should see Alice's unread message, got ${JSON.stringify(conv)}`);
    }
    await call("POST", `/conversations/${conv.matchId}/read`, { token: bastien.token });
    await page.getByText(/· Lu/).waitFor({ timeout: 10_000 });
    await call("POST", `/conversations/${conv.matchId}/messages`, { token: bastien.token, body: { body: "Avec plaisir ! Samedi matin ?" } });
    await page.getByText("Avec plaisir ! Samedi matin ?").waitFor({ timeout: 10_000 });
    await shot("05-conversation");

    step("Matches list and notifications");
    await page.locator('[aria-label="Retour"]').first().click();
    await page.getByText("Matchs", { exact: true }).last().click();
    await byTestId("conversation-Bastien").waitFor();
    await shot("06-matches");
    await page.getByText("Activité", { exact: true }).last().click();
    await page.getByText("Nouveau match").waitFor();
    await shot("07-notifications");

    step("Block from the conversation menu");
    await page.getByText("Matchs", { exact: true }).last().click();
    await byTestId("conversation-Bastien").click();
    await byTestId("conversation-menu").click();
    await page.getByText("Bloquer Bastien", { exact: true }).click();
    await page.getByText("Bloquer", { exact: true }).click();
    await page.getByText("Bastien est bloqué·e").waitFor();
    const after = await call("GET", "/conversations", { token: bastien.token });
    if (after.conversations.some((c) => c.matchId === conv.matchId)) {
      throw new Error("Blocking must remove the conversation for both members");
    }

    step("Delete account");
    await page.getByText("Profil", { exact: true }).last().click();
    await page.getByText("Supprimer mon compte").click();
    await byTestId("delete-password").fill(password);
    await byTestId("delete-confirm").click();
    await byTestId("welcome-register").waitFor();
    await shot("08-deleted");
    let loginFailed = false;
    try {
      await call("POST", "/auth/login", { body: { email: aliceEmail, password } });
    } catch {
      loginFailed = true;
    }
    if (!loginFailed) throw new Error("Deleted account must not be able to log in");

    if (errors.length) throw new Error(`Page errors:\n${errors.join("\n")}`);
    console.log("\n✅ Critical flow passed");
  } catch (error) {
    await shot("failure").catch(() => undefined);
    console.error(`\n❌ ${error.message}`);
    if (errors.length) console.error(errors.join("\n"));
    process.exitCode = 1;
  } finally {
    await call("DELETE", "/me", { token: bastien.token, body: { password } }).catch(() => undefined);
    await browser.close();
  }
}

main();
