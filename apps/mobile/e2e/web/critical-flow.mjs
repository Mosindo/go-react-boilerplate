// End-to-end test of the critical journey, driven through the real UI (Expo web):
// sign up → onboarding (profile, preferences, interests, photo, location)
// → discovery → like → match → realtime chat → read receipt → activity
// → report & block → account deletion.
//
// Prerequisites: the API and `npx expo start --web` are running.
//   E2E_WEB_URL (default http://localhost:8081)
//   E2E_API_URL (default http://localhost:18080)
import { createRequire } from "node:module";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const WEB_URL = process.env.E2E_WEB_URL ?? "http://localhost:8081";
const API_URL = process.env.E2E_API_URL ?? "http://localhost:18080";
const HERE = path.dirname(fileURLToPath(import.meta.url));
const SHOTS = path.join(HERE, "screenshots");
const PASSWORD = "Parcours2026";
const run = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
// A quiet spot (Clermont-Ferrand area) with a random offset, away from the demo seed.
const spot = { latitude: 45.7 + Math.random() * 0.2, longitude: 3.0 + Math.random() * 0.2 };

mkdirSync(SHOTS, { recursive: true });

async function shot(page, name) {
  await page.waitForTimeout(700); // let transitions settle
  await page.screenshot({ path: path.join(SHOTS, `${name}.png`) });
}

function step(name) {
  console.log(`[e2e] ${name}`);
}

async function api(pathname, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_URL}${pathname}`, { method, headers, body: form ?? (body !== undefined ? JSON.stringify(body) : undefined) });
  const text = await response.text();
  const data = text ? JSON.parse(text) : undefined;
  if (!response.ok) throw new Error(`${method} ${pathname} -> ${response.status} ${text}`);
  return data;
}

async function makePhoto(browser, hue) {
  const page = await browser.newPage({ viewport: { width: 400, height: 500 } });
  await page.setContent(`<body style="margin:0;background:linear-gradient(160deg,hsl(${hue},60%,60%),hsl(${hue + 40},50%,25%));height:500px"></body>`);
  const png = await page.screenshot({ type: "png" });
  await page.close();
  return png;
}

/** Creates a complete member through the API (the "other side" of the match). */
async function createPeer(browser) {
  const email = `e2e_peer_${run}@lueur.test`;
  const auth = await api("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  const token = auth.accessToken;
  await api("/me/profile", { method: "PUT", token, body: { firstName: "Gaspard", birthdate: "1993-04-02", gender: "man", bio: "Profil de test E2E.", relationshipGoal: "long_term", city: "Test" } });
  await api("/me/preferences", { method: "PUT", token, body: { interestedIn: ["woman"], minAge: 18, maxAge: 60, maxDistanceKm: 50 } });
  await api("/me/location", { method: "PUT", token, body: { latitude: spot.latitude + 0.02, longitude: spot.longitude } });
  const form = new FormData();
  form.append("photo", new Blob([await makePhoto(browser, 210)], { type: "image/png" }), "peer.png");
  await api("/me/photos", { method: "POST", token, form });
  return { email, token, id: auth.user.id };
}

const byTestId = (page, id) => page.locator(`[data-testid="${id}"]:visible`).first();

async function main() {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 420, height: 860 },
    geolocation: spot,
    permissions: ["geolocation"],
    locale: "fr-FR"
  });
  const page = await context.newPage();
  const consoleErrors = [];
  page.on("pageerror", (e) => consoleErrors.push(e.message));
  page.on("console", (m) => m.type() === "error" && consoleErrors.push(m.text()));

  const peer = await createPeer(browser);
  const email = `e2e_member_${run}@lueur.test`;
  let memberToken = null;

  try {
    step("sign up through the UI");
    await page.goto(WEB_URL, { waitUntil: "networkidle", timeout: 180_000 });
    await byTestId(page, "welcome-signup").click();
    await byTestId(page, "signup-email").fill(email);
    await byTestId(page, "signup-password").fill(PASSWORD);
    await byTestId(page, "signup-submit").click();

    step("onboarding: identity");
    await byTestId(page, "onboarding-identity").waitFor({ timeout: 15_000 });
    await byTestId(page, "onboarding-continue").click();
    await page.getByText("Choisis une option.").waitFor();
    await byTestId(page, "profile-first-name").fill("Lou");
    await byTestId(page, "profile-birth-day").fill("12");
    await byTestId(page, "profile-birth-month").fill("07");
    await byTestId(page, "profile-birth-year").fill(String(new Date().getFullYear() - 16));
    await byTestId(page, "profile-gender-woman").click();
    await byTestId(page, "onboarding-continue").click();
    await page.getByText("Lueur est réservée aux personnes majeures.").waitFor();
    await byTestId(page, "profile-birth-year").fill("1996");
    await shot(page, "01-onboarding-identity");
    await byTestId(page, "onboarding-continue").click();

    step("onboarding: about");
    await byTestId(page, "onboarding-about").waitFor();
    await byTestId(page, "profile-goal-long_term").click();
    await byTestId(page, "profile-bio").fill("Randos, café serré et concerts de jazz.");
    await byTestId(page, "onboarding-continue").click();

    step("onboarding: preferences");
    await byTestId(page, "onboarding-preferences").waitFor();
    await byTestId(page, "prefs-gender-woman").click();
    await byTestId(page, "prefs-gender-nonbinary").click();
    await byTestId(page, "onboarding-continue").click();

    step("onboarding: interests");
    await byTestId(page, "onboarding-interests").waitFor();
    await byTestId(page, "interest-hiking").click();
    await byTestId(page, "interest-jazz").click();
    await byTestId(page, "onboarding-continue").click();

    step("onboarding: photos");
    await byTestId(page, "onboarding-photos").waitFor();
    await byTestId(page, "onboarding-continue").click();
    await byTestId(page, "onboarding-error").waitFor();
    const chooser = page.waitForEvent("filechooser");
    await byTestId(page, "photo-add").click();
    await (await chooser).setFiles({ name: "me.png", mimeType: "image/png", buffer: await makePhoto(browser, 15) });
    await byTestId(page, "photo-slot-0").waitFor({ timeout: 20_000 });
    await shot(page, "02-onboarding-photos");
    await byTestId(page, "onboarding-continue").click();

    step("onboarding: location");
    await byTestId(page, "onboarding-location").waitFor();
    await byTestId(page, "location-update").click();
    await page.getByText("Position enregistrée").waitFor({ timeout: 15_000 });
    await byTestId(page, "onboarding-continue").click();
    await byTestId(page, "onboarding-done").waitFor();
    await byTestId(page, "onboarding-continue").click();

    step("the peer likes the new member (API)");
    memberToken = (await api("/auth/login", { method: "POST", body: { email, password: PASSWORD } })).accessToken;
    const me = await api("/me", { token: memberToken });
    await api("/swipes", { method: "POST", token: peer.token, body: { targetUserId: me.id, decision: "like" } });

    step("discovery shows the peer, like creates a match");
    await byTestId(page, "discover-screen").waitFor();
    await byTestId(page, "discover-card").waitFor({ timeout: 15_000 });
    for (let i = 0; i < 10; i += 1) {
      const label = await byTestId(page, "discover-card").getAttribute("aria-label");
      if (label?.startsWith("Gaspard")) break;
      await byTestId(page, "discover-pass").click();
      await page.waitForTimeout(600);
    }
    const label = await byTestId(page, "discover-card").getAttribute("aria-label");
    if (!label?.startsWith("Gaspard, ")) throw new Error(`expected Gaspard on top, got ${label}`);
    if (!/km/.test(label)) throw new Error(`expected an approximate distance in "${label}"`);
    await shot(page, "03-discover");
    await byTestId(page, "discover-like").click();
    await byTestId(page, "match-celebration").waitFor({ timeout: 10_000 });
    await shot(page, "04-match");

    step("chat in realtime");
    await byTestId(page, "match-open-chat").click();
    await byTestId(page, "chat-screen").waitFor();
    await byTestId(page, "chat-input").fill("Salut Gaspard ! Tu randonnes où ?");
    await byTestId(page, "chat-send").click();
    await page.getByText("Salut Gaspard ! Tu randonnes où ?").waitFor();

    const conversations = await api("/conversations", { token: peer.token });
    const conversation = conversations.conversations.find((c) => c.otherUser.userId === me.id);
    if (!conversation || conversation.unreadCount !== 1) throw new Error(`peer should have 1 unread message: ${JSON.stringify(conversations)}`);
    await api(`/conversations/${conversation.id}/read`, { method: "POST", token: peer.token });
    await page.getByText("· Lu").waitFor({ timeout: 10_000 });
    await api(`/conversations/${conversation.id}/messages`, { method: "POST", token: peer.token, body: { body: "Dans les volcans, évidemment 🌋" } });
    await page.getByText("Dans les volcans, évidemment 🌋").waitFor({ timeout: 10_000 });
    await shot(page, "05-chat");

    step("matches and activity tabs");
    await byTestId(page, "chat-back").click();
    await byTestId(page, "tab-Matches").click();
    await byTestId(page, `conversation-${peer.id}`).waitFor();
    await shot(page, "06-matches");
    await byTestId(page, "tab-Activity").click();
    await byTestId(page, "notification-match").waitFor();
    await shot(page, "07-activity");

    step("report (and block) from the chat");
    await byTestId(page, "tab-Matches").click();
    await byTestId(page, `conversation-${peer.id}`).click();
    await byTestId(page, "chat-menu").click();
    await page.getByText("Voir le profil").click();
    await byTestId(page, "profile-detail").waitFor();
    await shot(page, "08-profile");
    await byTestId(page, "profile-detail-menu").click();
    await byTestId(page, "profile-report").click();
    await byTestId(page, "report-reason-spam").click();
    await byTestId(page, "report-submit").click();
    await byTestId(page, "matches-screen").waitFor();
    await page.waitForTimeout(800);
    if (await byTestId(page, `conversation-${peer.id}`).isVisible()) throw new Error("conversation should disappear after report");
    const peerView = await api("/conversations", { token: peer.token });
    if (peerView.conversations.length !== 0) throw new Error("blocked peer must lose the conversation");

    step("delete the account through the UI");
    await byTestId(page, "tab-Me").click();
    await byTestId(page, "me-account").click();
    await byTestId(page, "account-delete").click();
    await byTestId(page, "account-delete-password").fill(PASSWORD);
    await byTestId(page, "account-delete-confirm").click();
    await byTestId(page, "welcome-screen").waitFor({ timeout: 15_000 });
    const relogin = await fetch(`${API_URL}/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password: PASSWORD }) });
    if (relogin.status !== 401) throw new Error(`deleted account must not log in (got ${relogin.status})`);
    memberToken = null;

    const unexpected = consoleErrors.filter((e) => !/Failed to load resource/.test(e));
    if (unexpected.length > 0) throw new Error(`browser errors:\n${unexpected.join("\n")}`);
    step("PASS");
  } catch (error) {
    await page.screenshot({ path: path.join(SHOTS, "failure.png") }).catch(() => undefined);
    console.error(error);
    process.exitCode = 1;
  } finally {
    await api("/me", { method: "DELETE", token: peer.token, body: { password: PASSWORD } }).catch(() => undefined);
    if (memberToken) await api("/me", { method: "DELETE", token: memberToken, body: { password: PASSWORD } }).catch(() => undefined);
    await browser.close();
  }
}

main();
