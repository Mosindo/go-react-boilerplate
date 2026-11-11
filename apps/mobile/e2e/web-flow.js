// End-to-end UI test of the critical journey on the web build:
// register → onboarding (profile, preferences, interests, location, photo)
// → discover → like → match → real-time chat → read receipt → report → delete.
//
// Requirements: a running API (E2E_API_URL, default http://localhost:18080)
// and the web build served at E2E_WEB_URL (default http://localhost:8099),
// built with EXPO_PUBLIC_API_URL pointing at the same API. Playwright with
// Chromium must be installed (npx playwright install chromium).
const fs = require("fs");
const path = require("path");
const { call, expect, makePng, onboardMember, registerMember, uniqueEmail } = require("./fixtures");

let chromium;
try {
  ({ chromium } = require("playwright"));
} catch {
  console.error("[web-e2e] Playwright is not installed. Run: npm i -g playwright && npx playwright install chromium");
  process.exit(1);
}

const WEB_URL = process.env.E2E_WEB_URL || "http://localhost:8099";
const OUT = path.join(__dirname, ".results");
fs.mkdirSync(OUT, { recursive: true });

async function main() {
  const lat = -20 + Math.random() * 40;
  const lng = -100 + Math.random() * 60;
  const email = uniqueEmail("web_a");
  const password = "Password123";

  // The other member is driven through the API.
  const bruno = await registerMember("web_b");
  await onboardMember(bruno, { firstName: "Bruno", gender: "man", interestedIn: ["woman"], lat: lat + 0.01, lng });

  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 420, height: 860 },
    geolocation: { latitude: lat, longitude: lng },
    permissions: ["geolocation"],
    locale: "fr-FR"
  });
  const page = await context.newPage();
  page.on("pageerror", (e) => console.error("[browser]", e.message));
  let step = 0;
  const shot = async (name) =>
    page.screenshot({ path: path.join(OUT, `${String(++step).padStart(2, "0")}-${name}.png`) });

  await page.goto(WEB_URL);
  await page.getByTestId("welcome-register").click();
  await page.getByTestId("register-email").fill(email);
  await page.getByTestId("register-password").fill(password);
  await page.getByTestId("register-adult").click();
  await shot("register");
  await page.getByTestId("register-submit").click();

  // Onboarding — step 1: basics
  await page.getByTestId("onboarding-step-0").waitFor();
  await page.getByTestId("profile-first-name").fill("Alice");
  await page.getByTestId("profile-birthdate").pressSequentially("15061995");
  await page.getByTestId("gender-woman").click();
  await page.getByTestId("profile-bio").fill("J'adore les randonnées et les petits cafés.");
  await shot("onboarding-profile");
  await page.getByTestId("profile-submit").click();

  // Step 2: preferences
  await page.getByTestId("onboarding-step-1").waitFor();
  await page.getByTestId("interested-man").click();
  await page.getByTestId("preferences-submit").click();

  // Step 3: interests
  await page.getByTestId("onboarding-step-2").waitFor();
  await page.getByTestId("interest-hiking").click();
  await page.getByTestId("interest-coffee").click();
  await shot("onboarding-interests");
  await page.getByTestId("interests-submit").click();

  // Step 4: location (browser geolocation is granted with fixed coordinates)
  await page.getByTestId("onboarding-step-3").waitFor();
  await page.getByTestId("location-use").click();

  // Step 5: photo upload through the native file chooser
  await page.getByTestId("onboarding-step-4").waitFor();
  const photoFile = path.join(OUT, "alice.png");
  fs.writeFileSync(photoFile, makePng(600, 800, [90, 120, 200]));
  const [chooser] = await Promise.all([page.waitForEvent("filechooser"), page.getByTestId("photo-add").click()]);
  await chooser.setFiles(photoFile);

  // Completing the profile switches to the main app.
  await page.getByTestId("discover-screen").waitFor({ timeout: 30_000 });
  const login = expect(await call("POST", "/auth/login", { body: { email, password } }), 200, "api login alice");
  const alice = { id: login.user.id, token: login.accessToken };

  await page.getByTestId(`card-${bruno.id}`).waitFor({ timeout: 15_000 });
  await shot("discover");

  // Bruno already likes Alice, so Alice's like creates the match.
  expect(
    await call("POST", "/swipes", { token: bruno.token, body: { targetUserId: alice.id, direction: "like" } }),
    200,
    "bruno likes alice"
  );
  await page.getByTestId("discover-like").click();
  await page.getByTestId("match-modal").waitFor();
  await page.waitForTimeout(400); // let the fade-in finish
  await shot("match");
  await page.getByTestId("match-send-message").click();

  await page.getByTestId("chat-screen").waitFor();
  await page.getByTestId("chat-input").fill("Salut Bruno, ravie de ce match !");
  await page.getByTestId("chat-send").click();
  await page.getByText("Salut Bruno, ravie de ce match !").waitFor();

  const convs = expect(await call("GET", "/conversations", { token: bruno.token }), 200, "bruno conversations");
  const conv = convs.conversations.find((c) => c.user.userId === alice.id);
  if (!conv || conv.lastMessage?.body !== "Salut Bruno, ravie de ce match !")
    throw new Error("Bruno did not receive the message");

  // Reply from Bruno arrives in real time over the WebSocket, then a read receipt.
  expect(
    await call("POST", `/conversations/${conv.matchId}/messages`, {
      token: bruno.token,
      body: { body: "Hello Alice ! Une rando ce week-end ?" }
    }),
    201,
    "bruno replies"
  );
  await page.getByText("Hello Alice ! Une rando ce week-end ?").waitFor({ timeout: 10_000 });
  expect(await call("POST", `/conversations/${conv.matchId}/read`, { token: bruno.token }), 204, "bruno reads");
  await page.getByText(/· Vu/).waitFor({ timeout: 10_000 });
  await shot("chat");

  // Notifications: Alice got a match + coalesced message notification.
  await page.locator('a[aria-label$="back"]:visible').first().click();
  await page.getByTestId("tab-notifications").click();
  await page.getByTestId("notifications-screen").waitFor();
  await page.getByTestId("notification-match").first().waitFor();
  await shot("notifications");

  // Report Bruno from his profile: the match disappears.
  await page.getByTestId("tab-matches").click();
  await page.getByTestId(`conversation-${bruno.id}`).click();
  await page.getByRole("button", { name: "Voir le profil de Bruno" }).click();
  await page.getByTestId("profile-detail").waitFor();
  await page.getByTestId("profile-report").click();
  await page.getByTestId("report-reason-spam").click();
  await shot("report");
  await page.getByTestId("report-submit").click();
  await page.getByTestId("matches-screen").waitFor();
  const after = expect(await call("GET", "/conversations", { token: alice.token }), 200, "alice conversations");
  if (after.conversations.length !== 0) throw new Error("reported match should be gone");

  // Delete the account from the settings.
  await page.getByTestId("tab-me").click();
  await page.getByTestId("me-account").click();
  await page.getByTestId("account-delete-password").fill(password);
  await page.getByTestId("account-delete").click();
  await page.getByTestId("confirm-dialog-confirm").click();
  await page.getByTestId("welcome-screen").waitFor({ timeout: 15_000 });
  await shot("deleted");
  expect(await call("POST", "/auth/login", { body: { email, password } }), 401, "login after deletion");

  await call("DELETE", "/me", { token: bruno.token, body: { password: bruno.password } });
  await browser.close();
  console.log(`[web-e2e] PASS — screenshots in ${OUT}`);
}

main().catch((error) => {
  console.error(`[web-e2e] FAIL ${error.stack || error.message}`);
  process.exit(1);
});
