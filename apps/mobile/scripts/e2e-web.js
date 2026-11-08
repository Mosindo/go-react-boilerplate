// Browser end-to-end test of the critical journey, run against the Expo web build:
//   register -> onboarding (gender, preferences, photo, location, bio) -> discovery -> like -> match
//   -> conversation (real-time reply) -> account deletion.
//
// Requirements: a running API (E2E_API_URL), the web build served (E2E_WEB_URL, default http://localhost:19006)
// and Playwright available (`npm i -D playwright` or a preinstalled one).
// Build:  EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web && npx serve dist -l 19006
const fs = require("fs");
const os = require("os");
const path = require("path");
const { PASSWORD, assert, request, makePng, uniqueEmail, createCompleteUser } = require("./lib-e2e");

const WEB_URL = process.env.E2E_WEB_URL || "http://localhost:19006";
const SHOTS = process.env.E2E_SCREENSHOTS;
const AREA = { latitude: 47.2184, longitude: -1.5536 }; // Nantes

let currentPage = null;

async function main() {
  const { chromium } = require("playwright");
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 420, height: 860 },
    geolocation: { latitude: AREA.latitude, longitude: AREA.longitude },
    permissions: ["geolocation"],
    deviceScaleFactor: SHOTS ? 2 : 1
  });
  const page = await context.newPage();
  currentPage = page;
  page.setDefaultTimeout(15000);
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));

  const shot = async (name) => {
    if (SHOTS) {
      fs.mkdirSync(SHOTS, { recursive: true });
      await page.screenshot({ path: path.join(SHOTS, `${name}.png`) });
    }
  };
  const step = (label) => console.log(`✓ ${label}`);
  const tid = (id) => page.getByTestId(id);

  // The other person exists already (API-prepared) and is in the same area.
  const suffix = Math.random().toString(36).slice(2, 6);
  const bobName = `Bastien-${suffix}`; // unique, so reruns on a dirty database stay unambiguous
  const bob = await createCompleteUser({
    firstName: bobName,
    gender: "man",
    interestedIn: ["woman"],
    ...AREA
  });

  // ---- Register ----
  const email = uniqueEmail("claire");
  await page.goto(WEB_URL);
  await shot("01-welcome");
  await tid("welcome-register").click();
  await tid("register-firstname").fill("Claire");
  await tid("register-email").fill(email);
  await tid("register-password").fill(PASSWORD);
  await tid("register-day").fill("12");
  await tid("register-month").fill("5");
  await tid("register-year").fill("1994");
  await shot("02-register");
  await tid("register-submit").click();
  step("registered");

  // ---- Onboarding ----
  await tid("gender-woman").click();
  await shot("03-onboarding-gender");
  await tid("onboarding-next").click();
  await tid("pref-man").click();
  await tid("onboarding-next").click();
  const photoFile = path.join(os.tmpdir(), "ember-e2e.png");
  fs.writeFileSync(photoFile, makePng(500, 640, [90, 140, 200]));
  const [chooser] = await Promise.all([page.waitForEvent("filechooser"), tid("photo-add").click()]);
  await chooser.setFiles(photoFile);
  await tid("photo-0").waitFor();
  await shot("04-onboarding-photo");
  await tid("onboarding-next").click();
  await tid("onboarding-locate").click();
  await tid("onboarding-bio").waitFor();
  await tid("onboarding-bio").fill("Cuisine, randonnées et bons films.");
  await page.getByRole("button", { name: "Musique" }).click();
  await page.getByRole("button", { name: "Cuisine" }).click();
  await shot("05-onboarding-about");
  await tid("onboarding-finish").click();
  step("onboarding completed");

  // ---- Discover + like ----
  await tid("deck-like").waitFor();
  await shot("06-discover");

  // Bastien likes Claire first (she is discoverable now), then Claire likes back => match.
  const me = await request("GET", "/profile", { token: await accessTokenFromStorage(page) });
  assert(me.status === 200 && me.json.complete === true, "profile should be complete after onboarding");
  const like = await request("POST", "/swipes", {
    token: bob.token,
    body: { userId: me.json.userId, action: "like" }
  });
  assert(like.status === 200, `bob like: ${like.status}`);

  // Profiles that already liked her come first in the deck.
  await page.reload();
  await page.getByTestId("deck-top-card").getByText(`${bobName}, 3`).waitFor();
  await tid("deck-like").click();
  await page.getByText("C'est un match !").waitFor();
  await shot("07-match");
  step("liked -> match");
  await tid("match-send-message").click();

  // ---- Chat ----
  await tid("chat-input").fill("Salut Bastien, ravie de te rencontrer !");
  await tid("chat-send").click();
  await page.getByText("Salut Bastien, ravie de te rencontrer !").waitFor();
  const conv =
    like.json.match?.conversationId ??
    (await request("GET", "/conversations", { token: bob.token })).json.conversations[0].id;
  const reply = await request("POST", `/conversations/${conv}/messages`, {
    token: bob.token,
    body: { body: "Salut Claire ! Moi aussi 😊" }
  });
  assert(reply.status === 201, `bob reply: ${reply.status}`);
  await page.getByText("Salut Claire ! Moi aussi 😊").waitFor(); // arrives over the WebSocket, no reload
  await shot("08-chat");
  step("chat real-time");

  // ---- Messages list + notifications + profile ----
  await page.getByLabel("Retour").first().click();
  await tid("tab-messages").click();
  await tid(`conversation-${bobName}`).waitFor();
  await shot("09-messages");
  await tid("tab-me").click();
  await tid("row-security").waitFor();
  await shot("10-profile");

  // ---- Account deletion ----
  await tid("row-security").click();
  await tid("delete-account").click();
  await tid("delete-password").fill(PASSWORD);
  await page.getByRole("button", { name: "Supprimer mon compte" }).last().click();
  await tid("welcome-register").waitFor();
  const login = await request("POST", "/auth/login", { body: { email, password: PASSWORD } });
  assert(login.status === 401, "deleted account must not be able to log in");
  step("account deleted");

  assert(errors.length === 0, `uncaught page errors: ${errors.join(" | ")}`);
  await browser.close();
  console.log("\nWeb end-to-end journey passed.");
}

async function accessTokenFromStorage(page) {
  const raw = await page.evaluate(() => localStorage.getItem("ember.auth.tokens"));
  return JSON.parse(raw).accessToken;
}

main().catch(async (err) => {
  console.error(err);
  if (SHOTS && currentPage) {
    fs.mkdirSync(SHOTS, { recursive: true });
    await currentPage.screenshot({ path: path.join(SHOTS, "failure.png") }).catch(() => undefined);
  }
  process.exit(1);
});
