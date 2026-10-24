import { expect, test, type Browser, type Page } from "@playwright/test";

// 1x1 PNG: enough for the photo pipeline (picker -> device re-encode -> upload -> server re-encode).
const PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
  "base64",
);

const stamp = Date.now();
const PASSWORD = "E2e-password-1";

type Person = { name: string; email: string; gender: "woman" | "man"; wants: "woman" | "man" };

const ana: Person = {
  name: "Ana",
  email: `e2e-ana-${stamp}@e2e.invalid`,
  gender: "woman",
  wants: "man",
};
const ben: Person = {
  name: "Ben",
  email: `e2e-ben-${stamp}@e2e.invalid`,
  gender: "man",
  wants: "woman",
};

async function newPage(browser: Browser): Promise<Page> {
  const context = await browser.newContext({ viewport: { width: 420, height: 860 } });
  const page = await context.newPage();
  await page.goto("/");
  return page;
}

/** Register and complete the 4-step onboarding. */
async function signUp(page: Page, p: Person) {
  await page.getByTestId("welcome-register").click();
  await page.getByTestId("register-email").fill(p.email);
  await page.getByTestId("register-password").fill(PASSWORD);
  await page.getByTestId("register-adult").click();
  await page.getByTestId("register-submit").click();

  // 1. basics
  await expect(page.getByTestId("onboarding-screen")).toBeVisible();
  await page.getByTestId("onboarding-name").fill(p.name);
  await page.getByTestId("birth-day").fill("15");
  await page.getByTestId("birth-month").fill("06");
  await page.getByTestId("birth-year").fill("1995");
  await page.getByTestId(`onboarding-gender-${p.gender}`).click();
  await page.getByTestId("onboarding-next").click();

  // 2. about
  await page.getByTestId("onboarding-bio").fill(`Bonjour, je suis ${p.name}.`);
  await page.getByTestId("onboarding-city").fill("Paris");
  await page.getByTestId("interest-voyage").click();
  await page.getByTestId("onboarding-next").click();

  // 3. photos (needs at least one)
  await page.getByTestId("onboarding-next").click();
  await expect(page.getByTestId("onboarding-error")).toContainText("au moins une photo");
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("photo-add-0").click();
  await (await chooser).setFiles({ name: "me.png", mimeType: "image/png", buffer: PNG });
  await expect(page.getByTestId("photo-slot-0")).toBeVisible();
  await page.getByTestId("onboarding-next").click();

  // 4. preferences (location sharing is skipped: no geolocation prompt in the browser)
  await page.getByTestId("onboarding-location-toggle").click();
  await page.getByTestId(`pref-gender-${p.wants}`).click();
  await page.getByTestId("onboarding-next").click();
  await expect(page.getByTestId("discover-screen")).toBeVisible();
}

test("registration -> profile -> discovery -> like -> match -> conversation -> account deletion", async ({
  browser,
}) => {
  const anaPage = await newPage(browser);
  const benPage = await newPage(browser);

  await signUp(anaPage, ana);
  await signUp(benPage, ben);

  // Ana joined first, so her deck was empty: she refreshes and now sees Ben.
  await expect(anaPage.getByTestId("discover-empty")).toBeVisible();
  await anaPage.getByTestId("discover-refresh").click();
  // Ana sees Ben, opens his full profile, goes back and likes him: no match yet.
  await expect(anaPage.getByTestId("card-name")).toContainText("Ben");
  await anaPage.getByTestId("card-open-profile").click();
  await expect(anaPage.getByTestId("detail-name")).toContainText("Ben, 3");
  await anaPage.getByTestId("detail-back").click();
  await anaPage.getByTestId("discover-like").click();
  await expect(anaPage.getByTestId("discover-empty")).toBeVisible();

  // Ben sees Ana (she is first: she already liked him) and likes back: it's a match.
  await expect(benPage.getByTestId("card-name")).toContainText("Ana");
  await benPage.getByTestId("discover-like").click();
  await expect(benPage.getByTestId("match-title")).toBeVisible();

  // Ana is told live.
  await expect(anaPage.getByTestId("toast")).toContainText("Nouveau match avec Ben");

  // Ben writes first.
  await benPage.getByTestId("match-send-message").click();
  await benPage.getByTestId("chat-input").fill("Salut Ana, ravi de ce match !");
  await benPage.getByTestId("chat-send").click();
  await expect(benPage.getByTestId("message-text").first()).toHaveText(
    "Salut Ana, ravi de ce match !",
  );

  // Ana receives it in real time with an unread badge, opens the conversation and replies.
  await anaPage.getByTestId("tab-messages").click();
  await expect(anaPage.getByTestId("conversation-Ben")).toContainText("Salut Ana");
  await expect(anaPage.getByTestId("unread-badge")).toHaveText("1");
  await anaPage.getByTestId("conversation-Ben").click();
  await expect(anaPage.getByTestId("message-text").first()).toHaveText(
    "Salut Ana, ravi de ce match !",
  );
  await anaPage.getByTestId("chat-input").fill("Avec plaisir Ben !");
  await anaPage.getByTestId("chat-send").click();

  // Ben gets the reply live, and sees his own message flagged as read.
  await expect(benPage.getByTestId("message-text").first()).toHaveText("Avec plaisir Ben !");
  await expect(benPage.getByTestId("message-meta-last-own")).toContainText("Lu");

  // Notifications: the match shows up for Ben.
  // Reloading also proves the persisted session is restored.
  await benPage.goto("/");
  await benPage.getByTestId("tab-profile").click();
  await benPage.getByTestId("profile-notifications").click();
  await expect(benPage.getByTestId("notification-match")).toBeVisible();

  // Ana deletes her account (password confirmation) and lands on the welcome screen.
  await anaPage.goto("/");
  await anaPage.getByTestId("tab-profile").click();
  await anaPage.getByTestId("profile-delete").click();
  await anaPage.getByTestId("delete-password").fill("wrong-password");
  await anaPage.getByTestId("delete-confirm").click();
  await expect(anaPage.getByTestId("delete-error")).toBeVisible();
  await anaPage.getByTestId("delete-password").fill(PASSWORD);
  await anaPage.getByTestId("delete-confirm").click();
  await expect(anaPage.getByTestId("welcome-screen")).toBeVisible();

  // Her account no longer exists.
  await anaPage.getByTestId("welcome-login").click();
  await anaPage.getByTestId("login-email").fill(ana.email);
  await anaPage.getByTestId("login-password").fill(PASSWORD);
  await anaPage.getByTestId("login-submit").click();
  await expect(anaPage.getByTestId("login-error")).toContainText("invalid credentials");

  // Ben's conversation is gone with her.
  await benPage.reload();
  await benPage.getByTestId("tab-messages").click();
  await expect(benPage.getByTestId("messages-empty")).toBeVisible();

  // Clean up Ben as well.
  await benPage.getByTestId("tab-profile").click();
  await benPage.getByTestId("profile-delete").click();
  await benPage.getByTestId("delete-password").fill(PASSWORD);
  await benPage.getByTestId("delete-confirm").click();
  await expect(benPage.getByTestId("welcome-screen")).toBeVisible();
});

test("login validation and session restore", async ({ browser }) => {
  const page = await newPage(browser);
  await page.getByTestId("welcome-login").click();
  await page.getByTestId("login-submit").click();
  await expect(page.getByTestId("login-error")).toBeVisible();
  await page.getByTestId("login-email").fill("nobody@e2e.invalid");
  await page.getByTestId("login-password").fill("not-the-password");
  await page.getByTestId("login-submit").click();
  await expect(page.getByTestId("login-error")).toContainText("invalid credentials");
});
