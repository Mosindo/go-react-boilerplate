/**
 * Browser end-to-end test of the critical path on the web build, against a real API:
 *   register → onboarding (profile, photo upload, interests) → discovery → like
 *   → "who liked me" → like back → match → realtime chat both ways → read receipt
 *   → notifications → account deletion.
 *
 * Requirements: the API running with an EMPTY database (two fresh users must only see
 * each other), and the web build served (npm run build:web, then any static server).
 *
 *   WEB_URL=http://localhost:8081 API_URL=http://localhost:18080 npm run e2e:web
 */
const fs = require("fs");
const path = require("path");
const { chromium } = require("playwright");
const { makePng, uniqueEmail, assert } = require("./fixtures");

const WEB_URL = process.env.WEB_URL || "http://localhost:8081";
const API_URL = process.env.API_URL || "http://localhost:18080";
const OUT = path.join(__dirname, "output");
const PASSWORD = "e2e-Passw0rd";
const TIMEOUT = 15_000;

fs.mkdirSync(OUT, { recursive: true });
const photoPath = path.join(OUT, "photo.png");
fs.writeFileSync(photoPath, makePng());

const byId = (page, id) => page.locator(`[data-testid="${id}"]`).filter({ visible: true }).first();
const text = (page, value) => page.getByText(value).filter({ visible: true }).first();

async function shot(page, name) {
  await page.screenshot({ path: path.join(OUT, `${name}.png`) });
}

async function step(label, fn) {
  const started = Date.now();
  await fn();
  console.log(`✓ ${label} (${Date.now() - started} ms)`);
}

async function registerAndOnboard(page, { prefix, name, gender, interestedIn }) {
  const email = uniqueEmail(prefix);
  await page.goto(WEB_URL);
  await byId(page, "welcome-register").click();
  await byId(page, "register-email").fill(email);
  await byId(page, "register-password").fill(PASSWORD);
  await byId(page, "register-adult").click();
  await byId(page, "register-submit").click();

  // Step 1: identity
  await byId(page, "onb-firstname").fill(name);
  await byId(page, "onb-birthdate").fill("14/02/1994");
  await byId(page, "onb-next").click();
  // Step 2: gender + preferences
  await byId(page, `onb-gender-${gender}`).click();
  await byId(page, `onb-interested-${interestedIn}`).click();
  await byId(page, "onb-next").click();
  // Step 3: photo upload through the real picker (file chooser on web)
  const [chooser] = await Promise.all([page.waitForEvent("filechooser", { timeout: TIMEOUT }), byId(page, "photo-add").click()]);
  await chooser.setFiles(photoPath);
  await byId(page, "photo-slot-0").waitFor({ timeout: TIMEOUT });
  await byId(page, "onb-next").click();
  // Step 4: about + interests
  await byId(page, "onb-bio").fill(`Bonjour, je suis ${name} et j'adore les balades.`);
  await byId(page, "interest-hiking").click();
  await byId(page, "interest-cinema").click();
  await byId(page, "onb-next").click();
  // Step 5: location (skipped: the API then applies no distance filter)
  await byId(page, "onb-skip-location").click();
  await byId(page, "tab-discover").waitFor({ timeout: TIMEOUT });
  return email;
}

(async () => {
  const health = await fetch(`${API_URL}/health`).then((r) => r.json());
  assert(health.status === "ok", "API must be healthy");

  const browser = await chromium.launch();
  const viewport = { width: 420, height: 860 };
  const ctxA = await browser.newContext({ viewport });
  const ctxB = await browser.newContext({ viewport });
  const alice = await ctxA.newPage();
  const bruno = await ctxB.newPage();
  const errors = [];
  for (const p of [alice, bruno]) p.on("pageerror", (e) => errors.push(e.message));

  try {
    await step("Alice registers and completes onboarding", () =>
      registerAndOnboard(alice, { prefix: "alice", name: "Alice", gender: "woman", interestedIn: "man" })
    );
    await shot(alice, "01-alice-discover-empty");

    await step("Bruno registers and completes onboarding", () =>
      registerAndOnboard(bruno, { prefix: "bruno", name: "Bruno", gender: "man", interestedIn: "woman" })
    );

    await step("Bruno discovers Alice and likes her", async () => {
      await text(bruno, /^Alice, \d+$/).waitFor({ timeout: TIMEOUT });
      await shot(bruno, "02-bruno-discover");
      await byId(bruno, "swipe-like").click();
      await text(bruno, "Vous avez fait le tour").waitFor({ timeout: TIMEOUT });
    });

    await step("Alice sees Bruno in 'Vous ont liké' (free) and likes back", async () => {
      await byId(alice, "tab-likes").click();
      await alice.reload();
      await byId(alice, "tab-likes").click();
      await text(alice, /Bruno, \d+/).waitFor({ timeout: TIMEOUT });
      await shot(alice, "03-alice-likes");
      await text(alice, /Bruno, \d+/).click();
      await byId(alice, "detail-like").click();
    });

    await step("Both get the match celebration", async () => {
      await byId(alice, "match-modal").waitFor({ timeout: TIMEOUT });
      await alice.waitForTimeout(500); // let the fade-in finish before the screenshot
      await shot(alice, "04-alice-match");
      await byId(bruno, "match-modal").waitFor({ timeout: TIMEOUT }); // realtime event
      await byId(bruno, "match-close").click();
    });

    await step("Alice opens the chat and sends a message", async () => {
      await byId(alice, "match-message").click();
      await byId(alice, "chat-input").fill("Salut Bruno ! Une balade dimanche ?");
      await byId(alice, "chat-send").click();
      await text(alice, "Salut Bruno ! Une balade dimanche ?").waitFor({ timeout: TIMEOUT });
    });

    await step("Bruno receives it in real time and replies", async () => {
      await byId(bruno, "tab-chats").click();
      await text(bruno, /Salut Bruno/).waitFor({ timeout: TIMEOUT });
      await text(bruno, /Salut Bruno/).click();
      await byId(bruno, "chat-input").fill("Avec plaisir 🙂");
      await byId(bruno, "chat-send").click();
      await text(alice, "Avec plaisir 🙂").waitFor({ timeout: TIMEOUT });
      await shot(bruno, "05-bruno-chat");
    });

    await step("Alice's message shows a read receipt", async () => {
      await byId(alice, "chat-input").fill("Super, à dimanche !");
      await byId(alice, "chat-send").click();
      await alice.locator('[data-testid="read-receipt"]', { hasText: "Lu" }).waitFor({ timeout: TIMEOUT });
      await shot(alice, "06-alice-chat-read");
    });

    await step("Bruno sees match/message notifications", async () => {
      await bruno.getByRole("button", { name: "Retour" }).filter({ visible: true }).first().click();
      await byId(bruno, "tab-discover").click();
      await byId(bruno, "open-notifications").click();
      await byId(bruno, "notification-match").waitFor({ timeout: TIMEOUT });
      await shot(bruno, "07-bruno-notifications");
    });

    await step("Alice deletes her account", async () => {
      await alice.getByRole("button", { name: "Retour" }).filter({ visible: true }).first().click();
      await byId(alice, "tab-me").click();
      await byId(alice, "me-settings").click();
      await byId(alice, "settings-delete").click();
      await byId(alice, "delete-password").fill(PASSWORD);
      await byId(alice, "delete-submit").click();
      await byId(alice, "confirm-dialog-ok").click();
      await byId(alice, "welcome-register").waitFor({ timeout: TIMEOUT });
    });

    await step("Bruno's conversation with Alice disappears", async () => {
      await bruno.reload();
      await byId(bruno, "tab-chats").click();
      await text(bruno, "Pas encore de conversation").waitFor({ timeout: TIMEOUT });
    });

    assert(errors.length === 0, `no uncaught page errors (got: ${errors.join(" | ")})`);
    console.log(`\nWeb E2E passed. Screenshots in ${OUT}`);
  } catch (err) {
    await shot(alice, "failure-alice").catch(() => undefined);
    await shot(bruno, "failure-bruno").catch(() => undefined);
    console.error(err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
})();
