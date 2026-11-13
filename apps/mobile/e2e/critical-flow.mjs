/**
 * Browser end-to-end test of the critical journey on the Expo web build:
 * register -> onboarding (profile, photo, location, interests) -> discovery
 * -> like -> match -> chat (send + realtime receive) -> notifications ->
 * account deletion.
 *
 * Requires a running API (E2E_API_URL, default http://localhost:18080) whose
 * ALLOWED_ORIGINS contains the web origin, and the exported web app served at
 * E2E_WEB_URL (default http://localhost:8081). See README "Tests".
 */
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { chromium } from "playwright";
import { apiClient, gradientPng } from "./helpers.mjs";

const API = process.env.E2E_API_URL ?? "http://localhost:18080";
const WEB = process.env.E2E_WEB_URL ?? "http://localhost:8081";
const ARTIFACTS = process.env.E2E_ARTIFACTS ?? join(process.cwd(), ".e2e");
const HEADLESS = process.env.E2E_HEADED !== "1";

// A random spot far from any seeded demo data so the deck is deterministic.
const LAT = Number((-40 + Math.random() * 20).toFixed(3));
const LON = Number((-150 + Math.random() * 20).toFixed(3));
const RUN = Date.now();
const PASSWORD = "Password123!";

const api = apiClient(API);
mkdirSync(ARTIFACTS, { recursive: true });

function step(label) {
  console.log(`\n▶ ${label}`);
}

async function main() {
  step("Seed the other member through the API");
  const bastien = await api.createMember({
    email: `e2e-bastien-${RUN}@dating.test`,
    password: PASSWORD,
    firstName: "Bastien",
    gender: "man",
    interestedIn: ["woman"],
    birthdate: "1993-05-14",
    latitude: LAT + 0.01,
    longitude: LON
  });

  const browser = await chromium.launch({ headless: HEADLESS });
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
    geolocation: { latitude: LAT, longitude: LON },
    permissions: ["geolocation"],
    locale: "fr-FR"
  });
  const page = await context.newPage();
  page.on("pageerror", (err) => console.error("[page error]", err.message));
  const byId = (id) => page.getByTestId(id);
  const shot = (name) => page.screenshot({ path: join(ARTIFACTS, `${name}.png`) });
  // Every visible <img> must finish loading: proves signed media URLs work
  // cross-origin and are not blocked by CORP/CSP headers.
  const imagesLoaded = (label) =>
    page
      .waitForFunction(
        () => {
          const visible = [...document.images].filter((img) => img.getBoundingClientRect().width > 0);
          return visible.length > 0 && visible.every((img) => img.complete && img.naturalWidth > 0);
        },
        null,
        { timeout: 10_000 }
      )
      .catch(() => {
        throw new Error(`images did not load on ${label}`);
      });

  try {
    step("Register");
    await page.goto(WEB);
    await byId("welcome-screen").waitFor();
    await shot("01-welcome");
    await byId("welcome-register").click();
    const aliceEmail = `e2e-alix-${RUN}@dating.test`;
    await byId("register-email").fill(aliceEmail);
    await byId("register-password").fill(PASSWORD);
    await byId("register-consent").click();
    await byId("register-submit").click();

    step("Onboarding: identity");
    await byId("onboarding-identity").waitFor();
    await byId("onboarding-first-name").fill("Alix");
    await byId("onboarding-birth-day").fill("21");
    await byId("onboarding-birth-month").fill("09");
    await byId("onboarding-birth-year").fill("1996");
    await byId("onboarding-next").click();

    step("Onboarding: orientation");
    await byId("onboarding-orientation").waitFor();
    await byId("gender-woman").click();
    await byId("interested-man").click();
    await byId("onboarding-next").click();

    step("Onboarding: photos");
    await byId("onboarding-photos").waitFor();
    const photoPath = join(tmpdir(), `aimant-e2e-${RUN}.png`);
    writeFileSync(photoPath, gradientPng(800, 1000, [194, 83, 58], [242, 193, 141]));
    const [chooser] = await Promise.all([page.waitForEvent("filechooser"), byId("photo-add").click()]);
    await chooser.setFiles(photoPath);
    await byId("photo-slot-0").waitFor({ timeout: 20_000 });
    await shot("02-photos");
    await byId("onboarding-next").click();

    step("Onboarding: location");
    await byId("onboarding-location").waitFor();
    await byId("location-city-label").fill("Testville");
    await byId("location-use-current").click();
    await byId("location-current").waitFor({ timeout: 15_000 });
    await byId("onboarding-next").click();

    step("Onboarding: about");
    await byId("onboarding-about").waitFor();
    await byId("onboarding-bio").fill("Randonnées le week-end, jazz le jeudi.");
    await byId("interest-hiking").click();
    await byId("interest-jazz").click();
    await shot("03-about");
    await byId("onboarding-finish").click();

    step("Other member likes Alix (API)");
    const login = await api.expectStatus(
      api.call("/auth/login", { method: "POST", body: { email: aliceEmail, password: PASSWORD } }),
      200,
      "alix login"
    );
    const alixId = login.user.id;
    const feed = await api.expectStatus(api.call("/discovery", { token: bastien.token }), 200, "bastien discovery");
    if (!feed.profiles.some((p) => p.userId === alixId)) {
      throw new Error("Bastien should discover Alix after onboarding");
    }
    await api.expectStatus(
      api.call("/swipes", { method: "POST", token: bastien.token, body: { targetUserId: alixId, action: "like" } }),
      200,
      "bastien like"
    );

    step("Discover and like -> match");
    await byId("discover-screen").waitFor();
    await byId("discover-deck").waitFor({ timeout: 15_000 });
    await page.getByTestId("card-name").filter({ hasText: "Bastien" }).first().waitFor();
    await shot("04-discover");
    await byId("discover-like").click();
    await byId("match-modal").waitFor({ timeout: 10_000 });
    await shot("05-match");
    await byId("match-message").click();

    step("Chat: send and receive in realtime");
    await byId("chat-screen").waitFor();
    await byId("chat-input").fill("Salut Bastien ! Tu randonnes où ?");
    await byId("chat-send").click();
    const sent = byId("chat-message").filter({ hasText: "Salut Bastien ! Tu randonnes où ?" });
    await sent.first().waitFor();
    await page.waitForTimeout(1500);
    const copies = await sent.count();
    if (copies !== 1) {
      throw new Error(`Sent message should be rendered once, found ${copies}`);
    }
    const thread = await api.expectStatus(
      api.call(`/chats/${alixId}/messages`, { token: bastien.token }),
      200,
      "bastien read"
    );
    if (!thread.messages.some((m) => m.content.includes("Tu randonnes"))) {
      throw new Error("Bastien should receive Alix's message");
    }
    await api.expectStatus(
      api.call(`/chats/${alixId}/messages`, {
        method: "POST",
        token: bastien.token,
        body: { content: "Dans les Vosges, et toi ?" }
      }),
      201,
      "bastien reply"
    );
    await byId("chat-message").filter({ hasText: "Dans les Vosges, et toi ?" }).first().waitFor({ timeout: 10_000 });
    await shot("06-chat");

    step("Matches list and notifications");
    await byId("header-back").click();
    await byId("tab-Matches").click();
    await byId(`conversation-${bastien.id}`).waitFor();
    await imagesLoaded("matches");
    await shot("07-matches");
    await byId("tab-Activity").click();
    await byId("notification-match").first().waitFor();
    await shot("08-activity");

    step("Delete account");
    await byId("tab-Me").click();
    await byId("me-screen").waitFor();
    await imagesLoaded("me");
    await shot("09-me");
    await byId("me-settings").click();
    await byId("settings-delete").click();
    await byId("delete-password").fill(PASSWORD);
    await byId("delete-confirm").click();
    await byId("welcome-screen").waitFor({ timeout: 10_000 });
    await api.expectStatus(
      api.call("/auth/login", { method: "POST", body: { email: aliceEmail, password: PASSWORD } }),
      401,
      "login after deletion"
    );
    const chats = await api.expectStatus(api.call("/chats", { token: bastien.token }), 200, "bastien chats");
    if (chats.chats.length !== 0) {
      throw new Error("Deleted member must disappear from conversations");
    }

    console.log("\n✅ Critical flow passed");
  } catch (error) {
    await shot("failure").catch(() => undefined);
    throw error;
  } finally {
    await browser.close();
    // Clean up the API-created member.
    await api.call("/me", { method: "DELETE", token: bastien.token, body: { password: PASSWORD } });
  }
}

main().catch((error) => {
  console.error(`\n❌ ${error.stack ?? error}`);
  process.exit(1);
});
