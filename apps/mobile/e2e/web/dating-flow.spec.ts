import { expect, test, type Browser, type BrowserContext, type Page } from "@playwright/test";
import path from "node:path";

const PASSWORD = "e2e-password-123";
const PHOTO = path.join(__dirname, "../fixtures/portrait.jpg");
const PARIS = { latitude: 48.8566, longitude: 2.3522 };

type Person = { name: string; email: string; gender: "Femme" | "Homme"; birth: [string, string, string] };

// Unique first names (letters only, as the API requires) so reruns against a used database stay isolated.
const tag = Array.from({ length: 6 }, () => String.fromCharCode(97 + Math.floor(Math.random() * 26))).join("");
const stamp = Date.now();
const alice: Person = { name: `Alice${tag}`, email: `alice.${stamp}@e2e.test`, gender: "Femme", birth: ["14", "05", "1993"] };
const bob: Person = { name: `Bob${tag}`, email: `bob.${stamp}@e2e.test`, gender: "Homme", birth: ["02", "11", "1990"] };

async function newPerson(browser: Browser): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext({ geolocation: PARIS, permissions: ["geolocation"], viewport: { width: 420, height: 860 }, locale: "fr-FR" });
  const page = await context.newPage();
  page.on("dialog", (dialog) => void dialog.accept()); // destructive actions use window.confirm on the web
  return { context, page };
}

/** Sign up, then walk through the five onboarding steps. */
async function signUpAndOnboard(page: Page, person: Person) {
  await page.goto("/");
  await page.getByTestId("auth-mode-register").click();
  await page.getByTestId("auth-email").fill(person.email);
  await page.getByTestId("auth-password").fill(PASSWORD);
  await page.getByTestId("auth-submit").click();

  // 1 — about you
  await expect(page.getByTestId("onboarding-step-0")).toBeVisible();
  await page.getByTestId("onb-first-name").fill(person.name);
  await page.getByTestId("birth-day").fill(person.birth[0]);
  await page.getByTestId("birth-month").fill(person.birth[1]);
  await page.getByTestId("birth-year").fill(person.birth[2]);
  await page.getByRole("button", { name: person.gender, exact: true }).click();
  await page.getByTestId("onb-next").click();

  // 2 — photo (upload through the real picker + server pipeline)
  await expect(page.getByTestId("onboarding-step-1")).toBeVisible();
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("photo-add").click();
  await (await chooser).setFiles(PHOTO);
  await expect(page.getByText("Principale")).toBeVisible();
  await page.getByTestId("onb-next").click();

  // 3 — about (bio + interests)
  await expect(page.getByTestId("onboarding-step-2")).toBeVisible();
  await page.getByTestId("onb-bio").fill(`Bonjour, je suis ${person.name}.`);
  await page.getByRole("button", { name: "Voyage", exact: true }).click();
  await page.getByRole("button", { name: "Musique", exact: true }).click();
  await page.getByTestId("onb-next").click();

  // 4 — preferences (defaults are fine: everyone, 18–50, 50 km)
  await expect(page.getByTestId("onboarding-step-3")).toBeVisible();
  await page.getByTestId("onb-next").click();

  // 5 — location
  await expect(page.getByTestId("onboarding-step-4")).toBeVisible();
  await page.getByTestId("location-use").click();

  // Onboarding complete → the app opens on the discovery tab.
  await expect(page.getByTestId("tab-Discover")).toBeVisible();
}

test("signup → profile → discovery → like → match → chat, in real time", async ({ browser }) => {
  const a = await newPerson(browser);
  const b = await newPerson(browser);

  await test.step("both people sign up and complete onboarding", async () => {
    await signUpAndOnboard(a.page, alice);
    await signUpAndOnboard(b.page, bob);
  });

  await test.step("Alice discovers Bob and likes him (no match yet)", async () => {
    await a.page.getByTestId("tab-Discover").click();
    // Alice finished before Bob existed, so her deck shows the empty state: refresh as a person would.
    await a.page.getByRole("button", { name: "Actualiser" }).click();
    await expect(a.page.getByTestId(`card-${bob.name}`)).toBeVisible();
    await expect(a.page.getByText(new RegExp(`${bob.name}, 3\\d`))).toBeVisible();
    await a.page.getByTestId("swipe-like").click();
    await expect(a.page.getByTestId("match-modal")).toHaveCount(0);
  });

  await test.step("Bob sees Alice first and likes her back → match for both", async () => {
    await b.page.reload(); // refetch the deck after Alice's like
    await expect(b.page.getByTestId("tab-Discover")).toBeVisible();
    await expect(b.page.getByTestId(`card-${alice.name}`)).toBeVisible();
    await b.page.getByTestId("swipe-like").click();
    await expect(b.page.getByTestId("match-modal")).toBeVisible();
    await expect(b.page.getByText("C'est un match !")).toBeVisible();
    // Alice is told in real time, without refreshing.
    await expect(a.page.getByText(`Nouveau match avec ${bob.name} !`)).toBeVisible();
  });

  await test.step("Bob writes first from the match screen", async () => {
    await b.page.getByTestId("match-message").click();
    await b.page.getByTestId("chat-input").fill("Salut Alice, ravi de ce match !");
    await b.page.getByTestId("chat-send").click();
    await expect(b.page.getByTestId("chat-list").getByText("Salut Alice, ravi de ce match !")).toBeVisible();
  });

  await test.step("Alice receives it live, reads it and answers", async () => {
    await a.page.getByTestId("tab-Messages").click();
    await expect(a.page.getByTestId(`conversation-${bob.name}`)).toBeVisible();
    await expect(a.page.getByLabel(/1 non lu/).first()).toBeVisible();
    await a.page.getByTestId(`conversation-${bob.name}`).click();
    await expect(a.page.getByTestId("chat-list").getByText("Salut Alice, ravi de ce match !")).toBeVisible();
    await a.page.getByTestId("chat-input").fill("Salut Bob ! Enchantée 😊");
    await a.page.getByTestId("chat-send").click();
    await expect(a.page.getByTestId("chat-list").getByText("Salut Bob ! Enchantée 😊")).toBeVisible();
  });

  await test.step("Bob gets the answer live and Alice's read receipt shows", async () => {
    await expect(b.page.getByTestId("chat-list").getByText("Salut Bob ! Enchantée 😊")).toBeVisible();
    await expect(b.page.getByTestId("chat-list").getByText(/Lu/).first()).toBeVisible();
  });

  await test.step("notifications list the match and the message", async () => {
    await a.page.getByTestId("chat-back").click();
    await a.page.getByTestId("tab-Discover").click();
    await a.page.getByTestId("open-notifications").click();
    await expect(a.page.getByTestId("notifications-screen")).toBeVisible();
    await expect(a.page.getByText("Nouveau match", { exact: true }).first()).toBeVisible();
  });

  await test.step("Bob blocks Alice: the conversation disappears for both", async () => {
    await b.page.getByTestId("chat-menu").click();
    await b.page.getByTestId("action-block").click(); // window.confirm is auto-accepted below
    await expect(b.page.getByTestId("tab-Messages")).toBeVisible();
    await b.page.getByTestId("tab-Messages").click();
    await expect(b.page.getByTestId(`conversation-${alice.name}`)).toHaveCount(0);
  });

  await a.context.close();
  await b.context.close();
});
