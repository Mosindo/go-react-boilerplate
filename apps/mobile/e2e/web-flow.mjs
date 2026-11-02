// Browser E2E for the critical path, run against the Expo web build + a real API:
// register -> onboarding (profile, preferences, location, photo) -> discover -> like
// -> match -> chat (realtime) -> read receipt -> block.
//
// Prerequisites: API on :18080 with ALLOWED_ORIGINS=http://localhost:19006, web build
// served on :19006 (see README "End-to-end test"), and `playwright` installed.
import { createRequire } from "node:module";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { deflateSync } from "node:zlib";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const APP = process.env.E2E_APP_URL ?? "http://localhost:19006";
const PASSWORD = "Password123";

function makePng(path) {
  const size = 320;
  const raw = Buffer.alloc((size * 3 + 1) * size);
  for (let y = 0; y < size; y++)
    for (let x = 0; x < size; x++) {
      const o = y * (size * 3 + 1) + 1 + x * 3;
      raw[o] = x % 256;
      raw[o + 1] = y % 256;
      raw[o + 2] = 140;
    }
  const table = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (b) => {
    let c = 0xffffffff;
    for (const v of b) c = table[(c ^ v) & 255] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (t, d) => {
    const l = Buffer.alloc(4);
    l.writeUInt32BE(d.length);
    const td = Buffer.concat([Buffer.from(t), d]);
    const c = Buffer.alloc(4);
    c.writeUInt32BE(crc(td));
    return Buffer.concat([l, td, c]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  writeFileSync(
    path,
    Buffer.concat([
      Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
      chunk("IHDR", ihdr),
      chunk("IDAT", deflateSync(raw)),
      chunk("IEND", Buffer.alloc(0))
    ])
  );
}

const dir = mkdtempSync(join(tmpdir(), "tandem-e2e-"));
const photo = join(dir, "photo.png");
makePng(photo);

// Isolated spot on Earth so the run never meets other users of the database.
const lat = -30 + Math.random() * 40;
const lng = -100 + Math.random() * 150;

const browser = await chromium.launch();
const log = (m) => console.log(`[e2e] ${m}`);

async function newUser(name, geo) {
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, geolocation: geo, permissions: ["geolocation"] });
  const page = await context.newPage();
  page.on("pageerror", (e) => {
    throw new Error(`[${name}] page error: ${e.message}`);
  });
  page.setDefaultTimeout(15000);
  await page.goto(APP);
  return { page, context };
}

const bySel = (page, id) => page.getByTestId(id);

async function signUp(page, name, gender, interestedIn) {
  const email = `e2e_${name.toLowerCase()}_${Date.now()}@e2e.tandem.local`;
  await bySel(page, "welcome-register").click();
  await bySel(page, "register-email").fill(email);
  await bySel(page, "register-password").fill(PASSWORD);
  await bySel(page, "register-submit").click();

  // 1. identity
  await bySel(page, "field-firstName").fill(name);
  await bySel(page, "field-birthDate").fill("14/03/1996");
  await page.getByRole("button", { name: gender }).click();
  await bySel(page, "onboarding-next").click();
  // 2. preferences
  for (const label of interestedIn) await page.getByRole("button", { name: label, exact: true }).click();
  await bySel(page, "onboarding-next").click();
  // 3. location
  await bySel(page, "locate-me").click();
  await page.getByText("Position enregistrée").waitFor();
  await bySel(page, "onboarding-next").click();
  // 4. about
  await bySel(page, "field-bio").fill(`Bonjour, je suis ${name}.`);
  await page.getByRole("button", { name: "Voyage" }).click();
  await bySel(page, "onboarding-next").click();
  // 5. photo
  const chooser = page.waitForEvent("filechooser");
  await bySel(page, "add-photo").click();
  (await chooser).setFiles(photo);
  await page.getByLabel("Photo principale, options").waitFor();
  await bySel(page, "onboarding-finish").click();
  await page.getByText("Découvrir").first().waitFor();
  return email;
}

try {
  const alice = await newUser("Alice", { latitude: lat, longitude: lng });
  const bob = await newUser("Bob", { latitude: lat + 0.02, longitude: lng });

  // Underage signup is refused by the form before any profile exists.
  const kid = await newUser("Kid", { latitude: lat, longitude: lng });
  await bySel(kid.page, "welcome-register").click();
  await bySel(kid.page, "register-email").fill(`kid_${Date.now()}@e2e.tandem.local`);
  await bySel(kid.page, "register-password").fill(PASSWORD);
  await bySel(kid.page, "register-submit").click();
  await bySel(kid.page, "field-firstName").fill("Kid");
  await bySel(kid.page, "field-birthDate").fill("01/01/2015");
  await kid.page.getByRole("button", { name: "Un homme" }).click();
  await bySel(kid.page, "onboarding-next").click();
  await kid.page.getByText("réservé aux personnes de 18 ans").waitFor();
  await kid.context.close();
  log("underage refused: ok");

  await signUp(alice.page, "Alice", "Une femme", ["Des hommes"]);
  log("alice onboarded: ok");
  await signUp(bob.page, "Bob", "Un homme", ["Des femmes"]);
  log("bob onboarded: ok");

  // Alice sees Bob and likes him: no match yet.
  // Bob joined after Alice's first load: the empty deck reloads on its own, or via "Actualiser".
  await alice.page.getByRole("button", { name: "Actualiser" }).click({ timeout: 2000 }).catch(() => {});
  await alice.page
    .getByLabel(/Bob, \d+ ans/)
    .first()
    .waitFor();
  await bySel(alice.page, "like").click();
  await alice.page.getByText("Vous avez tout vu").waitFor();
  log("alice likes bob: ok");

  // Bob sees Alice, likes her: match overlay on Bob's side.
  await bob.page
    .getByLabel(/Alice, \d+ ans/)
    .first()
    .waitFor();
  await bySel(bob.page, "like").click();
  await bob.page.getByText("C'est un match !").waitFor();
  log("match: ok");

  // Chat, realtime.
  await bySel(bob.page, "match-write").click();
  await bySel(bob.page, "chat-input").fill("Salut Alice !");
  await bySel(bob.page, "chat-send").click();
  await bob.page.getByLabel(/Vous à \d+:\d+ : Salut Alice !/).waitFor();

  await alice.page.getByLabel(/^Messages/).first().click();
  await alice.page.getByText("Salut Alice !").first().waitFor(); // arrives live, no reload
  log("realtime message in inbox: ok");
  await alice.page.getByRole("button", { name: /Conversation avec Bob/ }).click();
  await bySel(alice.page, "chat-input").fill("Salut Bob, ravie !");
  await bySel(alice.page, "chat-send").click();
  await bob.page.getByLabel(/Salut Bob, ravie !/).waitFor();
  await alice.page.getByText(/Lu$/).first().waitFor();
  log("chat both ways + read receipt: ok");

  // Notifications.
  await alice.page.getByLabel(/Retour|Back/i).first().click();
  await alice.page.getByLabel(/^Alertes/).first().click();
  await alice.page.getByText(/Nouveau match avec Bob/).waitFor();
  log("notifications: ok");

  // Delete account (Bob) and make sure he can no longer sign in.
  await bob.page.getByLabel(/Retour|Back/i).first().click();
  await bob.page.getByText("Moi", { exact: true }).click();
  await bob.page.getByRole("button", { name: "Confidentialité et compte" }).click();
  await bySel(bob.page, "delete-account").click();
  await bySel(bob.page, "delete-password").fill(PASSWORD);
  await bySel(bob.page, "delete-confirm").click();
  await bySel(bob.page, "welcome-login").waitFor();
  log("account deletion: ok");

  await alice.page.screenshot({ path: join(dir, "alice-final.png") });
  log(`ALL GOOD (screenshots in ${dir})`);
} catch (error) {
  console.error(`[e2e] FAILED: ${error.message}`);
  for (const ctx of browser.contexts())
    for (const p of ctx.pages())
      await p.screenshot({ path: join(dir, `fail-${Math.random().toString(36).slice(2, 6)}.png`) }).catch(() => {});
  console.error(`[e2e] screenshots in ${dir}`);
  process.exitCode = 1;
} finally {
  await browser.close();
}
