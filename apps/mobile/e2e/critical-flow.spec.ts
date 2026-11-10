import { expect, test, type Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

const API = process.env.E2E_API_URL ?? "http://localhost:18080";
const PASSWORD = "Password123";
const PHOTO = path.join(__dirname, "fixtures", "portrait.png");
// Each run uses its own random area so previous runs never interfere.
const HOME = { latitude: -40 + Math.random() * 80, longitude: -150 + Math.random() * 300 };

type ApiUser = { id: string; token: string };

async function api<T>(pathname: string, init: { method?: string; token?: string; body?: unknown; form?: FormData } = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (init.token) headers.Authorization = `Bearer ${init.token}`;
  if (init.body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${pathname}`, {
    method: init.method ?? (init.body || init.form ? "POST" : "GET"),
    headers,
    body: init.form ?? (init.body !== undefined ? JSON.stringify(init.body) : undefined)
  });
  const text = await res.text();
  if (!res.ok) throw new Error(`${init.method ?? "GET"} ${pathname} -> ${res.status} ${text}`);
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Creates a complete, discoverable profile through the public API. */
async function createApiUser(firstName: string, gender: string, interestedIn: string[]): Promise<ApiUser> {
  const email = `e2e-${firstName.toLowerCase()}-${Date.now()}-${Math.floor(Math.random() * 1e6)}@e2e.test`;
  const auth = await api<{ accessToken: string; user: { id: string } }>("/auth/register", { body: { email, password: PASSWORD } });
  const token = auth.accessToken;
  await api("/profile", {
    method: "PUT",
    token,
    body: { firstName, birthdate: "1994-05-12", gender, bio: `Je suis ${firstName}, ravi·e de vous lire.`, interestIds: [1, 21] }
  });
  await api("/profile/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 60, maxDistanceKm: 25 } });
  await api("/profile/location", { method: "PUT", token, body: { latitude: HOME.latitude + 0.01, longitude: HOME.longitude + 0.01 } });
  const form = new FormData();
  form.append("photo", new Blob([fs.readFileSync(PHOTO)], { type: "image/png" }), "photo.png");
  await api("/profile/photos", { token, form });
  return { id: auth.user.id, token };
}

async function next(page: Page) {
  await page.getByTestId("onboarding-next").click();
}

test("register → onboarding → discover → like → match → realtime chat", async ({ browser }) => {
  const bob = await createApiUser("Bob", "man", ["woman"]);

  const context = await browser.newContext({ geolocation: HOME, permissions: ["geolocation"] });
  const page = await context.newPage();
  const pageErrors: string[] = [];
  page.on("pageerror", (e) => pageErrors.push(e.message));
  await page.goto("/");

  // --- Registration ---------------------------------------------------------
  const email = `e2e-alice-${Date.now()}@e2e.test`;
  await page.getByTestId("auth-email").fill(email);
  await page.getByTestId("auth-password").fill("short");
  await page.getByTestId("auth-submit").click();
  await expect(page.getByText("8 caractères minimum.")).toBeVisible();
  await page.getByTestId("auth-password").fill(PASSWORD);
  await page.getByTestId("auth-submit").click();
  await expect(page.getByText(/J’ai 18 ans ou plus/).first()).toBeVisible();
  await page.getByTestId("auth-consent").click();
  await page.getByTestId("auth-submit").click();

  // --- Onboarding -------------------------------------------------------------
  await expect(page.getByText("Comment vous appelez-vous ?")).toBeVisible();
  await page.getByTestId("onboarding-firstname").fill("Alice");
  await next(page);

  await page.getByTestId("onboarding-day").fill("15");
  await page.getByTestId("onboarding-month").fill("06");
  await page.getByTestId("onboarding-year").fill(String(new Date().getFullYear() - 16));
  await next(page);
  await expect(page.getByText("Lueur est réservée aux personnes de 18 ans et plus.")).toBeVisible();
  await page.getByTestId("onboarding-year").fill("1996");
  await next(page);

  await page.getByTestId("gender-woman").click();
  await next(page);
  await page.getByTestId("interested-man").click();
  await next(page);

  await expect(page.getByText("Ajoutez vos photos")).toBeVisible();
  await next(page);
  await expect(page.getByText("Ajoutez au moins une photo de vous.")).toBeVisible();
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("photo-add").click();
  await (await chooser).setFiles(PHOTO);
  await expect(page.getByTestId("photo-slot-0")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText("Principale", { exact: true })).toBeVisible();
  await next(page);

  await page.getByTestId("onboarding-bio").fill("Randonnées le week-end et cafés en terrasse.");
  await page.getByTestId("interest-hiking").click();
  await page.getByTestId("interest-coffee").click();
  await next(page);

  await expect(page.getByText("Trouver des personnes près de vous")).toBeVisible();
  await next(page);

  // --- Discovery --------------------------------------------------------------
  await expect(page.getByText("Découvrir").first()).toBeVisible();
  const card = page.getByTestId("swipe-card");
  await expect(card).toBeVisible();
  await expect(card.getByText("Bob")).toBeVisible();
  await expect(card.getByText(/À (moins de 2|\d+) km/)).toBeVisible();

  // Bob already liked Alice: her like creates the match.
  await api("/discovery/swipes", { token: bob.token, body: { targetId: await aliceId(email), action: "like" } });
  await page.getByTestId("action-like").click();
  await expect(page.getByTestId("match-celebration")).toBeVisible();
  await expect(page.getByText("C’est réciproque !")).toBeVisible();
  await page.getByTestId("celebration-chat").click();

  // --- Chat -------------------------------------------------------------------------
  await expect(page.getByText(/Vous avez matché avec Bob/)).toBeVisible();
  await page.getByTestId("chat-input").fill("Salut Bob ! Tu randonnes où ?");
  await page.getByTestId("chat-send").click();
  await expect(page.getByText("Salut Bob ! Tu randonnes où ?")).toBeVisible();

  const matches = await api<{ matches: { id: string; lastMessage: { body: string } | null }[] }>("/matches", { token: bob.token });
  expect(matches.matches[0].lastMessage?.body).toBe("Salut Bob ! Tu randonnes où ?");
  const matchId = matches.matches[0].id;

  // Bob reads and answers: Alice receives the reply and the read receipt in realtime.
  await api(`/matches/${matchId}/read`, { method: "POST", token: bob.token });
  await expect(page.getByText(/· Lu/)).toBeVisible();
  await api(`/matches/${matchId}/messages`, { token: bob.token, body: { body: "Dans le Vercors, et toi ?" } });
  await expect(page.getByText("Dans le Vercors, et toi ?")).toBeVisible();

  // --- Conversation list & notifications --------------------------------------------
  await page.getByLabel("Retour").click();
  await page.getByText("Messages").last().click();
  await expect(page.getByTestId("conversation-Bob")).toBeVisible();
  await page.getByText("Activité").last().click();
  await expect(page.getByTestId("notification-match")).toBeVisible();

  expect(pageErrors).toEqual([]);
  await context.close();
});

async function aliceId(email: string): Promise<string> {
  const login = await api<{ user: { id: string } }>("/auth/login", { body: { email, password: PASSWORD } });
  return login.user.id;
}

test("report from a conversation, then delete the account", async ({ browser }) => {
  const email = `e2e-safety-${Date.now()}@e2e.test`;
  const auth = await api<{ accessToken: string; user: { id: string } }>("/auth/register", { body: { email, password: PASSWORD } });
  const token = auth.accessToken;
  await api("/profile", { method: "PUT", token, body: { firstName: "Lina", birthdate: "1995-02-02", gender: "woman", bio: "", interestIds: [] } });
  await api("/profile/preferences", { method: "PUT", token, body: { interestedIn: ["man"], minAge: 18, maxAge: 60, maxDistanceKm: 25 } });
  await api("/profile/location", { method: "PUT", token, body: HOME });
  const form = new FormData();
  form.append("photo", new Blob([fs.readFileSync(PHOTO)], { type: "image/png" }), "photo.png");
  await api("/profile/photos", { token, form });
  const max = await createApiUser("Max", "man", ["woman"]);
  await api("/discovery/swipes", { token: max.token, body: { targetId: auth.user.id, action: "like" } });
  const res = await api<{ matchId: string }>("/discovery/swipes", { token, body: { targetId: max.id, action: "like" } });
  await api(`/matches/${res.matchId}/messages`, { token: max.token, body: { body: "Message déplacé" } });

  const context = await browser.newContext();
  const page = await context.newPage();
  page.on("dialog", (d) => d.accept());
  await page.goto("/");
  await page.getByText("Se connecter").first().click();
  await page.getByTestId("auth-email").fill(email);
  await page.getByTestId("auth-password").fill(PASSWORD);
  await page.getByTestId("auth-submit").click();

  await page.getByText("Messages").last().click();
  await page.getByTestId("conversation-Max").click();
  await expect(page.getByText("Message déplacé").last()).toBeVisible();
  await page.getByLabel("Options").click();
  await page.getByTestId("profile-report").click();
  await page.getByTestId("report-harassment").click();
  await page.getByTestId("report-submit").click();
  await expect(page.getByText(/Max est bloqué·e/)).toBeVisible();

  // The conversation is gone for both people.
  await page.getByText("Messages").last().click();
  await expect(page.getByText("Pas encore de match")).toBeVisible();
  const maxMatches = await api<{ matches: unknown[] }>("/matches", { token: max.token });
  expect(maxMatches.matches).toHaveLength(0);

  // Account deletion.
  await page.getByText("Profil").last().click();
  await page.getByTestId("me-account").click();
  await page.getByTestId("account-delete").click();
  await page.getByTestId("account-current-password").fill(PASSWORD);
  await page.getByTestId("account-delete-confirm").click();
  await expect(page.getByTestId("auth-submit")).toBeVisible();
  // The account no longer exists: even a still-unexpired access token is rejected.
  const me = await fetch(`${API}/me`, { headers: { Authorization: `Bearer ${token}` } });
  expect(me.status).toBe(401);
  await context.close();
});
