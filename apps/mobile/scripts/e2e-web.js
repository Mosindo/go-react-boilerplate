/*
 * Browser E2E of the critical journey on the web build:
 * register → onboarding (name, birthdate, gender, seeking, photo, bio,
 * location) → likes received → like back → match → realtime chat → read
 * receipt → notifications → swipes → preferences → profile edit → report →
 * settings → account deletion.
 *
 * Requires a running API (E2E_API_URL, default http://localhost:18080) with
 * ALLOWED_ORIGINS containing the web origin, and the exported web build served
 * at E2E_WEB_URL (default http://localhost:19006).
 */
const fs = require("fs");
const os = require("os");
const path = require("path");
const { chromium } = require("playwright");
const { createCompleteUser, expectStatus, gradientPng, request, uniqueEmail } = require("./lib");

const WEB_URL = process.env.E2E_WEB_URL || "http://localhost:19006";
const SHOTS = process.env.E2E_SCREENSHOTS || "";
const PASSWORD = "Secret123";

async function shot(page, name) {
  if (!SHOTS) return;
  fs.mkdirSync(SHOTS, { recursive: true });
  await page.screenshot({ path: path.join(SHOTS, `${name}.png`) });
}

function step(label) {
  console.log(`[e2e:web] ${label}`);
}

async function main() {
  const browser = await chromium.launch(process.env.CHROMIUM_PATH ? { executablePath: process.env.CHROMIUM_PATH } : {});
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    colorScheme: process.env.E2E_COLOR_SCHEME === "light" ? "light" : "dark",
    permissions: ["geolocation"],
    geolocation: { latitude: 48.8584, longitude: 2.2945 }
  });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  const tid = (id) => page.getByTestId(id);

  try {
    const email = uniqueEmail("ui");
    step("register");
    await page.goto(WEB_URL);
    await tid("welcome-register").click();
    await tid("register-email").fill(email);
    await tid("register-password").fill(PASSWORD);
    await tid("register-adult").click();
    await shot(page, "01-register");
    await tid("register-submit").click();

    step("onboarding");
    await tid("onboarding-name").fill("Juliette");
    await tid("onboarding-next").click();
    await tid("birth-day").fill("14");
    await tid("birth-month").fill("02");
    await tid("birth-year").fill("1994");
    await shot(page, "02-birthdate");
    await tid("onboarding-next").click();
    await tid("gender-woman").click();
    await tid("onboarding-next").click();
    await tid("onboarding-step-seeking").waitFor();
    await tid("onboarding-next").click();

    const photoPath = path.join(os.tmpdir(), `lueur-e2e-${Date.now()}.png`);
    fs.writeFileSync(photoPath, gradientPng(600, 800, [232, 112, 138]));
    const [chooser] = await Promise.all([page.waitForEvent("filechooser"), tid("photo-add").click()]);
    await chooser.setFiles(photoPath);
    await tid("photo-slot-0").waitFor({ timeout: 20000 });
    await shot(page, "03-photos");
    await tid("onboarding-next").click();
    await tid("onboarding-bio").fill("Lectrice, randonneuse et amatrice de bons cafés.");
    await tid("interest-hiking").click();
    await tid("interest-coffee").click();
    await tid("onboarding-next").click();
    await tid("onboarding-city").fill("Paris");
    await tid("onboarding-locate").click();
    await tid("discovery-screen").waitFor({ timeout: 15000 });
    await page.waitForTimeout(1200);
    await shot(page, "04-discovery");

    step("second member likes the UI user (through the API)");
    const login = expectStatus(await request("POST", "/auth/login", { body: { email, password: PASSWORD } }), 200, "api login");
    const me = login.user;
    const profile = expectStatus(await request("GET", "/me/profile", { token: login.accessToken }), 200, "own profile");
    if (!profile.completeness.isComplete || !profile.profile.hasLocation) throw new Error("onboarding did not complete the profile");
    const romain = await createCompleteUser({ name: "Romain", gender: "man", seeking: ["woman"], lat: 48.86, lng: 2.34 });
    expectStatus(
      await request("POST", "/swipes", { token: romain.token, body: { targetUserId: me.id, action: "like" } }),
      200,
      "romain likes"
    );

    step("likes received → like back → match");
    await tid("tab-likes").click();
    await tid("likes-screen").waitFor();
    await page.getByRole("button", { name: /Romain, \d+ ans, vous a liké/ }).click();
    await tid("profile-detail").waitFor();
    await shot(page, "05-profile-detail");
    await tid("detail-like").click();

    step("chat in realtime");
    await tid("chat-screen").waitFor({ timeout: 10000 });
    await tid("chat-input").fill("Bonjour Romain, ravie de ce match !");
    await tid("chat-send").click();
    await page.getByText("Bonjour Romain, ravie de ce match !").waitFor();
    const conversations = expectStatus(await request("GET", "/conversations", { token: romain.token }), 200, "romain conversations");
    const conversation = conversations.conversations[0];
    if (!conversation || conversation.lastMessage?.body !== "Bonjour Romain, ravie de ce match !")
      throw new Error("message not received by Romain");
    expectStatus(await request("POST", `/conversations/${conversation.id}/read`, { token: romain.token }), 200, "romain reads");
    expectStatus(
      await request("POST", `/conversations/${conversation.id}/messages`, {
        token: romain.token,
        body: { body: "Avec plaisir ! Un café cette semaine ?" }
      }),
      201,
      "romain replies"
    );
    await page.getByText("Avec plaisir ! Un café cette semaine ?").waitFor({ timeout: 8000 });
    await shot(page, "06-chat");

    step("conversations list and notifications");
    await page.getByRole("button", { name: "Retour" }).click();
    await tid("tab-messages").click();
    await tid("conversation-row").first().waitFor();
    await shot(page, "07-conversations");
    await tid("tab-discover").click();
    await tid("open-notifications").click();
    await tid("notifications-screen").waitFor();
    await tid("notification-row").first().waitFor();
    await shot(page, "08-notifications");
    await page.getByRole("button", { name: "Retour" }).click();

    step("swipe in discovery (when other profiles exist), preferences");
    await tid("tab-discover").click();
    await tid("discovery-screen").waitFor();
    await page.waitForTimeout(800);
    if (await tid("swipe-pass").isVisible()) {
      await tid("swipe-pass").click();
      await page.waitForTimeout(600);
      if (await tid("swipe-like").isVisible()) await tid("swipe-like").click();
      await page.waitForTimeout(600);
    }
    await tid("open-preferences").click();
    await tid("preferences-screen").waitFor();
    await page.getByRole("button", { name: "Augmenter Âge maximum" }).click();
    await tid("preferences-save").click();
    await tid("discovery-screen").waitFor();

    step("edit profile");
    await tid("tab-me").click();
    await tid("open-edit-profile").click();
    await tid("edit-bio").fill("Nouvelle bio écrite pendant le test de bout en bout.");
    await tid("edit-profile-save").click();
    await page.getByText("Nouvelle bio écrite pendant le test de bout en bout.").waitFor();

    step("report from the conversation");
    await tid("tab-messages").click();
    await tid("conversation-row").first().click();
    await tid("chat-menu").click();
    await tid("chat-report").click();
    await tid("report-harassment").click();
    await tid("report-submit").click();
    await tid("conversations-screen").waitFor();
    await page.getByText("Aucun match pour l'instant").waitFor({ timeout: 8000 });
    expectStatus(
      await request("GET", `/conversations/${conversation.id}/messages`, { token: romain.token }),
      404,
      "conversation removed after report"
    );

    step("profile, settings, delete account");
    await tid("tab-me").click();
    await tid("me-screen").waitFor();
    await shot(page, "09-me");
    await tid("open-settings").click();
    await tid("settings-screen").waitFor();
    await shot(page, "10-settings");
    await tid("open-delete-account").click();
    await tid("delete-password").fill(PASSWORD);
    await tid("delete-submit").click();
    await tid("welcome-screen").waitFor({ timeout: 10000 });
    expectStatus(await request("POST", "/auth/login", { body: { email, password: PASSWORD } }), 401, "deleted account cannot log in");

    expectStatus(await request("DELETE", "/me", { token: romain.token, body: { password: romain.password } }), 204, "cleanup romain");
    if (errors.length) throw new Error(`page errors: ${errors.join(" | ")}`);
    console.log("[e2e:web] OK — full journey passed");
  } catch (error) {
    await shot(page, "failure");
    throw error;
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error("[e2e:web] FAILED:", error.message);
  process.exit(1);
});
