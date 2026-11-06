import path from "node:path";
import { expect, test, type Browser, type BrowserContext, type Page } from "@playwright/test";

const PHOTO = path.join(__dirname, "fixtures", "photo.jpg");
const PASSWORD = "E2e-password-1";
const run = Date.now().toString(36);

type Who = { page: Page; context: BrowserContext; email: string; name: string };

async function open(browser: Browser, name: string, geolocation = false): Promise<Who> {
  const context = await browser.newContext({
    viewport: { width: 420, height: 860 },
    ...(geolocation ? { geolocation: { latitude: 48.8566, longitude: 2.3522 }, permissions: ["geolocation"] } : {})
  });
  const page = await context.newPage();
  page.on("pageerror", (e) => console.log(`[${name}] pageerror:`, e.message));
  await page.goto("/");
  return { page, context, email: `${name.toLowerCase()}.${run}@e2e.test`, name };
}

async function signUpAndOnboard(
  who: Who,
  opts: { gender: "woman" | "man" | "nonbinary"; seeking: "woman" | "man" | "nonbinary"; useGps?: boolean }
) {
  const { page } = who;
  await page.getByTestId("auth-toggle").click();
  await page.getByTestId("auth-email").fill(who.email);
  await page.getByTestId("auth-password").fill(PASSWORD);
  await page.getByTestId("auth-submit").click();

  // 1/5 basics — validation first
  await expect(page.getByTestId("onboarding-step")).toContainText("Étape 1 sur 5");
  await page.getByTestId("onboarding-next").click();
  await expect(page.getByText("Indiquez votre prénom")).toBeVisible();
  await page.getByTestId("basics-firstname").fill(who.name);
  await page.getByTestId("basics-birthdate").fill("01/01/2015");
  await page.getByTestId(`gender-${opts.gender}`).click();
  await page.getByTestId("onboarding-next").click();
  await expect(page.getByText("réservé aux personnes majeures")).toBeVisible();
  await page.getByTestId("basics-birthdate").fill("12/05/1994");
  await page.getByTestId("onboarding-next").click();

  // 2/5 location
  await expect(page.getByTestId("onboarding-step")).toContainText("Étape 2 sur 5");
  if (opts.useGps) await page.getByTestId("location-use").click();
  else await page.getByTestId("city-Paris").click();
  await expect(page.getByTestId("location-set")).toBeVisible();
  await page.getByTestId("onboarding-next").click();

  // 3/5 about
  await page.getByTestId("about-bio").fill(`Bonjour, moi c'est ${who.name} !`);
  await page.getByTestId("interest-travel").click();
  await page.getByTestId("interest-music").click();
  await page.getByTestId("onboarding-next").click();

  // 4/5 preferences: only the wanted gender
  for (const g of ["woman", "man", "nonbinary"]) {
    if (g !== opts.seeking) await page.getByTestId(`seek-${g}`).click();
  }
  await page.getByTestId("onboarding-next").click();

  // 5/5 photo — finishing without one is refused
  await expect(page.getByTestId("onboarding-step")).toContainText("Étape 5 sur 5");
  await page.getByTestId("onboarding-finish").click();
  await expect(page.getByTestId("onboarding-error")).toContainText("au moins une photo");
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("photo-add").click();
  await (await chooser).setFiles(PHOTO);
  await expect(page.getByTestId("photo-slot-0")).toBeVisible();
  await page.getByTestId("onboarding-finish").click();
  await expect(page.getByTestId("tab-discover")).toBeVisible();
}

test.describe.configure({ mode: "serial" });

test("signup → profile → discovery → like → match → realtime chat → block → delete account", async ({ browser }) => {
  const alice = await open(browser, "Alice", true);
  const bob = await open(browser, "Bob");

  await test.step("Alice signs up and builds her profile (GPS location)", async () => {
    await signUpAndOnboard(alice, { gender: "woman", seeking: "man", useGps: true });
  });
  await test.step("Bob signs up and builds his profile (city picker)", async () => {
    await signUpAndOnboard(bob, { gender: "man", seeking: "woman" });
  });

  await test.step("Alice discovers Bob and can open his full profile", async () => {
    await alice.page.reload();
    await expect(alice.page.getByTestId("deck-name")).toContainText("Bob, 3");
    await alice.page.getByTestId("swipe-info").click();
    await expect(alice.page.getByTestId("detail-name")).toContainText("Bob");
    await expect(alice.page.getByText("Bonjour, moi c'est Bob !")).toBeVisible();
    await expect(alice.page.getByTestId("profile-detail").getByText("À moins de 5 km")).toBeVisible();
    await alice.page.getByTestId("detail-like").click(); // like from the detail view
    await expect(alice.page.getByTestId("tab-discover")).toBeVisible();
    await expect(alice.page.getByText("Vous avez tout vu pour le moment")).toBeVisible();
  });

  await test.step("Bob likes Alice back: it is a match", async () => {
    await bob.page.reload();
    await expect(bob.page.getByTestId("deck-name")).toContainText("Alice, 3");
    await bob.page.getByTestId("swipe-like").click();
    await expect(bob.page.getByTestId("match-modal")).toBeVisible();
    await bob.page.getByTestId("match-message").click();
    await expect(bob.page.getByTestId("chat-input")).toBeVisible();
  });

  await test.step("Alice is told about the match in real time, without reloading", async () => {
    await expect(alice.page.getByTestId("toast")).toContainText("Bob");
    await alice.page.getByTestId("tab-messages").click();
    await expect(alice.page.getByTestId("new-match-Bob")).toBeVisible();
  });

  await test.step("Realtime messaging with read receipts", async () => {
    await bob.page.getByTestId("chat-input").fill("Salut Alice, ravi du match !");
    await bob.page.getByTestId("chat-send").click();
    await expect(bob.page.getByTestId("bubble-mine")).toContainText("ravi du match");

    // Alice sees an unread conversation appear live, opens it and answers.
    await expect(alice.page.getByTestId("conversation-Bob")).toContainText("ravi du match");
    await alice.page.getByTestId("conversation-Bob").click();
    await expect(alice.page.getByTestId("bubble-theirs")).toContainText("ravi du match");
    await alice.page.getByTestId("chat-input").fill("Moi aussi 😊");
    await alice.page.getByTestId("chat-send").click();

    await expect(bob.page.getByTestId("bubble-theirs")).toContainText("Moi aussi");
    // Alice has been reading the thread, so Bob's message shows as read for Bob.
    await expect(bob.page.getByText("· Lu")).toBeVisible();
  });

  await test.step("Alice has notifications for the match and the message", async () => {
    await alice.page.goBack();
    await alice.page.getByTestId("tab-notifications").click();
    await expect(alice.page.getByTestId("notification-match")).toBeVisible();
    await alice.page.getByTestId("notifications-read-all").click();
    await expect(alice.page.getByTestId("notifications-read-all")).toBeHidden();
  });

  await test.step("Bob reports and blocks Alice from the chat", async () => {
    await bob.page.getByTestId("chat-menu").click();
    await bob.page.getByTestId("chat-report").click();
    await bob.page.getByTestId("reason-spam").click();
    await bob.page.getByTestId("report-submit").click();
    await expect(bob.page.getByTestId("toast")).toContainText("signalement");
    await expect(bob.page.getByTestId("chat-input")).toBeHidden();
    await bob.page.getByTestId("tab-messages").click();
    await expect(bob.page.getByText("Pas encore de match")).toBeVisible();
  });

  await test.step("Alice's conversation disappears in real time", async () => {
    await alice.page.getByTestId("tab-messages").click();
    await expect(alice.page.getByText("Pas encore de match")).toBeVisible();
  });

  await test.step("Alice logs out, fails a login, logs back in", async () => {
    await alice.page.getByTestId("tab-profile").click();
    await expect(alice.page.getByTestId("me-name")).toContainText("Alice");
    await alice.page.getByTestId("logout").click();
    await alice.page.getByTestId("auth-email").fill(alice.email);
    await alice.page.getByTestId("auth-password").fill("wrong-password");
    await alice.page.getByTestId("auth-submit").click();
    await expect(alice.page.getByTestId("auth-error")).toContainText("incorrect");
    await alice.page.getByTestId("auth-password").fill(PASSWORD);
    await alice.page.getByTestId("auth-submit").click();
    await expect(alice.page.getByTestId("tab-discover")).toBeVisible();
  });

  await test.step("Privacy: pausing the profile is reflected", async () => {
    await alice.page.getByTestId("tab-profile").click();
    await alice.page.getByTestId("menu-privacy").click();
    await alice.page.getByTestId("toggle-discoverable").click({ force: true });
    await alice.page.goBack();
    await expect(alice.page.getByTestId("paused-notice")).toBeVisible();
  });

  await test.step("Alice deletes her account for good", async () => {
    await alice.page.getByTestId("delete-account").click();
    await alice.page.getByTestId("delete-password").fill("wrong-password");
    await alice.page.getByTestId("confirm-yes").click();
    await expect(alice.page.getByText("incorrect")).toBeVisible();
    await alice.page.getByTestId("delete-password").fill(PASSWORD);
    await alice.page.getByTestId("confirm-yes").click();
    await expect(alice.page.getByTestId("auth-submit")).toBeVisible();
    await alice.page.getByTestId("auth-email").fill(alice.email);
    await alice.page.getByTestId("auth-password").fill(PASSWORD);
    await alice.page.getByTestId("auth-submit").click();
    await expect(alice.page.getByTestId("auth-error")).toContainText("incorrect");
  });

  await alice.context.close();
  await bob.context.close();
});

test("swipe gesture passes a profile", async ({ browser }) => {
  const carol = await open(browser, "Carol");
  const dan = await open(browser, "Dan");
  // Distinct genders keep this scenario isolated from the other test's users.
  await signUpAndOnboard(carol, { gender: "woman", seeking: "nonbinary" });
  await signUpAndOnboard(dan, { gender: "nonbinary", seeking: "woman" });

  await carol.page.reload();
  const card = carol.page.getByTestId("deck-top-card");
  await expect(card).toBeVisible();
  const box = await card.boundingBox();
  if (!box) throw new Error("no card");
  await carol.page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await carol.page.mouse.down();
  await carol.page.mouse.move(box.x - 150, box.y + box.height / 2 + 10, { steps: 10 });
  await carol.page.mouse.up();
  await expect(carol.page.getByText("Vous avez tout vu pour le moment")).toBeVisible();

  // Passed profiles stay hidden until explicitly restored.
  await carol.page.reload();
  await expect(carol.page.getByText("Vous avez tout vu pour le moment")).toBeVisible();
  await carol.page.getByTestId("discover-restore").click();
  await expect(carol.page.getByTestId("deck-name")).toContainText("Dan");
  await carol.context.close();
  await dan.context.close();
});
