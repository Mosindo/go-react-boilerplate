// UI end-to-end test of the critical flow, driven in Chromium against the web
// build of the app:
//   register → onboarding (profile, preferences, bio, photo, location)
//   → discover → like → match → realtime chat & read receipt → notification
//   → block → account deletion.
//
// Prerequisites: API running, web build served (npm run build:web), and
// Playwright available (`npm i -g playwright && npx playwright install chromium`).
// Usage: E2E_API_URL=http://localhost:18080 E2E_WEB_URL=http://localhost:8081 npm run e2e:web
const fs = require("fs");
const path = require("path");
const { PASSWORD, assert, expectStatus, makePng, onboard, register, request, uniqueEmail } = require("../lib");

const WEB_URL = process.env.E2E_WEB_URL || "http://localhost:8081";
const OUTPUT = path.join(__dirname, ".output");

async function main() {
  let playwright;
  try {
    playwright = require("playwright");
  } catch {
    throw new Error("Playwright is not installed. Run `npm i -g playwright && npx playwright install chromium`.");
  }
  fs.mkdirSync(OUTPUT, { recursive: true });
  // A short settle delay lets transitions finish so screenshots are meaningful.
  const shot = async (page, name) => {
    await page.waitForTimeout(450);
    await page.screenshot({ path: path.join(OUTPUT, `${name}.png`) });
  };

  const area = { latitude: 12.3 + Math.random(), longitude: -30.4 - Math.random() };
  const browser = await playwright.chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    geolocation: area,
    permissions: ["geolocation"],
    locale: "fr-FR"
  });
  const page = await context.newPage();
  page.on("pageerror", (err) => console.error("[page error]", err.message));
  const byId = (id) => page.getByTestId(id);

  try {
    // Partner account prepared through the API, located in the same area.
    const bob = await register("bob");
    await onboard(bob, { firstName: "Bob", gender: "man", interestedIn: ["woman"], ...area });

    // 1. Registration
    await page.goto(WEB_URL);
    await byId("welcome-screen").waitFor();
    await shot(page, "01-welcome");
    await byId("welcome-register").click();
    const email = uniqueEmail("camille");
    await byId("register-email").fill(email);
    await byId("register-password").fill(PASSWORD);
    await byId("register-confirm").fill(PASSWORD);
    await byId("register-adult").click();
    await byId("register-submit").click();

    // 2. Onboarding
    await byId("onboarding-basics").waitFor();
    await byId("profile-first-name").fill("Camille");
    await byId("profile-birthdate").fill("15061994");
    await byId("gender-woman").click();
    await shot(page, "02-onboarding-basics");
    await byId("onboarding-continue").click();

    await byId("onboarding-preferences").waitFor();
    await byId("interested-man").click();
    await byId("onboarding-continue").click();

    await byId("onboarding-about").waitFor();
    await byId("profile-bio").fill("Amatrice de cafés en terrasse et de randonnées.");
    await byId("interest-cooking").click();
    await byId("interest-hiking").click();
    await byId("onboarding-continue").click();

    await byId("onboarding-photos").waitFor();
    const chooser = page.waitForEvent("filechooser");
    await byId("photo-add").click();
    await (await chooser).setFiles({ name: "me.png", mimeType: "image/png", buffer: makePng(600, 750, [120, 140, 200]) });
    await byId("photo-slot-0").waitFor({ timeout: 20_000 });
    await shot(page, "03-onboarding-photos");
    await byId("onboarding-continue").click();

    await byId("onboarding-location").waitFor();
    await byId("onboarding-continue").click();
    await byId("discover-screen").waitFor({ timeout: 20_000 });
    console.log("[e2e-web] onboarding completed");

    // 3. Bob likes Camille (through the API) so that Camille's like matches.
    const bobDeck = await expectStatus(request("/discover", { token: bob.token }), 200, "bob discover");
    const camille = bobDeck.profiles.find((p) => p.firstName === "Camille");
    assert(camille, "Camille is visible to Bob");
    await expectStatus(
      request("/swipes", { method: "POST", token: bob.token, body: { targetUserId: camille.userId, action: "like" } }),
      200,
      "bob likes"
    );

    // 4. Discover → like → match
    await byId("discover-card").getByText("Bob, ").waitFor();
    await shot(page, "04-discover");
    await byId("discover-like").click();
    await byId("match-modal").waitFor();
    await shot(page, "05-match");
    await byId("match-message").click();

    // 5. Chat in real time
    await byId("chat-screen").waitFor();
    await byId("chat-input").fill("Salut Bob ! Tu randonnes où d'habitude ?");
    await byId("chat-send").click();
    await byId("message-mine").first().waitFor();
    await page.getByText("· Envoyé").waitFor();

    const conversations = await expectStatus(request("/conversations", { token: bob.token }), 200, "bob conversations");
    const conversationId = conversations.conversations[0].id;
    const thread = await expectStatus(request(`/conversations/${conversationId}/messages`, { token: bob.token }), 200, "bob messages");
    assert(thread.messages[0].body.startsWith("Salut Bob"), "Bob received Camille's message");
    await expectStatus(request(`/conversations/${conversationId}/read`, { method: "POST", token: bob.token }), 200, "bob reads");
    await page.getByText("· Lu").waitFor({ timeout: 10_000 });
    await expectStatus(
      request(`/conversations/${conversationId}/messages`, {
        method: "POST",
        token: bob.token,
        body: { body: "Dans le Vercors, et toi ?" }
      }),
      201,
      "bob replies"
    );
    await page.getByText("Dans le Vercors, et toi ?").waitFor({ timeout: 10_000 });
    await shot(page, "06-chat");
    console.log("[e2e-web] realtime chat ok");

    // 6. Notifications
    await byId("chat-back").click();
    await byId("tab-activity").click();
    await byId("notification-match").waitFor();
    await shot(page, "07-activity");

    // 7. Block from the chat menu
    await byId("tab-matches").click();
    await byId("conversation-Bob").click();
    await byId("chat-menu").click();
    await byId("safety-block").click();
    await byId("block-dialog-confirm").click();
    await byId("matches-empty").waitFor({ timeout: 10_000 });
    await expectStatus(request(`/conversations/${conversationId}/messages`, { token: bob.token }), 404, "conversation closed for Bob");
    console.log("[e2e-web] block ok");

    // 8. Profile + account deletion
    await byId("tab-me").click();
    await byId("me-screen").waitFor();
    await shot(page, "08-me");
    await byId("me-account").click();
    await byId("account-delete").click();
    await byId("delete-password").fill(PASSWORD);
    await byId("delete-dialog-confirm").click();
    await byId("welcome-screen").waitFor({ timeout: 10_000 });
    await expectStatus(
      request("/auth/login", { method: "POST", body: { email, password: PASSWORD } }),
      401,
      "deleted account cannot log in"
    );
    await expectStatus(request("/me", { method: "DELETE", token: bob.token, body: { password: PASSWORD } }), 204, "cleanup bob");
    console.log("[e2e-web] OK");
  } catch (error) {
    await shot(page, "failure").catch(() => undefined);
    throw error;
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(`[e2e-web] FAILED: ${error.message}`);
  process.exit(1);
});
