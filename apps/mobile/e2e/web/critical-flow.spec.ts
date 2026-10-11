import { expect, test } from '@playwright/test';
import type { Browser, BrowserContext, Page } from '@playwright/test';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fr } from '../../src/i18n/fr';
import { makePng } from './png';

/**
 * Drives the real UI (web build) for two brand new users: register, onboarding, discover,
 * mutual like, match modal, realtime chat, then unmatch. Needs API_URL and WEB_URL (see
 * scripts/e2e-web.mjs). Users are unique per run so the test never depends on existing data.
 */
const SHOTS_DIR = process.env.SHOTS_DIR ?? '/tmp/e2e-shots';
const run = Date.now().toString(36);
const letters = (n: number) =>
  Array.from(n.toString(26), (c) => String.fromCharCode(97 + parseInt(c, 26))).join('');
const nameFor = (label: string) => `${label}${letters(Date.now() % 100000)}`;

interface Actor {
  label: string;
  firstName: string;
  email: string;
  gender: keyof typeof fr.gender.single;
  seeks: keyof typeof fr.gender.plural;
  context: BrowserContext;
  page: Page;
  errors: string[];
}

async function newActor(
  browser: Browser,
  label: string,
  gender: Actor['gender'],
  seeks: Actor['seeks'],
): Promise<Actor> {
  const context = await browser.newContext({
    viewport: { width: 420, height: 860 },
    locale: 'fr-FR',
    baseURL: process.env.WEB_URL ?? 'http://localhost:8099',
  });
  const page = await context.newPage();
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(`pageerror: ${error.message}`));
  page.on('console', (msg) => {
    if (msg.type() === 'error') errors.push(`console: ${msg.text()}`);
  });
  return {
    label,
    firstName: nameFor(label),
    email: `${label.toLowerCase()}-${run}@e2e.test`,
    gender,
    seeks,
    context,
    page,
    errors,
  };
}

const shot = (actor: Actor, step: string) =>
  actor.page.screenshot({ path: path.join(SHOTS_DIR, `${run}-${actor.label}-${step}.png`) });

async function registerAndOnboard(actor: Actor, photo: string, birthYear: string) {
  const { page } = actor;
  await page.goto('/');
  await page.getByRole('button', { name: fr.welcome.register }).click();
  await page.getByTestId('register-email').fill(actor.email);
  await page.getByTestId('register-password').fill('Passw0rd-e2e!');
  await page.getByTestId('register-confirm').fill('Passw0rd-e2e!');
  await page.getByTestId('register-adult').click();
  await shot(actor, '01-register');
  await page.getByTestId('register-submit').click();

  // Step 1: identity
  await expect(page.getByText(fr.onboarding.steps.identity.title)).toBeVisible();
  await page.getByTestId('first-name').fill(actor.firstName);
  await page.getByTestId('birth-day').fill('14');
  await page.getByTestId('birth-month').fill('06');
  await page.getByTestId('birth-year').fill(birthYear);
  await expect(page.getByText(/Vous avez \d+ ans\./)).toBeVisible();
  await page.getByRole('radio', { name: fr.gender.single[actor.gender] }).click();
  await shot(actor, '02-identity');
  await page.getByTestId('onboarding-next').click();

  // Step 2: preferences
  await expect(page.getByText(fr.onboarding.steps.preferences.title)).toBeVisible();
  await page.getByRole('checkbox', { name: fr.gender.plural[actor.seeks] }).click();
  await shot(actor, '03-preferences');
  await page.getByTestId('onboarding-next').click();

  // Step 3: location (city only; the browser geolocation is not needed)
  await expect(page.getByText(fr.onboarding.steps.location.title)).toBeVisible();
  await page.getByLabel(fr.profile.city).fill('Lyon');
  await page.getByTestId('onboarding-next').click();

  // Step 4: bio
  await expect(page.getByText(fr.onboarding.steps.bio.title)).toBeVisible();
  await page.getByTestId('bio').fill(`Bonjour, je suis ${actor.firstName}.`);
  await page.getByTestId('onboarding-next').click();

  // Step 5: interests (saves profile + preferences)
  await expect(page.getByText(fr.onboarding.steps.interests.title)).toBeVisible();
  const chips = page.getByRole('checkbox');
  await expect(chips.first()).toBeVisible();
  await chips.nth(0).click();
  await chips.nth(1).click();
  await expect(page.getByText(fr.profile.interestsCount(2, 10))).toBeVisible();
  await shot(actor, '04-interests');
  await page.getByTestId('onboarding-next').click();

  // Step 6: photo upload through the real file chooser
  await expect(page.getByText(fr.onboarding.steps.photos.title)).toBeVisible();
  await expect(page.getByTestId('onboarding-finish')).toBeDisabled();
  const chooser = page.waitForEvent('filechooser');
  await page.getByTestId('photo-add').click();
  await (await chooser).setFiles(photo);
  await expect(page.getByRole('button', { name: fr.photos.photoLabel(1, true) })).toBeVisible();
  await expect(page.getByTestId('onboarding-finish')).toBeEnabled();
  await shot(actor, '05-photos');
  await page.getByTestId('onboarding-finish').click();

  await expect(page.getByTestId('discover-like')).toBeVisible();
  await shot(actor, '06-discover');
}

/** Passes other people's cards (left by earlier runs) until `target`'s card is on top. */
async function likeTarget(actor: Actor, target: Actor) {
  const { page } = actor;
  const card = page.getByLabel(fr.discover.openProfile(target.firstName), { exact: false });
  for (let i = 0; i < 30; i++) {
    await expect(page.getByTestId('discover-like')).toBeVisible();
    if (await card.first().isVisible()) break;
    await page.getByTestId('discover-pass').click();
    await page.waitForTimeout(400);
  }
  await expect(card.first()).toBeVisible();
  await shot(actor, '07-target-card');
  await page.getByTestId('discover-like').click();
}

test('two new users register, match, chat in realtime and unmatch', async ({ browser }) => {
  mkdirSync(SHOTS_DIR, { recursive: true });
  const dir = mkdtempSync(path.join(tmpdir(), 'ember-e2e-'));
  const photoA = path.join(dir, 'a.png');
  const photoB = path.join(dir, 'b.png');
  writeFileSync(photoA, makePng(400, 1));
  writeFileSync(photoB, makePng(400, 2));

  const a = await newActor(browser, 'Alix', 'woman', 'man');
  const b = await newActor(browser, 'Brice', 'man', 'woman');

  try {
    await registerAndOnboard(a, photoA, '1994');
    await registerAndOnboard(b, photoB, '1992');

    // A likes B: no match yet.
    await likeTarget(a, b);
    await expect(a.page.getByText(fr.match.title)).toHaveCount(0);

    // B opens the Likes tab (realtime badge), then likes A back from Discover: match modal.
    await likeTarget(b, a);
    await expect(b.page.getByText(fr.match.title)).toBeVisible();
    await shot(b, '08-match');

    // A is notified live of the new match without reloading.
    await expect(a.page.getByText(fr.realtime.newMatch(b.firstName))).toBeVisible();
    await shot(a, '08-match-toast');

    // B goes to the chat from the modal; A through the Messages tab.
    await b.page.getByRole('button', { name: fr.match.sendMessage }).click();
    const inputB = b.page.getByTestId('chat-input');
    await expect(inputB).toBeVisible();

    await a.page.getByRole('tab', { name: /^Messages/ }).click();
    await a.page.getByLabel(fr.chat.newMatchLabel(b.firstName)).click();
    const inputA = a.page.getByTestId('chat-input');
    await expect(inputA).toBeVisible();

    const hello = `Salut ${b.firstName}, ça va ? ${run}`;
    await inputA.fill(hello);
    await a.page.getByTestId('chat-send').click();
    await expect(b.page.getByText(hello)).toBeVisible();
    await shot(b, '09-chat-received');

    const reply = `Très bien ${a.firstName} ! ${run}`;
    await inputB.fill(reply);
    await b.page.getByTestId('chat-send').click();
    await expect(a.page.getByText(reply)).toBeVisible();
    await expect(a.page.getByText(hello)).toBeVisible();
    await shot(a, '10-chat-reply');

    // A unmatches through the chat menu; B's conversation disappears live.
    await a.page.getByRole('button', { name: fr.chat.menu }).click();
    await shot(a, '11-menu');
    await a.page.getByRole('button', { name: fr.safety.unmatch }).click();
    await a.page.getByRole('button', { name: fr.safety.unmatch }).last().click();
    await expect(a.page.getByTestId('chat-input')).toHaveCount(0);
    await expect(b.page.getByText(fr.realtime.conversationRemoved)).toBeVisible();
    await shot(b, '12-unmatched');

    for (const actor of [a, b]) {
      expect(actor.errors, `${actor.label} console/page errors`).toEqual([]);
    }
  } finally {
    await a.context.close();
    await b.context.close();
  }
});
