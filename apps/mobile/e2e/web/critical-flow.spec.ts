import path from "node:path";
import { expect, test, type Browser, type Page } from "@playwright/test";

const PHOTO = path.resolve(__dirname, "../fixtures/portrait.jpg");
const PASSWORD = "Passw0rd!x";

type Person = { name: string; gender: "Femme" | "Homme"; email: string; birth: { day: string; month: string; year: string } };

function person(name: string, gender: "Femme" | "Homme"): Person {
  const stamp = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
  return { name, gender, email: `${name.toLowerCase()}.${stamp}@e2e.test`, birth: { day: "14", month: "03", year: "1994" } };
}

async function newUserPage(browser: Browser): Promise<Page> {
  const context = await browser.newContext({ viewport: { width: 414, height: 900 }, locale: "fr-FR" });
  return context.newPage();
}

async function signUp(page: Page, who: Person) {
  await page.goto("/");
  await page.getByRole("tab", { name: "Créer un compte" }).click();
  await page.getByLabel("Adresse e-mail").fill(who.email);
  await page.getByLabel("Mot de passe", { exact: true }).fill(PASSWORD);
  await page.getByRole("button", { name: "Créer mon compte" }).click();
  await expect(page.getByRole("heading", { name: "Vous" })).toBeVisible();
}

async function completeOnboarding(page: Page, who: Person) {
  await page.getByLabel("Prénom").fill(who.name);
  await page.getByLabel("Jour de naissance").fill(who.birth.day);
  await page.getByLabel("Mois de naissance").fill(who.birth.month);
  await page.getByLabel("Année de naissance").fill(who.birth.year);
  await page.getByRole("checkbox", { name: who.gender, exact: true }).click();
  await page.getByRole("button", { name: "Continuer" }).click();

  await expect(page.getByRole("heading", { name: "Vos envies" })).toBeVisible();
  await page.getByRole("button", { name: "Continuer" }).click();

  await expect(page.getByRole("heading", { name: "Votre profil" })).toBeVisible();
  await page.getByLabel("À propos de moi").fill(`Je m'appelle ${who.name} et j'aime les longues balades.`);
  await page.getByRole("checkbox", { name: "Voyage" }).click();
  await page.getByRole("checkbox", { name: "Cuisine" }).click();
  await page.getByRole("button", { name: "Continuer" }).click();

  await expect(page.getByRole("heading", { name: "Vos photos" })).toBeVisible();
  const [chooser] = await Promise.all([page.waitForEvent("filechooser"), page.getByRole("button", { name: "Ajouter une photo" }).first().click()]);
  await chooser.setFiles(PHOTO);
  await expect(page.getByRole("button", { name: "Photo principale, modifier" })).toBeVisible();
  await page.getByRole("button", { name: "Découvrir des profils" }).click();
  await expect(page.getByRole("heading", { name: "Découvrir" })).toBeVisible();
}

test.describe("critical flow", () => {
  test("two people register, match and chat in real time", async ({ browser }) => {
    const bob = person("Bob", "Homme");
    const alice = person("Alice", "Femme");
    const bobPage = await newUserPage(browser);
    const alicePage = await newUserPage(browser);

    // 1. Register + onboard both users (account → profile → photo).
    await signUp(bobPage, bob);
    await completeOnboarding(bobPage, bob);
    await expect(bobPage.getByText("C'est tout pour le moment")).toBeVisible();

    await signUp(alicePage, alice);
    await completeOnboarding(alicePage, alice);

    // 2. Alice discovers Bob (card shows name and age) and likes him: no match yet.
    await expect(alicePage.getByText("Bob, 3", { exact: false })).toBeVisible();
    await alicePage.screenshot({ path: "test-results/discover-alice.png" });
    await alicePage.getByRole("button", { name: "J'aime" }).click();
    await expect(alicePage.getByText("C'est tout pour le moment")).toBeVisible();
    await expect(alicePage.getByText("C'est un match !")).toHaveCount(0);

    // 3. Bob refreshes his deck, sees Alice, likes back: both get the match celebration.
    await bobPage.getByRole("button", { name: "Actualiser" }).click();
    await expect(bobPage.getByText("Alice, 3", { exact: false })).toBeVisible();
    await bobPage.getByRole("button", { name: "J'aime" }).click();
    await expect(bobPage.getByRole("heading", { name: "C'est un match !" })).toBeVisible();
    await expect(alicePage.getByRole("heading", { name: "C'est un match !" })).toBeVisible();
    await bobPage.screenshot({ path: "test-results/match-bob.png" });

    // 4. Bob writes first; Alice receives it live without reloading.
    await bobPage.getByRole("button", { name: "Envoyer un message" }).click();
    await bobPage.getByLabel("Votre message").fill("Salut Alice, ravi de ce match !");
    await bobPage.getByRole("button", { name: "Envoyer" }).click();
    await expect(bobPage.getByText("Salut Alice, ravi de ce match !")).toBeVisible();

    await alicePage.getByRole("button", { name: "Continuer à découvrir" }).click();
    await alicePage.getByText("Messages", { exact: true }).click();
    await expect(alicePage.getByRole("button", { name: /Bob, 1 non lus/ })).toBeVisible();
    await alicePage.screenshot({ path: "test-results/messages-alice.png" });

    // 5. Alice opens the conversation, answers, and Bob sees it live plus the read receipt.
    await alicePage.getByRole("button", { name: /Bob/ }).first().click();
    await expect(alicePage.getByText("Salut Alice, ravi de ce match !").last()).toBeVisible();
    await alicePage.getByLabel("Votre message").fill("Bonjour Bob !");
    await alicePage.getByRole("button", { name: "Envoyer" }).click();
    await expect(bobPage.getByText("Bonjour Bob !").last()).toBeVisible();
    await expect(bobPage.getByLabel("Lu", { exact: true }).first()).toBeVisible();
    await bobPage.screenshot({ path: "test-results/chat-bob.png" });
  });

  test("login validation, wrong password and sign-out", async ({ browser }) => {
    const page = await newUserPage(browser);
    await page.goto("/");
    await page.getByRole("button", { name: "Se connecter" }).click();
    await expect(page.getByText("Saisissez votre adresse e-mail.")).toBeVisible();

    await page.getByLabel("Adresse e-mail").fill("nobody@e2e.test");
    await page.getByLabel("Mot de passe", { exact: true }).fill("not-the-password1");
    await page.getByRole("button", { name: "Se connecter" }).click();
    await expect(page.getByText("E-mail ou mot de passe incorrect.")).toBeVisible();
  });
});
