/**
 * Browser end-to-end test of the critical journey, run against the web build:
 * register → profile → photo → discover → like → match (live) → chat → block.
 *
 *   1. start the API (docker compose up -d --build, or go run ./cmd/api) with
 *      ALLOWED_ORIGINS=http://localhost:8081
 *   2. EXPO_PUBLIC_API_URL=http://localhost:18080 npx expo export --platform web --clear
 *   3. serve ./dist on :8081 (any static server with SPA fallback: npx serve -s dist -l 8081)
 *   4. npx playwright install chromium && npm run e2e:web
 *
 * It creates throwaway users (…@smoke.example) and takes screenshots into $E2E_OUT.
 */
const { chromium } = require("playwright");
const fs = require("fs");
const os = require("os");
const path = require("path");
const S = process.env.E2E_OUT || fs.mkdtempSync(path.join(os.tmpdir(), "ember-e2e-"));
const BASE = process.env.E2E_WEB_URL || "http://localhost:8081";

// 200x300 PNG generated without deps
function png() {
  const zlib = require("zlib");
  const w = 200,
    h = 300;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) {
    raw[y * (w * 3 + 1)] = 0;
    for (let x = 0; x < w; x++) {
      const o = y * (w * 3 + 1) + 1 + x * 3;
      raw[o] = x;
      raw[o + 1] = y % 256;
      raw[o + 2] = 140;
    }
  }
  const crcT = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (b) => {
    let c = ~0;
    for (const x of b) c = crcT[(c ^ x) & 255] ^ (c >>> 8);
    return ~c >>> 0;
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
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}
fs.writeFileSync(`${S}/photo.png`, png());

let PAGES = {};
const step = (m) => console.log("✓", m);

async function onboard(browser, name, gender, wants, email) {
  const ctx = await browser.newContext({
    viewport: { width: 420, height: 860 },
    deviceScaleFactor: 1
  });
  const page = await ctx.newPage();
  page.on("pageerror", (e) => console.log("PAGEERROR", name, e.message));
  await page.goto(BASE);
  await page.getByText("Créer un compte").click();
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel("Mot de passe").fill("smoke-pass-123");
  await page.getByRole("button", { name: "Continuer" }).click();
  await page.getByText("Faisons connaissance").waitFor();
  await page.screenshot({ path: `${S}/${name}-1-profile.png` });
  await page.getByLabel("Prénom").fill(name);
  await page.getByLabel("Date de naissance").fill("12031994");
  await page.getByText(gender, { exact: true }).click();
  await page.getByLabel("Ville").fill("Lyon");
  await page.getByText("Voyages").click();
  await page.getByText("Partager ma position approximative").click(); // opt out (no GPS in headless)
  await page.getByRole("button", { name: "Continuer" }).click();
  await page.getByText("Ce que vous cherchez").waitFor();
  await page.getByText(wants, { exact: true }).click();
  await page.getByText("Partout", { exact: true }).click();
  await page.getByRole("button", { name: "Continuer" }).click();
  await page.getByText("Vos plus belles photos").waitFor();
  const [chooser] = await Promise.all([
    page.waitForEvent("filechooser"),
    page.getByRole("button", { name: "Ajouter une photo" }).first().click()
  ]);
  await chooser.setFiles(`${S}/photo.png`);
  await page.getByRole("button", { name: /Photo principale/ }).waitFor({ timeout: 10000 });
  await page.screenshot({ path: `${S}/${name}-2-photos.png` });
  await page.getByRole("button", { name: "Terminer" }).click();
  await page.getByText("Découvrir").first().waitFor();
  return { ctx, page };
}

(async () => {
  const browser = await chromium.launch();
  const stamp = Date.now();
  const a = await onboard(browser, "Alice", "Femme", "Homme", `alice${stamp}@smoke.example`);
  step("Alice: inscription → profil → préférences → photo → découverte");
  PAGES.a = a.page;
  const b = await onboard(browser, "Bruno", "Homme", "Femme", `bruno${stamp}@smoke.example`);
  step("Bruno: onboarding complet");
  PAGES.b = b.page;

  // Alice sees Bruno and likes him.
  // The deck may be empty (Bruno joined after) or already refreshed by the live connection.
  const bruno = a.page.getByText("Bruno, 32").first();
  if (!(await bruno.isVisible().catch(() => false))) {
    await a.page
      .getByRole("button", { name: "Actualiser" })
      .click()
      .catch(() => {});
  }
  await bruno.waitFor({ timeout: 10000 });
  await a.page.screenshot({ path: `${S}/Alice-3-discover.png` });
  step("Alice voit le profil de Bruno dans la découverte");
  await a.page.getByRole("button", { name: "J'aime" }).click();
  step("Alice like Bruno (pas encore de match)");

  // Bruno sees Alice and likes back → match modal.
  await b.page.getByText("Alice, 32").first().waitFor({ timeout: 10000 });
  await b.page.getByRole("button", { name: "J'aime" }).click();
  await b.page.getByText("C'est un match !").waitFor({ timeout: 10000 });
  await b.page.screenshot({ path: `${S}/Bruno-4-match.png` });
  step("Bruno like Alice → « C'est un match ! »");
  // Alice gets it live over the websocket.
  await a.page.getByText("C'est un match !").waitFor({ timeout: 10000 });
  step("Alice reçoit le match en temps réel (WebSocket)");

  await b.page.getByRole("button", { name: /Écrire à Alice/ }).click();
  await b.page.getByLabel("Votre message").fill("Salut Alice, ravi de ce match !");
  await b.page.getByRole("button", { name: "Envoyer" }).click();
  await b.page.getByText("Salut Alice, ravi de ce match !").waitFor();
  step("Bruno envoie un message");

  await a.page.getByRole("button", { name: "Continuer à découvrir" }).click();
  await a.page.getByText("Messages", { exact: true }).first().click();
  await a.page.getByText("Salut Alice, ravi de ce match !").waitFor({ timeout: 10000 });
  await a.page.screenshot({ path: `${S}/Alice-5-messages.png` });
  step("Alice voit la conversation avec le message non lu, sans recharger");
  await a.page.getByText("Salut Alice, ravi de ce match !").click();
  await a.page.getByLabel("Votre message").fill("Moi aussi 🙂");
  await a.page.getByRole("button", { name: "Envoyer" }).click();
  await b.page.getByText("Moi aussi 🙂").waitFor({ timeout: 10000 });
  await b.page.screenshot({ path: `${S}/Bruno-6-chat.png` });
  step("Réponse d'Alice reçue en direct par Bruno");
  await a.page.getByRole("button", { name: "Options de la conversation" }).click();
  await a.page.screenshot({ path: `${S}/Alice-7-menu.png` });
  await a.page.getByText("Bloquer", { exact: true }).click();
  await a.page.getByRole("button", { name: "Bloquer" }).last().click();
  await a.page.getByText("Messages", { exact: true }).first().waitFor({ timeout: 10000 });
  step("Alice bloque Bruno");
  await b.page.getByText("Conversation terminée").waitFor({ timeout: 10000 });
  step("Bruno voit la conversation terminée");

  await browser.close();
  console.log("SMOKE OK");
})().catch(async (e) => {
  console.error("SMOKE FAILED:", e.message);
  for (const k of Object.keys(PAGES)) {
    await PAGES[k].screenshot({ path: `${S}/fail-${k}.png` }).catch(() => {});
    console.log(
      k,
      (
        await PAGES[k]
          .locator("body")
          .innerText()
          .catch(() => "")
      )
        .slice(0, 200)
        .replace(/\n+/g, " | ")
    );
  }
  process.exit(1);
});
