/* eslint-disable no-console */
// End-to-end test of the critical journey on the web build:
// register → onboarding (profile, photo, interests) → discover → like →
// match → chat (send, realtime reply, read receipt) → notifications →
// report/block → account deletion.
//
// A second member ("the other person") is driven through the REST API.
//
// Usage:
//   E2E_API_URL=http://localhost:18080 E2E_WEB_URL=http://localhost:8081 node e2e/critical-flow.mjs
// The web build must be exported with EXPO_PUBLIC_API_URL=$E2E_API_URL and the
// API must allow the web origin (ALLOWED_ORIGINS).
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";
import path from "node:path";

const API = process.env.E2E_API_URL ?? "http://localhost:18080";
const WEB = process.env.E2E_WEB_URL ?? "http://localhost:8081";
const SHOTS = process.env.E2E_SCREENSHOTS ?? path.resolve(".e2e/screenshots");
const PASSWORD = "Password123";
const TIMEOUT = 20_000;

mkdirSync(SHOTS, { recursive: true });

const letters = (n) => Array.from({ length: n }, () => "abcdefghijklmnopqrstuvwxyz"[Math.floor(Math.random() * 26)]).join("");
const runId = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
const me = { email: `e2e_me_${runId}@lueur.test`, name: "Alice" };
const other = { email: `e2e_other_${runId}@lueur.test`, name: `Zed${letters(4)}` };

async function api(method, route, body, token) {
  const headers = { Accept: "application/json" };
  let payload;
  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(`${API}${route}`, { method, headers, body: payload });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error(`${method} ${route} -> ${res.status} ${text}`);
  return data;
}

async function pngOf(browser, color) {
  const page = await browser.newPage({ viewport: { width: 600, height: 800 } });
  await page.setContent(`<body style="margin:0;background:linear-gradient(160deg,${color},#f2a65a);width:600px;height:800px"></body>`);
  const buffer = await page.screenshot({ type: "png" });
  await page.close();
  return buffer;
}

function step(label) {
  console.log(`▶ ${label}`);
}

async function shot(page, name) {
  await page.waitForTimeout(600); // let images and transitions settle
  await page.screenshot({ path: path.join(SHOTS, `${name}.png`) });
}

async function main() {
  const browser = await chromium.launch();
  const failures = [];
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, locale: "fr-FR" });
  const page = await context.newPage();
  page.on("pageerror", (err) => failures.push(`page error: ${err.message}`));
  page.setDefaultTimeout(TIMEOUT);

  try {
    // --- The other person, prepared through the API ----------------------
    step("prepare the other member through the API");
    const otherSession = await api("POST", "/auth/register", { email: other.email, password: PASSWORD });
    const otherToken = otherSession.accessToken;
    await api("PUT", "/me/profile", { firstName: other.name, birthdate: "1993-05-20", gender: "man", bio: "Cuisine et randonnée.", city: "Paris" }, otherToken);
    await api("PUT", "/me/preferences", { interestedIn: ["woman"], minAge: 18, maxAge: 99, maxDistanceKm: 50 }, otherToken);
    const form = new FormData();
    form.append("photo", new Blob([await pngOf(browser, "#3d405b")], { type: "image/png" }), "other.png");
    await api("POST", "/me/photos", form, otherToken);

    // --- Registration -----------------------------------------------------
    step("register through the UI");
    await page.goto(WEB);
    await page.getByTestId("welcome-register").click();
    await page.getByTestId("register-email").fill(me.email);
    await page.getByTestId("register-password").fill(PASSWORD);
    await page.getByTestId("register-adult").click();
    await shot(page, "01-register");
    await page.getByTestId("register-submit").click();

    // --- Onboarding -------------------------------------------------------
    step("onboarding: basics");
    await page.getByTestId("onboarding-basics").waitFor();
    await page.getByTestId("profile-first-name").fill(me.name);
    await page.getByTestId("profile-day").fill("14");
    await page.getByTestId("profile-month").fill("02");
    await page.getByTestId("profile-year").fill("1996");
    await page.getByTestId("gender-woman").click();
    await page.getByTestId("profile-city").fill("Paris");
    await shot(page, "02-onboarding-basics");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: preferences");
    await page.getByTestId("onboarding-preferences").waitFor();
    await shot(page, "03-onboarding-preferences");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: location (skipped)");
    await page.getByTestId("onboarding-location").waitFor();
    await shot(page, "04-onboarding-location");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: photo upload");
    await page.getByTestId("onboarding-photos").waitFor();
    const chooser = page.waitForEvent("filechooser");
    await page.getByTestId("photo-add").click();
    await (await chooser).setFiles({ name: "me.png", mimeType: "image/png", buffer: await pngOf(browser, "#b9482f") });
    await page.getByLabel("Photo principale, options").waitFor();
    await shot(page, "05-onboarding-photos");
    await page.getByTestId("onboarding-next").click();

    step("onboarding: bio and interests");
    await page.getByTestId("onboarding-about").waitFor();
    await page.getByTestId("profile-bio").fill("Amatrice de cafés en terrasse et de longues balades.");
    await page.getByTestId("interest-hiking").click();
    await page.getByTestId("interest-coffee").click();
    await shot(page, "06-onboarding-about");
    await page.getByTestId("onboarding-next").click();

    // --- Discovery: the other person likes me first (API), I like back ----
    step("discover and like");
    await page.getByTestId("discover-screen").waitFor();
    const mine = await api("POST", "/auth/login", { email: me.email, password: PASSWORD });
    await api("POST", "/discovery/swipes", { targetUserId: mine.user.id, action: "like" }, otherToken);

    let found = false;
    for (let i = 0; i < 40 && !found; i++) {
      const card = page.locator(`[data-testid="card-${other.name}"]`).first();
      if (await card.isVisible().catch(() => false)) {
        const likeLabel = await page.getByTestId("swipe-like").getAttribute("aria-label");
        if (likeLabel?.includes(other.name)) {
          found = true;
          break;
        }
      }
      if (await page.getByTestId("discover-empty").isVisible().catch(() => false)) {
        break;
      }
      if (await page.getByTestId("swipe-pass").isVisible().catch(() => false)) {
        await page.getByTestId("swipe-pass").click();
      }
      await page.waitForTimeout(400);
    }
    if (!found) throw new Error(`${other.name} never appeared in discovery`);
    await shot(page, "07-discover");
    await page.getByTestId("swipe-like").click();

    step("match modal");
    await page.getByTestId("match-modal").waitFor();
    await shot(page, "08-match");
    await page.getByTestId("match-message").click();

    // --- Chat ---------------------------------------------------------------
    step("chat: send a message");
    await page.getByTestId("chat-screen").waitFor();
    await page.getByTestId("chat-input").fill("Bonjour ! Ravie de ce match 🙂");
    await page.getByTestId("chat-send").click();
    await page.getByText("Bonjour ! Ravie de ce match 🙂").waitFor();

    const matches = await api("GET", "/matches", undefined, otherToken);
    const matchId = matches.matches[0].id;
    const received = await api("GET", `/matches/${matchId}/messages`, undefined, otherToken);
    if (!received.messages.some((m) => m.content.includes("Ravie de ce match"))) {
      throw new Error("message not delivered to the other member");
    }

    step("chat: realtime reply and read receipt");
    await api("POST", `/matches/${matchId}/read`, undefined, otherToken);
    await page.getByTestId("read-receipt").waitFor();
    await api("POST", `/matches/${matchId}/messages`, { content: "Avec plaisir, un café cette semaine ?" }, otherToken);
    await page.getByText("Avec plaisir, un café cette semaine ?").waitFor();
    await shot(page, "09-chat");

    // --- Inbox & notifications --------------------------------------------
    step("inbox and notifications");
    await page.getByLabel("Retour").first().click();
    await page.getByTestId("tab-Matches").click();
    await page.getByTestId(`conversation-${other.name}`).waitFor();
    await shot(page, "10-matches");
    await page.getByTestId("tab-Notifications").click();
    await page.getByTestId("notification-match").first().waitFor();
    await shot(page, "11-notifications");

    // --- Safety: report + block from the conversation ----------------------
    step("report and block from the conversation");
    await page.getByTestId("tab-Matches").click();
    await page.getByTestId(`conversation-${other.name}`).click();
    await page.getByTestId("chat-menu").click();
    await page.getByTestId("chat-report").click();
    await page.getByTestId("report-harassment").click();
    await shot(page, "12-report");
    await page.getByTestId("report-submit").click();
    await page.getByTestId("matches-screen").waitFor();
    const afterBlock = await api("GET", "/matches", undefined, otherToken);
    if (afterBlock.matches.length !== 0) throw new Error("block must remove the match");

    // --- Account deletion ---------------------------------------------------
    step("delete account");
    await page.getByTestId("tab-Me").click();
    await shot(page, "13-me");
    await page.getByTestId("me-delete").click();
    await page.getByTestId("delete-password").fill(PASSWORD);
    await page.getByTestId("delete-confirm").click();
    await page.getByTestId("welcome-screen").waitFor();
    const relogin = await fetch(`${API}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: me.email, password: PASSWORD })
    });
    if (relogin.status !== 401) throw new Error("deleted account can still log in");

    // Cleanup of the API-driven member.
    await api("DELETE", "/me", { password: PASSWORD }, otherToken);
  } catch (error) {
    failures.push(error instanceof Error ? error.message : String(error));
    await shot(page, "failure").catch(() => undefined);
  } finally {
    await browser.close();
  }

  if (failures.length) {
    console.error("✖ E2E failed:\n - " + failures.join("\n - "));
    process.exit(1);
  }
  console.log(`✔ E2E critical flow passed (screenshots in ${SHOTS})`);
}

main();
