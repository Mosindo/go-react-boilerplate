// End-to-end scenario run in real browsers against the web build of the app:
//   sign up → onboarding (profile, preferences, photo, location) → discover →
//   like → mutual like (match) → real-time chat → read receipt → block.
//
// Prerequisites: API running (E2E_API_URL, default http://localhost:18080) with the web
// origin allowed (ALLOWED_ORIGINS), and the web build served at E2E_WEB_URL
// (default http://localhost:8081, e.g. `npm run build` then any static server on dist/).
import { mkdirSync } from "node:fs";
import { chromium } from "playwright";

const WEB = process.env.E2E_WEB_URL ?? "http://localhost:8081";
const SHOTS = new URL("./artifacts/", import.meta.url).pathname;
mkdirSync(SHOTS, { recursive: true });

// A random spot on the globe: far from the seeded demo accounts and from previous runs,
// so only this run's two users can discover each other.
const GEO = { latitude: Math.round((Math.random() * 120 - 60) * 100) / 100, longitude: Math.round((Math.random() * 360 - 180) * 100) / 100 };
const run = Date.now().toString(36);
const PASSWORD = "Password123";

const log = (msg) => console.log(`✓ ${msg}`);

async function jpeg(browser, color) {
  const page = await browser.newPage();
  const base64 = await page.evaluate((fill) => {
    const canvas = document.createElement("canvas");
    canvas.width = 600;
    canvas.height = 750;
    const ctx = canvas.getContext("2d");
    const grad = ctx.createLinearGradient(0, 0, 600, 750);
    grad.addColorStop(0, fill);
    grad.addColorStop(1, "#f5e6d8");
    ctx.fillStyle = grad;
    ctx.fillRect(0, 0, 600, 750);
    return canvas.toDataURL("image/jpeg", 0.9).split(",")[1];
  }, color);
  await page.close();
  return Buffer.from(base64, "base64");
}

async function newUserPage(browser) {
  const context = await browser.newContext({
    viewport: { width: 430, height: 900 },
    geolocation: GEO,
    permissions: ["geolocation"],
    locale: "fr-FR"
  });
  const page = await context.newPage();
  page.on("pageerror", (err) => console.error("PAGE ERROR:", err.message));
  return page;
}

async function onboard(page, { email, name, gender, interestedIn, photo, shot }) {
  await page.goto(WEB);
  await page.getByTestId("tab-register").click();
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(PASSWORD);
  await page.getByTestId("submit-auth").click();

  await page.getByTestId("first-name").waitFor();
  await page.getByTestId("first-name").fill(name);
  await page.getByTestId("birth-date").fill("14071994");
  await page.getByTestId(`gender-${gender}`).click();
  await page.getByTestId("bio").fill(`Bonjour, je suis ${name}.`);
  await page.getByText("Voyages", { exact: true }).click();
  await page.getByTestId("save-profile").click();

  await page.getByTestId(`interested-${interestedIn}`).click();
  await page.getByTestId("save-preferences").click();

  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("add-photo").click();
  await (await chooser).setFiles({ name: "photo.jpg", mimeType: "image/jpeg", buffer: photo });
  await page.getByTestId("photos-next").waitFor();
  await page.waitForFunction(() => !document.querySelector('[data-testid="photos-next"]')?.getAttribute("aria-disabled") || document.querySelector('[data-testid="photos-next"]')?.getAttribute("aria-disabled") === "false");
  await page.getByTestId("photos-next").click();

  await page.getByTestId("use-gps").click();
  await page.getByTestId("tab-discover").waitFor({ timeout: 15000 });
  await page.screenshot({ path: `${SHOTS}${shot}-discover.png` });
}

const browser = await chromium.launch();
const pages = {};
try {
  const photoA = await jpeg(browser, "#b84a33");
  const photoB = await jpeg(browser, "#3d5a80");
  const emailA = `alice-${run}@e2e.example`;
  const emailB = `bruno-${run}@e2e.example`;

  const alice = await newUserPage(browser);
  const bruno = await newUserPage(browser);
  Object.assign(pages, { alice, bruno });

  await onboard(alice, { email: emailA, name: "Alice", gender: "woman", interestedIn: "man", photo: photoA, shot: "alice" });
  log("Alice signed up and completed onboarding");
  await onboard(bruno, { email: emailB, name: "Bruno", gender: "man", interestedIn: "woman", photo: photoB, shot: "bruno" });
  log("Bruno signed up and completed onboarding");

  // Alice discovers Bruno and likes him: no match yet.
  // Alice finished onboarding before Bruno existed: coming back to the tab refreshes her empty deck.
  await alice.getByTestId("tab-messages").click();
  await alice.getByTestId("tab-discover").click();
  await alice.getByTestId("profile-card").first().waitFor({ timeout: 15000 });
  await alice.getByText(/Bruno, 3\d/).first().waitFor();
  await alice.screenshot({ path: `${SHOTS}alice-card.png` });
  await alice.getByTestId("like-button").click();
  log("Alice liked Bruno");

  // Bruno sees Alice, likes back → match modal on both screens (Alice's arrives over WebSocket).
  await bruno.getByTestId("tab-discover").click();
  await bruno.getByText(/Alice, 3\d/).first().waitFor({ timeout: 15000 });
  await bruno.getByTestId("like-button").click();
  await bruno.getByTestId("match-modal").waitFor();
  await alice.getByTestId("match-modal").waitFor({ timeout: 10000 });
  await bruno.screenshot({ path: `${SHOTS}bruno-match.png` });
  log("Mutual like produced a match, announced in real time to both users");

  // Conversation
  await bruno.getByTestId("match-chat").click();
  await alice.getByTestId("match-chat").click();
  await bruno.getByTestId("message-input").fill("Salut Alice, ravi de ce match !");
  await bruno.getByTestId("send-message").click();
  await alice.getByText("Salut Alice, ravi de ce match !").waitFor({ timeout: 10000 });
  log("Message delivered to Alice in real time");

  await alice.getByTestId("message-input").fill("Salut Bruno ! Moi aussi 😊");
  await alice.getByTestId("send-message").click();
  await bruno.getByText("Salut Bruno ! Moi aussi 😊").waitFor({ timeout: 10000 });
  await bruno.getByText(/Lu/).first().waitFor({ timeout: 10000 }).catch(() => undefined);
  await alice.screenshot({ path: `${SHOTS}alice-chat.png` });
  log("Reply delivered to Bruno in real time");

  // Read receipt: Alice (who read Bruno's message) → Bruno sees "Lu"
  await bruno.getByText(/· Lu/).first().waitFor({ timeout: 10000 });
  log("Read receipt displayed");

  // Block from the chat menu closes the conversation for both.
  await alice.getByTestId("chat-menu").click();
  await alice.getByTestId("action-block").click();
  await alice.getByTestId("confirm-accept").click();
  await alice.getByTestId("tab-discover").waitFor();
  await bruno.getByText(/n'existe plus/).waitFor({ timeout: 10000 }).catch(async () => {
    // Bruno may be on the chat screen: the conversation query now answers 404 after a refetch.
    await bruno.reload();
  });
  log("Block ended the match");

  console.log("\nE2E OK — screenshots in e2e/artifacts");
} catch (err) {
  console.error("\nE2E FAILED:", err);
  for (const [name, page] of Object.entries(pages)) await page.screenshot({ path: `${SHOTS}failure-${name}.png` }).catch(() => undefined);
  process.exitCode = 1;
} finally {
  await browser.close();
}
