import { deflateSync } from "node:zlib";
import { expect, type Browser, type BrowserContext, type Page } from "@playwright/test";

export type Person = {
  email: string;
  password: string;
  name: string;
  gender: "man" | "woman";
  seeking: "man" | "woman";
  birth: { day: string; month: string; year: string };
};

export function person(label: string, gender: "man" | "woman", seeking: "man" | "woman"): Person {
  const unique = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;
  return {
    email: `${label.toLowerCase()}.${unique}@e2e.test`,
    password: "correct-horse-1",
    name: label,
    gender,
    seeking,
    birth: { day: "14", month: "3", year: "1993" }
  };
}

function crc32(buf: Buffer): number {
  let c = ~0;
  for (const byte of buf) {
    c ^= byte;
    for (let k = 0; k < 8; k++) c = (c >>> 1) ^ (0xedb88320 & -(c & 1));
  }
  return ~c >>> 0;
}

function chunk(type: string, data: Buffer): Buffer {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}

/** A real PNG (gradient) built without dependencies, so uploads exercise the image pipeline. */
export function makePng(width = 600, height = 800, hue = 0): Buffer {
  const rows: Buffer[] = [];
  for (let y = 0; y < height; y++) {
    const row = Buffer.alloc(1 + width * 3);
    for (let x = 0; x < width; x++) {
      row[1 + x * 3] = (x * 255) / width;
      row[2 + x * 3] = (y * 255) / height;
      row[3 + x * 3] = (hue + 90) % 256;
    }
    rows.push(row);
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(Buffer.concat(rows))),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

/** Surfaces browser-side crashes in the test output, so a blank screen is never a mystery. */
export function watch(page: Page, label = "page"): void {
  page.on("pageerror", (error) => console.log(`[${label}] pageerror: ${error.message}`));
  page.on("console", (message) => {
    if (message.type() === "error") console.log(`[${label}] console.error: ${message.text().slice(0, 400)}`);
  });
}

export async function newPlayer(browser: Browser, label = "player"): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext({ viewport: { width: 420, height: 880 }, locale: "fr-FR" });
  const page = await context.newPage();
  watch(page, label);
  return { context, page };
}

export async function register(page: Page, who: Person): Promise<void> {
  await page.goto("/");
  await page.getByTestId("tab-register").click();
  await page.getByTestId("auth-email").fill(who.email);
  await page.getByTestId("auth-password").fill(who.password);
  await page.getByTestId("auth-adult").click();
  await page.getByTestId("auth-submit").click();
  await expect(page.getByTestId("onboarding-step-0")).toBeVisible();
}

export async function completeOnboarding(page: Page, who: Person, hue: number): Promise<void> {
  // Step 1: identity
  await page.getByTestId("ob-name").fill(who.name);
  await page.getByTestId("birth-day").fill(who.birth.day);
  await page.getByTestId("birth-month").fill(who.birth.month);
  await page.getByTestId("birth-year").fill(who.birth.year);
  await page.getByTestId(`gender-${who.gender}`).click();
  await page.getByTestId("onboarding-next").click();

  // Step 2: preferences
  await expect(page.getByTestId("onboarding-step-1")).toBeVisible();
  await page.getByTestId(`seek-${who.seeking}`).click();
  await page.getByTestId("onboarding-next").click();

  // Step 3: location (approximate area, no browser permission needed)
  await expect(page.getByTestId("onboarding-step-2")).toBeVisible();
  await page.getByTestId("city-Lyon").click();
  await page.getByTestId("onboarding-next").click();

  // Step 4: bio and interests
  await expect(page.getByTestId("onboarding-step-3")).toBeVisible();
  await page.getByTestId("ob-bio").fill(`Je m'appelle ${who.name} et j'aime les longues balades.`);
  await page.getByTestId("interest-travel").click();
  await page.getByTestId("interest-music").click();
  await page.getByTestId("onboarding-next").click();

  // Step 5: photo (real file chooser, real upload)
  await expect(page.getByTestId("onboarding-step-4")).toBeVisible();
  await expect(page.getByTestId("onboarding-finish")).toBeDisabled();
  const [chooser] = await Promise.all([page.waitForEvent("filechooser"), page.getByTestId("photo-add").click()]);
  await chooser.setFiles({ name: "photo.png", mimeType: "image/png", buffer: makePng(600, 800, hue) });
  await expect(page.getByTestId("photo-slot-0")).toBeVisible();
  await expect(page.getByTestId("onboarding-finish")).toBeEnabled();
  await page.getByTestId("onboarding-finish").click();
  await expect(page.getByTestId("home-screen")).toBeVisible();
}
