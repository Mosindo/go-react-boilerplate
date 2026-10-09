import { expect, test, type Browser, type BrowserContext, type Page } from '@playwright/test';

/**
 * Full two-user journey against the exported web build and a REAL API (E2E_API_URL).
 *   register A -> onboarding -> register B -> onboarding -> A likes B -> B likes A -> match
 *   -> A messages B (realtime) -> B reads (A sees "Lu") -> reply -> report/block/unblock -> relogin
 */

const PASSWORD = 'Lumo-e2e-passw0rd';

function letters(n: number): string {
  let out = '';
  for (let i = 0; i < n; i++) out += String.fromCharCode(97 + Math.floor(Math.random() * 26));
  return out;
}
const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

type Persona = {
  name: string;
  email: string;
  birth: [string, string, string];
  gender: 'man' | 'woman' | 'nonbinary';
  interestedIn: 'man' | 'woman' | 'nonbinary';
  location: 'gps' | 'city';
};

const run = letters(6);
const alice: Persona = {
  name: `Alice${cap(run)}`,
  email: `alice-${Date.now()}-${run}@example.com`,
  birth: ['14', '03', '1995'],
  gender: 'woman',
  interestedIn: 'man',
  location: 'gps',
};
const bruno: Persona = {
  name: `Bruno${cap(run)}`,
  email: `bruno-${Date.now()}-${run}@example.com`,
  birth: ['02', '11', '1992'],
  gender: 'man',
  interestedIn: 'woman',
  location: 'city',
};

async function newUserContext(browser: Browser): Promise<{ ctx: BrowserContext; page: Page }> {
  const ctx = await browser.newContext({
    viewport: { width: 430, height: 900 },
    locale: 'fr-FR',
    geolocation: { latitude: 48.8566, longitude: 2.3522 },
    permissions: ['geolocation'],
  });
  const page = await ctx.newPage();
  page.on('pageerror', (e) => console.warn('[pageerror]', e.message));
  page.on('response', (r) => {
    if (r.status() >= 400) console.warn(`[http ${r.status()}] ${r.request().method()} ${r.url()}`);
  });
  return { ctx, page };
}

/** A real JPEG, generated in the browser so the test needs no image dependency. */
async function makeJpeg(page: Page, hue: number): Promise<Buffer> {
  const base64 = await page.evaluate((h) => {
    const canvas = document.createElement('canvas');
    canvas.width = 900;
    canvas.height = 1200;
    const g = canvas.getContext('2d')!;
    const grad = g.createLinearGradient(0, 0, 900, 1200);
    grad.addColorStop(0, `hsl(${h},70%,55%)`);
    grad.addColorStop(1, `hsl(${(h + 60) % 360},70%,35%)`);
    g.fillStyle = grad;
    g.fillRect(0, 0, 900, 1200);
    g.fillStyle = 'rgba(255,255,255,0.8)';
    g.beginPath();
    g.arc(450, 450, 200, 0, Math.PI * 2);
    g.fill();
    return canvas.toDataURL('image/jpeg', 0.9).split(',')[1]!;
  }, hue);
  return Buffer.from(base64, 'base64');
}

/** A chat bubble (not toasts / list previews) containing `text`. */
const bubble = (page: Page, text: string) =>
  page
    .locator('[data-testid^="message-"]:not([data-testid="message-list"])')
    .filter({ hasText: text });

async function choose(page: Page, testId: string) {
  const el = page.getByTestId(testId);
  const checked = (await el.getAttribute('aria-checked')) === 'true';
  if (!checked) await el.click();
}

async function register(page: Page, p: Persona) {
  await page.goto('/');
  await page.getByTestId('welcome-register').click();
  await page.getByTestId('register-email').fill(p.email);
  await page.getByTestId('register-password').fill(PASSWORD);
  await page.getByTestId('register-adult').click();
  await page.getByTestId('register-submit').click();
  await expect(page.getByTestId('onboarding-screen')).toBeVisible();
}

async function onboard(
  page: Page,
  p: Persona,
  hue: number,
  shot: (name: string) => Promise<unknown> = async () => undefined,
) {
  // 1. basics
  await page.getByTestId('onboarding-firstName').fill(p.name);
  await page.getByTestId('birth-day').fill(p.birth[0]);
  await page.getByTestId('birth-month').fill(p.birth[1]);
  await page.getByTestId('birth-year').fill(p.birth[2]);
  await choose(page, `onboarding-gender-${p.gender}`);
  await shot('ob-1-basics');
  await page.getByTestId('onboarding-next').click();

  // 2. location (GPS, or a preset city)
  await expect(page.getByTestId('onboarding-city')).toBeVisible();
  if (p.location === 'gps') {
    await page.getByTestId('location-use').click();
    await expect(page.getByTestId('location-status')).toContainText('Position enregistrée');
    await page.getByTestId('onboarding-city').fill('Paris');
  } else {
    await page.getByTestId('city-preset-paris').click();
    await expect(page.getByTestId('onboarding-city')).toHaveValue('Paris');
  }
  await shot('ob-2-location');
  await page.getByTestId('onboarding-next').click();

  // 3. photos (file chooser)
  await expect(page.getByTestId('photo-add')).toBeVisible();
  const jpeg = await makeJpeg(page, hue);
  const [chooser] = await Promise.all([
    page.waitForEvent('filechooser'),
    page.getByTestId('photo-add').click(),
  ]);
  await chooser.setFiles({ name: 'photo.jpg', mimeType: 'image/jpeg', buffer: jpeg });
  await expect(page.getByTestId('photo-tile-0')).toBeVisible();
  await shot('ob-3-photos');
  await page.getByTestId('onboarding-next').click();

  // 4. bio + interests
  await page
    .getByTestId('onboarding-bio')
    .fill(`Bonjour, je suis ${p.name}. Je cherche des conversations vraies.`);
  const chips = page.locator('[data-testid^="interest-"]');
  await expect(chips.first()).toBeVisible();
  await chips.nth(0).click();
  await chips.nth(1).click();
  await chips.nth(2).click();
  await shot('ob-4-bio');
  await page.getByTestId('onboarding-next').click();

  // 5. preferences
  await expect(page.getByTestId('pref-minAge-value')).toBeVisible();
  await choose(page, `pref-interestedIn-${p.interestedIn}`);
  await shot('ob-5-prefs');
  await page.getByTestId('onboarding-finish').click();
  await expect(page.getByTestId('tab-discover')).toBeVisible();
}

/** Passes other people until the card of `name` is on top. */
async function waitForCardOf(page: Page, name: string) {
  const card = page.getByTestId('card-name');
  const empty = page.getByTestId('discover-empty');
  for (let i = 0; i < 40; i++) {
    await expect(card.or(empty)).toBeVisible();
    if (await empty.isVisible()) throw new Error(`${name} never showed up in the deck`);
    const text = (await card.textContent()) ?? '';
    if (text.startsWith(name)) return;
    await page.getByTestId('discover-pass').click();
    await expect(card.filter({ hasText: text })).toHaveCount(0);
  }
  throw new Error(`${name} not found in the first 40 cards`);
}

test('two users: onboarding, match, realtime chat, safety', async ({ browser }, testInfo) => {
  const shot = (page: Page, name: string) =>
    page.screenshot({ path: testInfo.outputPath(`${name}.png`) });

  const A = await newUserContext(browser);
  const B = await newUserContext(browser);
  const a = A.page;
  const b = B.page;

  await test.step('A registers and completes onboarding', async () => {
    await a.goto('/');
    await expect(a.getByTestId('welcome-screen')).toBeVisible();
    await shot(a, '01-welcome');
    // client-side validation first
    await a.getByTestId('welcome-register').click();
    await a.getByTestId('register-submit').click();
    await expect(a.getByTestId('register-email-error')).toBeVisible();
    await expect(a.getByTestId('register-adult-error')).toBeVisible();
    await register(a, alice);
    await onboard(a, alice, 330, (n) => shot(a, n));
    await shot(a, '02-a-discover-empty');
  });

  await test.step('B registers and completes onboarding', async () => {
    await register(b, bruno);
    await onboard(b, bruno, 200);
  });

  await test.step('A discovers B and likes from the full profile screen', async () => {
    await a.getByTestId('tab-profile').click();
    await a.getByTestId('tab-discover').click();
    await waitForCardOf(a, bruno.name);
    await shot(a, '03-a-card-of-b');
    await a.getByTestId('card-open').click();
    await expect(a.getByTestId('profile-name')).toContainText(bruno.name);
    await expect(a.getByText('Je cherche des conversations vraies')).toBeVisible();
    await shot(a, '04-a-profile-detail');
    await a.getByTestId('detail-like').click();
    await expect(a.getByTestId('discover-screen')).toBeVisible();
  });

  await test.step('B sees A in "Likes", then likes back from Discover -> match modal', async () => {
    await b.getByTestId('tab-likes').click();
    await expect(b.getByText(new RegExp(`${alice.name}, \\d+`)).first()).toBeVisible();
    await shot(b, '05-b-likes');
    await b.getByTestId('tab-discover').click();
    await waitForCardOf(b, alice.name);
    await b.getByTestId('discover-like').click();
    await expect(b.getByTestId('match-title')).toBeVisible();
    await shot(b, '06-b-match');
    await b.getByTestId('match-keep').click();
  });

  await test.step('A is notified of the match in real time and opens the chat', async () => {
    await expect(a.getByTestId('toast-info').filter({ hasText: 'Nouveau match' })).toBeVisible();
    await a.getByTestId('toast-action').click();
    await expect(a.getByTestId('chat-input')).toBeVisible();
  });

  await test.step('A sends a message, B receives it live, reads it, A sees "Lu"', async () => {
    await b.getByTestId('tab-messages').click();
    await expect(b.getByTestId('conversation-list')).toBeVisible();

    await a.getByTestId('chat-input').fill('Salut, ravie de ce match !');
    await a.getByTestId('chat-send').click();
    await expect(bubble(a, 'Salut, ravie de ce match !')).toBeVisible();

    // realtime on B: preview + unread badge without any navigation or reload
    await expect(
      b
        .locator('[data-testid^="conversation-"]')
        .filter({ hasText: 'Salut, ravie de ce match !' })
        .first(),
    ).toBeVisible();
    await expect(b.locator('[data-testid^="conversation-badge-"]').first()).toBeVisible();
    await expect(b.getByTestId('tab-messages')).toContainText('1');
    await shot(b, '07-b-conversation-list-unread');

    await b
      .getByRole('button', { name: new RegExp(`^${alice.name}`) })
      .first()
      .click();
    await expect(bubble(b, 'Salut, ravie de ce match !')).toBeVisible();
    await expect(a.getByText(/· Lu/)).toBeVisible();
    await shot(a, '08-a-chat-read');

    await b.getByTestId('chat-input').fill('Moi aussi, ravi !');
    await b.getByTestId('chat-send').click();
    await expect(bubble(a, 'Moi aussi, ravi !')).toBeVisible();
    await shot(b, '09-b-chat');
  });

  await test.step('A reports then blocks B; B is cut off; A unblocks', async () => {
    await a.getByTestId('chat-menu').click();
    await a.getByTestId('user-menu-report').click();
    await a.getByTestId('report-reason-spam').click();
    await a.getByTestId('report-details').fill('Test e2e');
    await a.getByTestId('report-submit').click();
    await expect(a.getByTestId('toast-success').filter({ hasText: 'signalement' })).toBeVisible();

    await a.getByTestId('chat-menu').click();
    await a.getByTestId('user-menu-block').click();
    await a.getByTestId('confirm-dialog-confirm').click();
    await expect(a.getByTestId('chat-input')).toHaveCount(0); // back to the tabs
    await a.getByTestId('tab-messages').click();
    await expect(a.getByTestId('messages-empty')).toBeVisible();

    // B loses the conversation (realtime match.removed, or at worst on the next refetch)
    await expect(b.getByTestId('messages-empty')).toBeVisible({ timeout: 20_000 });

    await a.getByTestId('tab-profile').click();
    await a.getByTestId('row-blocked').click();
    await expect(
      a.getByTestId(/^blocked-[0-9a-f-]{36}$/).filter({ hasText: bruno.name }),
    ).toBeVisible();
    await shot(a, '10-a-blocked');
    await a.locator('[data-testid^="unblock-"]').first().click();
    await expect(a.getByTestId('blocked-empty')).toBeVisible();
  });

  await test.step('A logs out and logs back in', async () => {
    await a.goto('/'); // full reload: the persisted session must be restored
    await a.getByTestId('tab-profile').click();
    await a.getByTestId('row-logout').click();
    await a.getByTestId('confirm-dialog-confirm').click();
    await expect(a.getByTestId('welcome-screen')).toBeVisible();
    await a.getByTestId('welcome-login').click();
    await a.getByTestId('login-email').fill(alice.email);
    await a.getByTestId('login-password').fill('mauvais-mot-de-passe');
    await a.getByTestId('login-submit').click();
    await expect(a.getByTestId('login-error')).toContainText('incorrect');
    await a.getByTestId('login-password').fill(PASSWORD);
    await a.getByTestId('login-submit').click();
    await expect(a.getByTestId('tab-discover')).toBeVisible();
    // the session survives a full reload (tokens persisted, /me restored)
    await a.reload();
    await expect(a.getByTestId('tab-discover')).toBeVisible();

    // an expired/invalid access token is transparently refreshed (POST /auth/refresh, rotation)
    const before = await a.evaluate(() => JSON.parse(localStorage.getItem('lumo.auth.tokens')!));
    await a.evaluate(() => {
      const t = JSON.parse(localStorage.getItem('lumo.auth.tokens')!);
      t.accessToken = 'expired.invalid.token';
      localStorage.setItem('lumo.auth.tokens', JSON.stringify(t));
    });
    await a.reload();
    await expect(a.getByTestId('tab-discover')).toBeVisible();
    const after = await a.evaluate(() => JSON.parse(localStorage.getItem('lumo.auth.tokens')!));
    expect(after.refreshToken).not.toBe(before.refreshToken);
    expect(after.accessToken).not.toBe('expired.invalid.token');
  });

  await test.step('A edits profile, photos, privacy, notifications and password', async () => {
    await a.getByTestId('tab-profile').click();
    await expect(a.getByTestId('my-name')).toContainText(alice.name);

    await a.getByTestId('row-edit-profile').click();
    await a.getByTestId('edit-bio').fill('Bio modifiée par le test e2e');
    await a.getByTestId('edit-profile-save').click();
    await expect(a.getByTestId('toast-success').filter({ hasText: 'enregistrées' })).toBeVisible();
    await expect(a.getByTestId('profile-screen')).toBeVisible();

    await a.getByTestId('row-preview').click();
    await expect(a.getByText('Bio modifiée par le test e2e')).toBeVisible();
    await shot(a, '11-a-my-preview');
    await a
      .getByRole('link', { name: /Retour/ })
      .first()
      .click();

    await a.getByTestId('row-privacy').click();
    const distance = a.getByTestId('toggle-showDistance').getByRole('switch');
    await distance.click();
    await expect(distance).not.toBeChecked();
    await a.reload();
    await a.getByTestId('tab-profile').click();
    await a.getByTestId('row-privacy').click();
    await expect(a.getByTestId('toggle-showDistance').getByRole('switch')).not.toBeChecked();

    await a.goto('/');
    await a.getByTestId('tab-profile').click();
    await a.getByTestId('row-edit-photos').click();
    const jpeg = await makeJpeg(a, 90);
    const [chooser] = await Promise.all([
      a.waitForEvent('filechooser'),
      a.getByTestId('photo-add').click(),
    ]);
    await chooser.setFiles({ name: 'second.jpg', mimeType: 'image/jpeg', buffer: jpeg });
    await expect(a.getByTestId('photo-tile-1')).toBeVisible();
    await a.getByTestId('photo-tile-1').click();
    await a.getByTestId('photo-menu-main').click();
    await shot(a, '12-a-photos');
    await a.getByTestId('photo-delete-1').click();
    await a.getByTestId('confirm-dialog-confirm').click();
    await expect(a.getByTestId('photo-tile-1')).toHaveCount(0);

    await a.goto('/');
    await a.getByTestId('tab-profile').click();
    await a.getByTestId('row-notifications').click();
    await expect(a.getByTestId('notifications-list')).toBeVisible();
    await shot(a, '13-a-notifications');
    const markAll = a.getByTestId('notifications-mark-all');
    if (await markAll.isVisible()) {
      await markAll.click();
      await expect(markAll).toHaveCount(0);
    }

    await a.goto('/');
    await a.getByTestId('tab-profile').click();
    await a.getByTestId('row-change-password').click();
    await a.getByTestId('cp-current').fill('pas-le-bon-mot-de-passe');
    await a.getByTestId('cp-new').fill('Nouveau-mdp-e2e-1');
    await a.getByTestId('cp-confirm').fill('Nouveau-mdp-e2e-1');
    await a.getByTestId('cp-submit').click();
    await expect(a.getByTestId('cp-current-error')).toBeVisible();
    await a.getByTestId('cp-current').fill(PASSWORD);
    await a.getByTestId('cp-submit').click();
    await expect(
      a.getByTestId('toast-success').filter({ hasText: 'Mot de passe modifié' }),
    ).toBeVisible();
  });

  await test.step('B deletes the account with password confirmation', async () => {
    await b.getByTestId('tab-profile').click();
    await b.getByTestId('row-delete-account').click();
    await b.getByTestId('delete-password').fill('mauvais-mot-de-passe');
    await b.getByTestId('delete-submit').click();
    await b.getByTestId('confirm-dialog-confirm').click();
    await expect(b.getByTestId('delete-password-error')).toBeVisible();
    await b.getByTestId('delete-password').fill(PASSWORD);
    await b.getByTestId('delete-submit').click();
    await b.getByTestId('confirm-dialog-confirm').click();
    await expect(b.getByTestId('welcome-screen')).toBeVisible();
  });

  await A.ctx.close();
  await B.ctx.close();
});
