import { expect, test, type Browser, type Page } from "@playwright/test";
import { deflateSync } from "node:zlib";

const CRC_TABLE = Array.from({ length: 256 }, (_, n) => {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});

function crc32(buf: Buffer): number {
  let c = 0xffffffff;
  for (const byte of buf) c = (CRC_TABLE[(c ^ byte) & 0xff] as number) ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

/** Builds a small valid PNG in memory so the test needs no binary fixture. */
function makePng(width: number, height: number): Buffer {
  const chunk = (type: string, data: Buffer) => {
    const body = Buffer.concat([Buffer.from(type), data]);
    const out = Buffer.alloc(12 + data.length);
    out.writeUInt32BE(data.length, 0);
    body.copy(out, 4);
    out.writeUInt32BE(crc32(body), 8 + data.length);
    return out;
  };
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const rows: Buffer[] = [];
  for (let y = 0; y < height; y++) {
    const row = Buffer.alloc(1 + width * 3);
    for (let x = 0; x < width; x++) row.set([x % 255, y % 255, 150], 1 + x * 3);
    rows.push(row);
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(Buffer.concat(rows))),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

// A private corner of the map per run: nobody else in the database can appear in discovery.
const run = Date.now();
const geo = { latitude: -40 + (run % 1000) / 100, longitude: -100 + ((run >> 3) % 2000) / 100 };

async function newUserPage(browser: Browser): Promise<Page> {
  const context = await browser.newContext({
    viewport: { width: 420, height: 860 },
    permissions: ["geolocation"],
    geolocation: geo,
  });
  return context.newPage();
}

async function signUpAndOnboard(
  page: Page,
  opts: { email: string; name: string; gender: "woman" | "man"; seeks: "Femmes" | "Hommes" },
) {
  await page.goto("/");
  await page.getByTestId("welcome-register").click();
  await page.getByTestId("register-email").fill(opts.email);
  await page.getByTestId("register-password").fill("Password123");
  await page.getByTestId("register-submit").click();

  // Step 1: profile
  await page.getByTestId("profile-firstname").fill(opts.name);
  await page.getByTestId("profile-birth").fill("15061995");
  await page.getByTestId(`gender-${opts.gender}`).click();
  await page.getByTestId("profile-locate").click();
  await expect(page.getByRole("button", { name: "Mettre à jour ma position" })).toBeVisible();
  await page.getByTestId("profile-submit").click();

  // Step 2: preferences (default chips include everybody: narrow to the intended target)
  await expect(page.getByText("Vos préférences")).toBeVisible();
  for (const label of ["Femmes", "Hommes", "Non-binaires"]) {
    if (label !== opts.seeks) await page.getByRole("button", { name: label }).click();
  }
  await page.getByTestId("prefs-submit").click();

  // Step 3: photos
  await expect(page.getByText("Vos plus belles photos")).toBeVisible();
  const [chooser] = await Promise.all([
    page.waitForEvent("filechooser"),
    page.getByTestId("photo-add").click(),
  ]);
  await chooser.setFiles({ name: "me.png", mimeType: "image/png", buffer: makePng(480, 600) });
  await expect(page.getByRole("button", { name: "Photo 1, options" })).toBeVisible({
    timeout: 20_000,
  });
  await page.getByTestId("photos-done").click();
  await expect(page.getByRole("link", { name: /Messages/ })).toBeVisible({ timeout: 20_000 });
}

test("signup → profile → discover → like → match → live chat", async ({ browser }) => {
  const alice = await newUserPage(browser);
  const bob = await newUserPage(browser);
  const tag = String(run);

  await signUpAndOnboard(alice, {
    email: `alice-${tag}@e2e.test`,
    name: "Alice",
    gender: "woman",
    seeks: "Hommes",
  });
  await signUpAndOnboard(bob, {
    email: `bob-${tag}@e2e.test`,
    name: "Bob",
    gender: "man",
    seeks: "Femmes",
  });

  // Alice was alone when she arrived; she refreshes and now sees Bob. She likes him: no match yet.
  await alice.getByRole("button", { name: "Actualiser" }).click();
  await expect(alice.getByText("Bob", { exact: true }).first()).toBeVisible({ timeout: 20_000 });
  await alice.getByTestId("swipe-like").click();
  await expect(alice.getByText("C'est un match !")).toHaveCount(0);

  // Bob sees Alice, likes back: the match modal appears for him...
  await bob.reload();
  await expect(bob.getByText("Alice", { exact: true }).first()).toBeVisible({ timeout: 20_000 });
  await bob.getByTestId("swipe-like").click();
  await expect(bob.getByText("C'est un match !")).toBeVisible();

  // ...and he writes first.
  await bob.getByTestId("match-chat").click();
  await bob.getByTestId("chat-input").fill("Salut Alice, ravi du match !");
  await bob.getByTestId("chat-send").click();
  await expect(
    bob.getByTestId("chat-list").getByText("Salut Alice, ravi du match !"),
  ).toBeVisible();

  // Alice receives the match and the message in real time, without reloading.
  await alice.getByRole("link", { name: /Messages/ }).click();
  await alice.getByRole("button", { name: /Conversation avec Bob/ }).click({ timeout: 20_000 });
  await expect(
    alice.getByTestId("chat-list").getByText("Salut Alice, ravi du match !"),
  ).toBeVisible({ timeout: 20_000 });

  await alice.getByTestId("chat-input").fill("Moi aussi 😊");
  await alice.getByTestId("chat-send").click();
  await expect(bob.getByTestId("chat-list").getByText("Moi aussi 😊")).toBeVisible({
    timeout: 15_000,
  });
  // Bob has the chat open: Alice's message is marked read, shown as "Lu" on Bob's last message.
  await expect(bob.getByTestId("chat-list").getByText(/Lu$/)).toBeVisible({ timeout: 15_000 });
});

test("account deletion erases the account from the settings screen", async ({ browser }) => {
  const page = await newUserPage(browser);
  const email = `gone-${run}@e2e.test`;
  await signUpAndOnboard(page, { email, name: "Gone", gender: "woman", seeks: "Hommes" });

  await page.getByRole("link", { name: /Profil/ }).click();
  await page.getByTestId("me-settings").click();
  await page.getByRole("button", { name: "Supprimer mon compte" }).click();
  await page.getByRole("textbox", { name: "Mot de passe" }).fill("Password123");
  await page.getByTestId("delete-submit").click();
  await page.getByRole("button", { name: "Supprimer mon compte" }).last().click(); // confirmation sheet
  await expect(page.getByTestId("welcome-register")).toBeVisible({ timeout: 15_000 });

  await page.getByTestId("welcome-login").click();
  await page.getByTestId("login-email").fill(email);
  await page.getByTestId("login-password").fill("Password123");
  await page.getByTestId("login-submit").click();
  await expect(page.getByText("E-mail ou mot de passe incorrect.")).toBeVisible();
});
