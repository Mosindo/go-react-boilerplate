import { expect, test } from "@playwright/test";
import { completeOnboarding, newPlayer, person, register, watch } from "./helpers";

// Critical journey, through the real UI, API and database:
// sign up -> profile -> discovery -> like -> match -> real-time conversation -> safety -> account deletion.
test("two people meet, match and chat in real time", async ({ browser }) => {
  const alice = person("Alice", "woman", "man");
  const bob = person("Bob", "man", "woman");
  const a = await newPlayer(browser, "alice");
  const b = await newPlayer(browser, "bob");

  // Alice signs up and completes her profile; nobody is there yet.
  await register(a.page, alice);
  await completeOnboarding(a.page, alice, 20);
  await expect(a.page.getByTestId("deck-empty")).toBeVisible();

  // Bob signs up too and discovers Alice (she matches his preferences, and he hers).
  await register(b.page, bob);
  await completeOnboarding(b.page, bob, 120);
  await expect(b.page.getByTestId("deck-card")).toBeVisible();
  await expect(b.page.getByTestId("deck-card")).toContainText("Alice, 3");
  await b.page.screenshot({ path: ".e2e/screens/discover.png" });

  // The detail view shows the full profile.
  await b.page.getByTestId("swipe-info").click();
  await expect(b.page.getByTestId("person-screen")).toContainText("Je m'appelle Alice");
  await b.page.screenshot({ path: ".e2e/screens/person.png" });
  await b.page.getByTestId("nav-back").click();

  // Bob likes Alice. A one-sided like is not a match.
  await b.page.getByTestId("swipe-like").click();
  await expect(b.page.getByTestId("deck-empty")).toBeVisible();
  await expect(b.page.getByTestId("match-overlay")).toHaveCount(0);

  // Alice refreshes: she sees Bob, likes him back and gets the match overlay.
  await a.page.getByTestId("deck-refresh").click();
  await expect(a.page.getByTestId("deck-card")).toContainText("Bob");
  await a.page.getByTestId("swipe-like").click();
  await expect(a.page.getByTestId("match-overlay")).toBeVisible();
  await a.page.screenshot({ path: ".e2e/screens/match.png" });

  // Bob is told in real time (toast) and gets a notification.
  await expect(b.page.getByTestId("toast")).toContainText("match avec Alice");

  // Alice opens the conversation and writes.
  await a.page.getByTestId("match-message").click();
  await expect(a.page.getByTestId("chat-screen")).toBeVisible();
  await a.page.getByTestId("chat-input").fill("Salut Bob, ravie de te rencontrer !");
  await a.page.getByTestId("chat-send").click();
  await expect(a.page.getByTestId("message-mine")).toContainText("Salut Bob");

  // Bob receives it without reloading: unread badge, then the thread.
  await b.page.getByTestId("tab-messages").click();
  await expect(b.page.getByTestId("messages-screen")).toContainText("Salut Bob, ravie de te rencontrer !");
  await b.page.screenshot({ path: ".e2e/screens/messages.png" });
  await b.page.getByTestId("conversation-Alice").click();
  await expect(b.page.getByTestId("message-theirs")).toContainText("Salut Bob");
  await b.page.getByTestId("chat-input").fill("Salut Alice ! Moi aussi :)");
  await b.page.getByTestId("chat-send").click();

  // Alice sees the reply live, and Bob's read receipt on her message.
  await expect(a.page.getByTestId("message-theirs")).toContainText("Moi aussi");
  await expect(a.page.getByTestId("message-mine")).toContainText("Lu");
  await a.page.screenshot({ path: ".e2e/screens/chat.png" });

  // Bob blocks Alice: the conversation disappears for both of them.
  await b.page.getByTestId("chat-menu").click();
  await b.page.getByTestId("menu-block").click();
  await b.page.getByTestId("confirm-accept").click();
  await expect(b.page.getByTestId("messages-screen")).toBeVisible();
  await expect(b.page.getByTestId("messages-empty")).toBeVisible();

  // Alice deletes her account with her password and lands back on the sign-in screen.
  await a.page.getByTestId("nav-back").click();
  await a.page.getByTestId("tab-profile").click();
  await a.page.getByTestId("open-settings").click();
  await a.page.getByTestId("open-delete-account").click();
  await a.page.getByTestId("delete-password").fill(alice.password);
  await a.page.getByTestId("confirm-accept").click();
  await expect(a.page.getByTestId("auth-screen")).toBeVisible();

  // The deleted account can no longer sign in.
  await a.page.getByTestId("auth-email").fill(alice.email);
  await a.page.getByTestId("auth-password").fill(alice.password);
  await a.page.getByTestId("auth-submit").click();
  await expect(a.page.getByText("invalid credentials")).toBeVisible();

  await a.context.close();
  await b.context.close();
});

test("sign-in validation and account recovery entry points", async ({ page }) => {
  watch(page, "visitor");
  await page.goto("/");
  await page.getByTestId("auth-submit").click();
  await expect(page.getByText("Adresse e-mail invalide.")).toBeVisible();
  await page.getByTestId("tab-register").click();
  await page.getByTestId("auth-email").fill("someone@e2e.test");
  await page.getByTestId("auth-password").fill("correct-horse-1");
  await page.getByTestId("auth-submit").click();
  await expect(page.getByText("Vous devez avoir 18 ans ou plus.")).toBeVisible();

  await page.getByTestId("tab-login").click();
  await page.getByTestId("auth-forgot").click();
  await expect(page.getByTestId("forgot-screen")).toBeVisible();
  await page.getByTestId("forgot-email").fill("someone@e2e.test");
  await page.getByTestId("forgot-send").click();
  await expect(page.getByTestId("forgot-code")).toBeVisible();
});
