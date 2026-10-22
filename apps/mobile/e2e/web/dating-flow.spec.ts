import { expect, test, type Browser, type Page } from "@playwright/test";
import { solidPng } from "./png";

const PASSWORD = "e2e-password-123";
const run = Date.now().toString(36);
// Letters only (names are validated) so that leftovers of earlier runs never match this run's labels.
const tag = run.replace(/[0-9]/g, (digit) => "abcdefghij"[Number(digit)]);

type Person = { name: string; email: string; gender: "woman" | "man"; looksFor: "man" | "woman"; color: [number, number, number] };

const ana: Person = { name: `Ana${tag}`, email: `ana.${run}@e2e.test`, gender: "woman", looksFor: "man", color: [200, 90, 110] };
const ben: Person = { name: `Ben${tag}`, email: `ben.${run}@e2e.test`, gender: "man", looksFor: "woman", color: [70, 120, 190] };

const thread = (page: Page) => page.getByTestId("chat-messages");

async function openApp(browser: Browser): Promise<Page> {
  const context = await browser.newContext({ viewport: { width: 420, height: 860 } });
  const page = await context.newPage();
  await page.goto("/");
  return page;
}

async function signUpAndOnboard(page: Page, person: Person) {
  await page.getByTestId("auth-switch").click();
  await page.getByTestId("auth-email").fill(person.email);
  await page.getByTestId("auth-password").fill(PASSWORD);
  await page.getByTestId("auth-submit").click();

  // Step 1: identity (and the 18+ rule is enforced before anything is saved).
  await expect(page.getByTestId("onboarding-screen")).toBeVisible();
  await page.getByTestId("onboarding-name").fill(person.name);
  await page.getByTestId("onboarding-day").fill("10");
  await page.getByTestId("onboarding-month").fill("04");
  await page.getByTestId("onboarding-year").fill(String(new Date().getFullYear() - 16));
  await page.getByTestId(`gender-${person.gender}`).click();
  await page.getByTestId("onboarding-next").click();
  await expect(page.getByText("Vous devez avoir au moins 18 ans")).toBeVisible();
  await page.getByTestId("onboarding-year").fill(String(new Date().getFullYear() - 30));
  await page.getByTestId("onboarding-next").click();

  // Step 2: who to meet.
  await page.getByTestId(`interested-${person.looksFor}`).click();
  await page.getByTestId("onboarding-next").click();

  // Step 3: location, from the built-in city list so no browser permission is needed.
  await page.getByTestId("city-Montréal").click();
  await page.getByTestId("onboarding-next").click();

  // Step 4: a real photo upload through the file picker.
  await page.getByTestId("onboarding-next").click();
  await expect(page.getByText("Ajoutez au moins une photo.")).toBeVisible();
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("add-photo").click();
  await (await chooser).setFiles({ name: "photo.png", mimeType: "image/png", buffer: solidPng(480, 600, person.color) });
  await expect(page.getByTestId("photo-slot-0")).toBeVisible();
  await page.getByTestId("onboarding-next").click();

  // Step 5: about me.
  await page.getByTestId("onboarding-bio").fill(`Bonjour, je suis ${person.name}.`);
  await page.getByTestId("interest-cinema").click();
  await page.getByTestId("onboarding-finish").click();
  await expect(page.getByTestId("discover-screen")).toBeVisible();
}

test("sign up, discover, like, match, chat in real time, block, delete account", async ({ browser }) => {
  const anaPage = await openApp(browser);
  const benPage = await openApp(browser);

  await signUpAndOnboard(anaPage, ana);
  await signUpAndOnboard(benPage, ben);

  // Ana opened Discover before Ben existed, so her deck may be empty: she refreshes and sees him.
  await expect(anaPage.getByTestId("discover-card").or(anaPage.getByTestId("discover-empty"))).toBeVisible();
  if (await anaPage.getByTestId("discover-empty").isVisible()) {
    await anaPage.getByText("Actualiser").click();
  }
  await expect(anaPage.getByLabel(`${ben.name}, 30 ans`)).toBeVisible();
  await anaPage.getByTestId("like-button").click();
  await expect(anaPage.getByTestId("discover-empty")).toBeVisible();
  await expect(anaPage.getByTestId("match-overlay")).toHaveCount(0);

  // Ben sees Ana, likes her back: instant match on his side...
  await expect(benPage.getByLabel(`${ana.name}, 30 ans`)).toBeVisible();
  await benPage.getByTestId("like-button").click();
  await expect(benPage.getByTestId("match-overlay")).toBeVisible();
  // ...and in real time on Ana's side, without any refresh.
  await expect(anaPage.getByTestId("match-overlay")).toBeVisible();

  // Ben opens the conversation from the match screen and writes first.
  await benPage.getByTestId("match-chat").click();
  await benPage.getByTestId("chat-input").fill("Salut Ana, ravi de ce match !");
  await benPage.getByTestId("chat-send").click();
  await expect(thread(benPage).getByText("Salut Ana, ravi de ce match !")).toBeVisible();

  // Ana dismisses the overlay and sees the unread conversation arrive live.
  await anaPage.getByTestId("match-dismiss").click();
  await anaPage.getByTestId("tab-Messages").click();
  await expect(anaPage.getByTestId(`conversation-${ben.name}`)).toBeVisible();
  await anaPage.getByTestId(`conversation-${ben.name}`).click();
  await expect(thread(anaPage).getByText("Salut Ana, ravi de ce match !")).toBeVisible();

  // Ana replies; Ben receives it live and, with the chat open, his message becomes "read" for Ana.
  await anaPage.getByTestId("chat-input").fill("Avec plaisir Ben !");
  await anaPage.getByTestId("chat-send").click();
  await expect(thread(benPage).getByText("Avec plaisir Ben !")).toBeVisible();
  await benPage.getByTestId("chat-send").waitFor();
  await benPage.getByTestId("chat-input").fill("À très vite");
  await benPage.getByTestId("chat-send").click();
  await expect(thread(anaPage).getByText("À très vite")).toBeVisible();
  await expect(thread(benPage).getByText(/· Lu/)).toBeVisible();

  // Notifications were created for both of them.
  await benPage.getByTestId("chat-back").click();
  await benPage.getByTestId("tab-Notifications").click();
  await expect(benPage.getByTestId("notification-match")).toBeVisible();

  // Ana blocks Ben from the chat menu: the conversation disappears for both.
  await anaPage.getByTestId("chat-menu").click();
  await anaPage.getByTestId("chat-block").click();
  await anaPage.getByTestId("confirm-dialog-confirm").click();
  await expect(anaPage.getByTestId("messages-empty")).toBeVisible();
  await benPage.getByTestId("tab-Messages").click();
  await expect(benPage.getByTestId("messages-empty")).toBeVisible();

  // Ben deletes his account: he lands back on the sign-in screen and cannot log in again.
  await benPage.getByTestId("tab-Me").click();
  await benPage.getByTestId("delete-account").click();
  await benPage.getByTestId("delete-password").fill(PASSWORD);
  await benPage.getByTestId("confirm-dialog-confirm").click();
  await expect(benPage.getByTestId("auth-screen")).toBeVisible();
  await benPage.getByTestId("auth-email").fill(ben.email);
  await benPage.getByTestId("auth-password").fill(PASSWORD);
  await benPage.getByTestId("auth-submit").click();
  await expect(benPage.getByText("Email ou mot de passe incorrect.")).toBeVisible();
});
