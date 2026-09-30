import { expect, test, type APIRequestContext } from "@playwright/test";
import fs from "fs";
import path from "path";

const API = process.env.E2E_API_URL ?? "http://localhost:18080";
const PASSWORD = "Password123";
const photo = path.join(__dirname, "fixtures", "portrait.png");
const unique = () => `${Date.now()}_${Math.floor(Math.random() * 1e6)}`;

async function apiSession(request: APIRequestContext, email: string, register: boolean) {
  const res = await request.post(`${API}/auth/${register ? "register" : "login"}`, { data: { email, password: PASSWORD } });
  expect(res.ok(), await res.text()).toBeTruthy();
  const body = await res.json();
  return { token: body.accessToken as string, userId: body.user.id as string };
}

// Inscription -> création du profil -> découverte -> like -> match -> conversation (temps réel)
test("new member signs up, builds a profile, matches and chats in real time", async ({ page, request }) => {
  const runId = unique();
  const aliceEmail = `alice_${runId}@e2e.test`;
  const bobEmail = `bob_${runId}@e2e.test`;

  // Bob is prepared through the API: complete profile, interested in women.
  const bob = await apiSession(request, bobEmail, true);
  const bobAuth = { Authorization: `Bearer ${bob.token}` };
  expect((await request.patch(`${API}/profile`, { headers: bobAuth, data: { firstName: "Bob", birthdate: "1993-03-03", gender: "man", bio: "Cuisine et randonnée." } })).ok()).toBeTruthy();
  expect((await request.put(`${API}/profile/preferences`, { headers: bobAuth, data: { interestedIn: ["woman"], minAge: 18, maxAge: 99, maxDistanceKm: 100 } })).ok()).toBeTruthy();
  const upload = await request.post(`${API}/profile/photos`, {
    headers: bobAuth,
    multipart: { photo: { name: "bob.png", mimeType: "image/png", buffer: fs.readFileSync(photo) } }
  });
  expect(upload.ok(), await upload.text()).toBeTruthy();

  // Alice signs up in the UI.
  await page.goto("/");
  await page.getByTestId("welcome-signup").click();
  await page.getByTestId("signup-email").fill(aliceEmail);
  await page.getByTestId("signup-password").fill(PASSWORD);
  await page.getByTestId("signup-adult").click();
  await page.getByTestId("signup-submit").click();

  // Onboarding.
  await page.getByTestId("onboarding-firstname").fill("Alice");
  await page.getByTestId("onboarding-next").click();
  await page.getByTestId("onboarding-birthdate").fill("15061995");
  await page.getByTestId("onboarding-next").click();
  await page.getByTestId("gender-woman").click();
  await page.getByTestId("onboarding-next").click();
  await page.getByTestId("seeking-man").click();
  await page.getByTestId("onboarding-next").click();

  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("photo-add").click();
  await (await chooser).setFiles(photo);
  await expect(page.getByTestId("photo-slot-0")).toBeVisible();
  await page.getByTestId("onboarding-next").click();

  await expect(page.getByTestId("onboarding-step-location")).toBeVisible();
  await page.getByTestId("onboarding-next").click(); // skip location
  await page.getByTestId("onboarding-bio").fill("J'aime les expos et les brunchs.");
  await page.getByTestId("onboarding-next").click();

  // Bob has already liked Alice, so her like creates a match.
  const alice = await apiSession(request, aliceEmail, false);
  const like = await request.post(`${API}/swipes`, { headers: bobAuth, data: { targetUserId: alice.userId, action: "like" } });
  expect(like.ok(), await like.text()).toBeTruthy();

  await expect(page.getByTestId("discover-screen")).toBeVisible();
  await expect(page.getByText("Bob, ", { exact: false }).first()).toBeVisible();
  await page.getByTestId("discover-like").click();

  await expect(page.getByTestId("match-modal")).toBeVisible();
  await page.getByTestId("match-message").click();

  await page.getByTestId("chat-input").fill("Salut Bob !");
  await page.getByTestId("chat-send").click();
  await expect(page.getByText("Salut Bob !")).toBeVisible();

  // Bob answers through the API: Alice must receive it over the WebSocket
  // (no polling while connected).
  const conversations = await (await request.get(`${API}/conversations`, { headers: bobAuth })).json();
  const conversationId = conversations.conversations[0].id as string;
  const reply = await request.post(`${API}/conversations/${conversationId}/messages`, { headers: bobAuth, data: { body: "Salut Alice, ravi du match !" } });
  expect(reply.ok()).toBeTruthy();
  await expect(page.getByText("Salut Alice, ravi du match !")).toBeVisible({ timeout: 4_000 });

  // Bob sees Alice's message as read once she has the thread open.
  await expect
    .poll(async () => (await (await request.get(`${API}/conversations/${conversationId}/messages`, { headers: bobAuth })).json()).otherLastReadAt, { timeout: 8_000 })
    .not.toBeNull();
});

test("sign-in shows a clear error on wrong credentials", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("welcome-signin").click();
  await page.getByTestId("signin-email").fill(`nobody_${unique()}@e2e.test`);
  await page.getByTestId("signin-password").fill("WrongPassword1");
  await page.getByTestId("signin-submit").click();
  await expect(page.getByTestId("signin-error")).toContainText("incorrect");
});
