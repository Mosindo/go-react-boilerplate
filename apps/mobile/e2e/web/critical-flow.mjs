// UI end-to-end test of the critical journey, driving the Expo web build with Playwright:
// register → onboarding (profile, preferences, photo) → discover → like → match →
// realtime conversation (incoming message + read receipt) → account deletion.
//
// Usage (API running, web build served):
//   npx expo export --platform web --output-dir dist && npx serve dist -l 8081
//   E2E_API_URL=http://localhost:18080 E2E_WEB_URL=http://localhost:8081 node e2e/web/critical-flow.mjs
// Requires the `playwright` package and a Chromium build (set PLAYWRIGHT_CHROMIUM_PATH if needed).
import { mkdirSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { PASSWORD, assert, createMember, expectStatus, makePng, request, uniqueEmail } from "../lib/api.mjs";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const WEB_URL = process.env.E2E_WEB_URL || "http://localhost:8081";
const SHOTS = process.env.E2E_SCREENSHOTS || join(tmpdir(), "lueur-e2e");
mkdirSync(SHOTS, { recursive: true });

async function main() {
  const partner = await createMember("e2e_partner", { firstName: "Zelie", gender: "woman", interestedIn: ["man"] });
  const email = uniqueEmail("e2e_ui");
  const photoPath = join(SHOTS, "upload.png");
  writeFileSync(photoPath, makePng(480, 600, 200));

  const browser = await chromium.launch(process.env.PLAYWRIGHT_CHROMIUM_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH } : {});
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  const pageErrors = [];
  page.on("pageerror", (e) => pageErrors.push(String(e)));
  const shot = (name) => page.screenshot({ path: join(SHOTS, `${name}.png`) });
  page.setDefaultTimeout(15_000);

  try {
    // 1. Register
    await page.goto(WEB_URL);
    await page.getByTestId("welcome-register").click();
    await page.getByTestId("register-email").fill(email);
    await page.getByTestId("register-password").fill(PASSWORD);
    await page.getByTestId("register-adult").click();
    await shot("01-register");
    await page.getByTestId("register-submit").click();

    // 2. Onboarding: basics
    await page.getByTestId("onboarding-firstname").fill("Malo");
    await page.getByTestId("onboarding-birth-day").fill("14");
    await page.getByTestId("onboarding-birth-month").fill("02");
    await page.getByTestId("onboarding-birth-year").fill("1993");
    await page.getByRole("checkbox", { name: "Homme" }).click();
    await shot("02-onboarding-basics");
    await page.getByTestId("onboarding-next").click();

    // Preferences
    await page.getByRole("checkbox", { name: "Des femmes" }).click();
    await shot("03-onboarding-preferences");
    await page.getByTestId("onboarding-next").click();

    // About (pick two interests)
    await page.getByRole("checkbox", { name: "Randonnée" }).click();
    await page.getByRole("checkbox", { name: "Cinéma" }).click();
    await shot("04-onboarding-about");
    await page.getByTestId("onboarding-next").click();

    // Location: skip (city only)
    await page.getByTestId("onboarding-skip-location").click();

    // Photos: upload through the real file picker
    const chooser = page.waitForEvent("filechooser");
    await page.getByTestId("photo-add-0").click();
    await (await chooser).setFiles(photoPath);
    await page.getByTestId("photo-slot-0").waitFor();
    await shot("05-onboarding-photos");
    await page.getByTestId("onboarding-finish").click();

    // 3. Partner already liked us (server boosts people who liked you to the top).
    const login = await expectStatus(request("/auth/login", { method: "POST", body: { email, password: PASSWORD } }), 200, "api login");
    await expectStatus(
      request("/swipes", { method: "POST", token: partner.token, body: { targetUserId: login.user.id, decision: "like" } }),
      200,
      "partner likes"
    );

    await page.getByTestId("discover-card").waitFor();
    await page.reload();
    await page.getByTestId("discover-card").waitFor();
    for (let i = 0; i < 20; i++) {
      if (await page.getByTestId("discover-card").getByText("Zelie", { exact: false }).count()) break;
      await page.getByTestId("discover-pass").click();
      await page.waitForTimeout(400);
    }
    await shot("06-discover");
    await page.getByTestId("discover-like").click();

    // 4. Match → conversation
    await page.getByText("C’est réciproque !").waitFor();
    await shot("07-match");
    await page.getByTestId("match-message").click();
    await page.getByTestId("chat-input").fill("Bonjour Zélie ! Une rando ce week-end ?");
    await page.getByTestId("chat-send").click();
    await page.getByText("Bonjour Zélie ! Une rando ce week-end ?").waitFor();

    // Partner reads and answers through the API: the UI must update in real time (WebSocket).
    await expectStatus(request(`/chats/${login.user.id}/read`, { method: "POST", token: partner.token }), 200, "partner reads");
    await expectStatus(
      request(`/chats/${login.user.id}/messages`, { method: "POST", token: partner.token, body: { content: "Avec plaisir 🙂" } }),
      201,
      "partner replies"
    );
    await page.getByText("C’est réciproque !").waitFor({ state: "hidden" });
    await page.getByText("Avec plaisir 🙂").waitFor({ timeout: 8_000 });
    await page.getByText(/· Vu/).waitFor({ timeout: 8_000 });
    await shot("08-chat");

    // 5. Matches tab lists the conversation
    await page.getByRole("button", { name: "Retour" }).first().click();
    await page.getByText("Matchs", { exact: true }).last().click();
    await page.getByTestId("chat-row-Zelie").waitFor();
    await shot("09-matches");

    // 6. Delete the account from settings
    await page.getByText("Profil", { exact: true }).last().click();
    await page.waitForTimeout(1000);
    await shot("10-me");
    await page.getByRole("button", { name: "Mot de passe et suppression" }).click();
    await page.getByTestId("account-delete").click();
    await page.getByTestId("account-delete-password").fill(PASSWORD);
    await page.getByTestId("account-delete-confirm").click();
    await page.getByTestId("welcome-register").waitFor();
    await expectStatus(request("/auth/login", { method: "POST", body: { email, password: PASSWORD } }), 401, "login after deletion");

    assert(pageErrors.length === 0, `page errors: ${pageErrors.join(" | ")}`);
    console.log(`[web-e2e] OK — screenshots in ${SHOTS}`);
  } catch (error) {
    await shot("failure").catch(() => undefined);
    throw error;
  } finally {
    await browser.close();
    await request("/me", { method: "DELETE", token: partner.token, body: { password: PASSWORD } });
  }
}

main().catch((error) => {
  console.error(`[web-e2e] FAILED: ${error.message}`);
  process.exit(1);
});
