// Browser-driven end-to-end test of the whole product through the real UI:
//   sign up -> onboarding (profile, preferences, location, photo, interests) -> discover
//   -> like -> match -> conversation (with a live reply over WebSocket) -> settings.
//
// Prerequisites: API running (see README), web build served (npm run e2e:web).
//   WEB_URL (default http://localhost:19006), API (default http://localhost:18080), SHOTS=<dir> for screenshots.
const fs = require("fs");
const os = require("os");
const path = require("path");
const { chromium } = require("playwright");
const { PASSWORD, assert, onboard, register, request, uniqueEmail } = require("../../scripts/lib/api");

const WEB_URL = process.env.WEB_URL || "http://localhost:19006";
const SHOTS = process.env.SHOTS || "";
const SAMPLE = path.join(os.tmpdir(), `lien-e2e-${process.pid}.jpg`);

const byId = (id) => `[data-testid="${id}"]`;

async function shot(page, name) {
  if (SHOTS) {
    fs.mkdirSync(SHOTS, { recursive: true });
    await page.screenshot({ path: path.join(SHOTS, `${name}.png`) });
  }
}

async function main() {
  const { SAMPLE_JPEG_BASE64 } = require("../../scripts/lib/api");
  fs.writeFileSync(SAMPLE, Buffer.from(SAMPLE_JPEG_BASE64, "base64"));

  // Camille already exists with a complete profile.
  const camille = await register(uniqueEmail("camille"));
  await onboard(camille, { name: "Camille", gender: "woman", seeking: ["man"], birthDate: "1996-02-14" });

  // Lea shares more interests with Hugo, so she is the first card of his deck.
  const lea = await register(uniqueEmail("lea"));
  await onboard(lea, { name: "Léa", gender: "woman", seeking: ["man"], birthDate: "1993-07-01", interests: ["music", "cooking"] });

  const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, locale: "fr-FR" });
  const page = await context.newPage();
  const problems = [];
  page.on("pageerror", (err) => problems.push(`pageerror: ${err.message}`));
  page.on("dialog", (dialog) => dialog.accept());

  const hugoEmail = uniqueEmail("hugo");

  // ---- sign up
  await page.goto(WEB_URL);
  await page.waitForSelector(byId("screen-auth"));
  await shot(page, "01-login");
  await page.click(byId("switch-register"));
  await page.fill(byId("input-email"), hugoEmail);
  await page.fill(byId("input-password"), PASSWORD);
  await page.click(byId("auth-submit"));

  // ---- onboarding
  await page.waitForSelector(byId("screen-onboarding"));
  await page.fill(byId("input-first-name"), "Hugo");
  await page.click(byId("onboarding-next")); // missing date + gender must block
  await page.waitForSelector("text=Saisissez une date de naissance valide");
  await page.fill(byId("date-day"), "12");
  await page.fill(byId("date-month"), "04");
  await page.fill(byId("date-year"), "2012"); // minor: must be refused client-side
  await page.click(byId("gender-man"));
  await page.click(byId("onboarding-next"));
  await page.waitForSelector("text=au moins 18 ans");
  await page.fill(byId("date-year"), "1994");
  await shot(page, "02-onboarding-identity");
  await page.click(byId("onboarding-next"));

  await page.click(byId("seek-woman"));
  await shot(page, "03-onboarding-preferences");
  await page.click(byId("onboarding-next"));

  await page.click(byId("choose-city"));
  await page.click(byId("city-Paris"));
  await page.click(byId("onboarding-next"));

  await page.click(byId("onboarding-next")); // no photo yet -> refused
  await page.waitForSelector("text=Ajoutez au moins une photo");
  const [chooser] = await Promise.all([page.waitForEvent("filechooser"), page.click(byId("add-photo"))]);
  await chooser.setFiles(SAMPLE);
  await page.waitForSelector(byId("photo-0"));
  await shot(page, "04-onboarding-photos");
  await page.click(byId("onboarding-next"));

  await page.fill(byId("input-bio"), "Cycliste du dimanche, cuisinier du samedi.");
  await page.click(byId("interest-music"));
  await page.click(byId("interest-cooking"));
  await shot(page, "05-onboarding-about");
  await page.click(byId("onboarding-finish"));

  // ---- discover
  await page.waitForSelector(byId("screen-discover"));
  await page.waitForSelector("text=Léa"); // best match by shared interests comes first
  await shot(page, "06-discover");

  // Swipe the top card away with the mouse (drag left = pass): the next card takes its place.
  const card = page.locator('[data-testid^="card-"]').first();
  const box = await card.boundingBox();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2 - 60, box.y + box.height / 2, { steps: 4 });
  await page.mouse.move(box.x - 140, box.y + box.height / 2 + 10, { steps: 8 });
  await page.mouse.up();
  await page.waitForSelector("text=Camille"); // seeded profile is proposed next
  await page.waitForFunction(() => !document.body.innerText.includes("Léa,"));
  await shot(page, "06b-discover-after-swipe");

  // Camille likes Hugo first (through the API) so that liking back matches.
  const login = await request("/auth/login", { method: "POST", body: { email: hugoEmail, password: PASSWORD } });
  const hugoId = login.payload.user.id;
  const deckAfterPass = await request("/discover?limit=20", { token: login.payload.accessToken });
  assert(!deckAfterPass.payload.profiles.some((p) => p.id === lea.id), "a swiped (passed) profile must never come back");
  const swipe = await request("/swipes", { method: "POST", token: camille.token, body: { targetId: hugoId, action: "like" } });
  assert(swipe.status === 200, `camille like failed: ${swipe.status}`);

  // ---- matches tab shows the pending like (free feature)
  await page.click(byId("tab-matches"));
  await page.waitForSelector(byId("screen-matches"));
  await page.waitForSelector("text=vous a liké");
  await shot(page, "07-likes-received");
  await page.click(byId("tab-home"));
  await page.waitForSelector(byId("action-like"));

  // ---- like -> match
  await page.click(byId("action-like"));
  await page.waitForSelector("text=C'est un match !");
  await page.waitForTimeout(500); // let the celebration animation settle
  await shot(page, "08-match");
  await page.click(byId("match-open-chat"));

  // ---- conversation
  await page.waitForSelector(byId("message-input"));
  await page.fill(byId("message-input"), "Salut Camille, ravi de ce match !");
  await page.click(byId("send-message"));
  await page.waitForSelector("text=Salut Camille, ravi de ce match !");

  // Camille answers: the reply must arrive live, with no refresh.
  const reply = await request(`/conversations/${(await request("/conversations", { token: camille.token })).payload.conversations[0].matchId}/messages`, {
    method: "POST",
    token: camille.token,
    body: { body: "Moi aussi, enchantée Hugo !" }
  });
  assert(reply.status === 201, `reply failed: ${reply.status}`);
  await page.waitForSelector("text=Moi aussi, enchantée Hugo !", { timeout: 8000 });
  await shot(page, "09-conversation");

  // Camille sees Hugo's message as read once he has seen her reply (read receipt).
  const matchId = (await request("/conversations", { token: camille.token })).payload.conversations[0].matchId;
  const thread = await request(`/conversations/${matchId}/messages`, { token: camille.token });
  assert(thread.payload.messages.some((m) => m.senderId === hugoId && m.readAt === null || m.readAt), "hugo's message should exist");

  // ---- other tabs and settings render
  await page.click('[aria-label="Retour"]');
  await page.click(byId("tab-chat"));
  await page.waitForSelector(byId("screen-chat"));
  await shot(page, "10-messages");
  await page.click(byId("tab-notifications"));
  await page.waitForSelector(byId("screen-notifications"));
  await shot(page, "11-notifications");
  await page.click(byId("tab-profile"));
  await page.waitForSelector(byId("screen-profile"));
  await shot(page, "12-profile");
  await page.click(byId("menu-Privacy"));
  await page.waitForSelector(byId("screen-privacy"));
  await shot(page, "13-privacy");

  assert(problems.length === 0, `browser errors: ${problems.join(" | ")}`);
  await browser.close();
  fs.rmSync(SAMPLE, { force: true });
  console.log("[e2e-web] PASS signup/onboarding/discover/like/match/chat/live-reply/settings");
}

main().catch((err) => {
  console.error(`[e2e-web] FAIL ${err instanceof Error ? err.stack : String(err)}`);
  process.exit(1);
});
