import { expect, test } from "@playwright/test";
import { call, createMember, deleteMember, login, makePng, uniqueEmail, PASSWORD, type Member } from "./api";

/**
 * Critical path, through the real UI:
 * sign-up -> onboarding (name, birthdate, gender, preferences, photo upload,
 * geolocation, bio & interests) -> discovery -> like -> match -> chat with
 * real-time reply and read receipt.
 */
test("sign-up to match to real-time conversation", async ({ page }) => {
  const peer: Member = await createMember("Basile", "man", ["woman"]);
  const email = uniqueEmail("e2e_ui");
  let myToken: string | null = null;

  try {
    await page.goto("/");
    await page.getByTestId("welcome-signup").click();

    await page.getByTestId("signup-email").fill(email);
    await page.getByTestId("signup-password").fill(PASSWORD);
    await page.getByTestId("signup-adult").click();
    await page.getByTestId("signup-submit").click();

    // Onboarding
    await expect(page.getByTestId("onboarding-name")).toBeVisible();
    await page.getByTestId("onboarding-firstname").fill("Alix");
    await page.getByTestId("onboarding-next").click();

    await expect(page.getByTestId("onboarding-birthdate")).toBeVisible();
    await page.getByTestId("birth-day").fill("14");
    await page.getByTestId("birth-month").fill("02");
    await page.getByTestId("birth-year").fill("1995");
    await page.getByTestId("onboarding-next").click();

    await expect(page.getByTestId("onboarding-gender")).toBeVisible();
    await page.getByTestId("gender-woman").click();
    await page.getByTestId("onboarding-next").click();

    await expect(page.getByTestId("onboarding-interestedIn")).toBeVisible();
    await page.getByTestId("interested-man").click();
    await page.getByTestId("onboarding-next").click();

    await expect(page.getByTestId("onboarding-photos")).toBeVisible();
    const chooser = page.waitForEvent("filechooser");
    await page.getByTestId("photo-add").click();
    await (await chooser).setFiles({ name: "me.png", mimeType: "image/png", buffer: makePng() });
    await expect(page.getByTestId("photo-0")).toBeVisible();
    await page.getByTestId("onboarding-next").click();

    await expect(page.getByTestId("onboarding-location")).toBeVisible();
    await page.getByTestId("location-city").fill("Paris");
    await page.getByTestId("location-use").click();
    await expect(page.getByText("Position enregistrée")).toBeVisible();
    await page.getByTestId("onboarding-next").click();

    await expect(page.getByTestId("onboarding-about")).toBeVisible();
    await page.getByTestId("onboarding-bio").fill("Cafés, randos et concerts.");
    await page.getByTestId("interest-coffee").click();
    await page.getByTestId("onboarding-next").click();

    // The peer likes the new member first, so liking back creates a match.
    myToken = await login(email);
    const me = await call<{ id: string }>("/me", { token: myToken });
    const like = await call("/swipes", { method: "POST", token: peer.token, body: { targetUserId: me.data.id, action: "like" } });
    expect(like.status).toBe(200);

    // Discovery
    await expect(page.getByTestId("discover-screen")).toBeVisible();
    // Other members may exist in a shared database: pass them until the peer shows up.
    const peerCard = page.getByTestId(`swipe-card-${peer.id}`);
    for (let i = 0; i < 30 && !(await peerCard.isVisible()); i++) {
      await expect(page.getByTestId("swipe-card-name")).toBeVisible();
      await page.getByTestId("discover-pass").click();
      await page.waitForTimeout(500);
    }
    await expect(peerCard).toBeVisible();
    await expect(page.getByTestId("swipe-card-name")).toContainText("Basile");
    await page.getByTestId("discover-like").click();

    // Match
    await expect(page.getByTestId("match-modal")).toBeVisible();
    await page.getByTestId("match-message").click();

    // Chat
    await expect(page.getByTestId("chat-thread")).toBeVisible();
    await page.getByTestId("chat-input").fill("Bonjour Basile !");
    await page.getByTestId("chat-send").click();
    await expect(page.getByTestId("message-mine").last()).toContainText("Bonjour Basile !");
    await expect(page.getByTestId("message-status")).toHaveText("Envoyé");

    const thread = await call<{ messages: { content: string }[] }>(`/chats/${me.data.id}/messages`, { token: peer.token });
    expect(thread.data.messages.map((m) => m.content)).toContain("Bonjour Basile !");

    // Real-time: the peer reads and replies; the UI updates without reload.
    await call(`/chats/${me.data.id}/read`, { method: "POST", token: peer.token });
    await expect(page.getByTestId("message-status")).toHaveText("Lu");
    await call(`/chats/${me.data.id}/messages`, { method: "POST", token: peer.token, body: { content: "Salut Alix, ravi de matcher !" } });
    await expect(page.getByTestId("message-theirs").last()).toContainText("Salut Alix, ravi de matcher !");
  } finally {
    await deleteMember(peer.token);
    if (myToken) {
      await deleteMember(await login(email));
    }
  }
});

test("returning member signs in and sees the empty states", async ({ page }) => {
  const member = await createMember("Camille", "woman", ["woman"]);
  try {
    await page.goto("/");
    await page.getByTestId("welcome-signin").click();
    await page.getByTestId("signin-email").fill(member.email);
    await page.getByTestId("signin-password").fill("WrongPassword1");
    await page.getByTestId("signin-submit").click();
    await expect(page.getByTestId("signin-error")).toBeVisible();

    await page.getByTestId("signin-password").fill(PASSWORD);
    await page.getByTestId("signin-submit").click();
    await expect(page.getByTestId("discover-screen")).toBeVisible();

    await page.getByTestId("tab-messages").click();
    await expect(page.getByText("Pas encore de match")).toBeVisible();
    await page.getByTestId("tab-likes").click();
    await expect(page.getByText("Pas encore de like")).toBeVisible();
    await page.getByTestId("tab-me").click();
    await expect(page.getByTestId("me-screen")).toContainText("Camille");
  } finally {
    await deleteMember(member.token);
  }
});
