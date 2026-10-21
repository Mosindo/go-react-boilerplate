import { expect, test, type Page } from "@playwright/test";
import { deflateSync } from "node:zlib";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const API = process.env.E2E_API_URL ?? "http://localhost:18080";
const PASSWORD = "Sup3rSecret!";

// ---- helpers -----------------------------------------------------------------------------------

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

/** A 400x500 gradient PNG, generated on the fly (no binary fixture in the repo). */
function makePng(path: string): string {
  const w = 400;
  const h = 500;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) {
    raw[y * (w * 3 + 1)] = 0;
    for (let x = 0; x < w; x++) {
      const i = y * (w * 3 + 1) + 1 + x * 3;
      raw[i] = 200;
      raw[i + 1] = x % 256;
      raw[i + 2] = y % 256;
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  const png = Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
  writeFileSync(path, png);
  return path;
}

type Json = Record<string, unknown>;

async function api(
  path: string,
  opts: { method?: string; token?: string; body?: unknown; form?: FormData } = {}
): Promise<{ status: number; json: Json }> {
  const res = await fetch(`${API}${path}`, {
    method: opts.method ?? "GET",
    headers: {
      ...(opts.token ? { Authorization: `Bearer ${opts.token}` } : {}),
      ...(opts.body ? { "Content-Type": "application/json" } : {})
    },
    body: opts.form ?? (opts.body ? JSON.stringify(opts.body) : undefined)
  });
  const text = await res.text();
  return { status: res.status, json: text ? (JSON.parse(text) as Json) : {} };
}

/** Creates a complete member through the API (the counterpart in the scenario). */
async function createMember(name: string, gender: string, photo: string) {
  const email = `e2e-${name.toLowerCase()}-${Date.now()}@test.alma.invalid`;
  const reg = await api("/auth/register", { method: "POST", body: { email, password: PASSWORD, birthDate: "1992-04-12" } });
  expect(reg.status).toBe(201);
  const token = reg.json.accessToken as string;
  const id = (reg.json.user as Json).id as string;
  expect(
    (
      await api("/profile", {
        method: "PUT",
        token,
        body: { firstName: name, gender, bio: "Membre de test", city: "Lyon", interests: ["music"] }
      })
    ).status
  ).toBe(200);
  const form = new FormData();
  form.append("photo", new Blob([await (await import("node:fs/promises")).readFile(photo)], { type: "image/png" }), "p.png");
  expect((await api("/photos", { method: "POST", token, form })).status).toBe(201);
  return { id, token, email };
}

/** In-app back navigation (the web build has no URL routing, so the browser back button is not used). */
async function back(page: Page) {
  await page
    .getByRole("button", { name: /retour|back/i })
    .first()
    .click();
}

async function tokensOf(page: Page): Promise<{ accessToken: string; refreshToken: string }> {
  const raw = await page.evaluate(() => localStorage.getItem("alma.auth.tokens"));
  return JSON.parse(raw ?? "{}");
}

// ---- scenario ----------------------------------------------------------------------------------

test("sign up → profile → photo → discover → like → match → chat (realtime) → block → delete account", async ({ page }) => {
  page.on("dialog", (dialog) => void dialog.accept());
  const dir = mkdtempSync(join(tmpdir(), "alma-e2e-"));
  const photo = makePng(join(dir, "photo.png"));
  const suffix = String(Date.now()).slice(-6);
  const myEmail = `e2e-alice-${suffix}@test.alma.invalid`;
  const bastien = await createMember(`Bastien${suffix}`, "man", photo);
  const bastienName = `Bastien${suffix}`;

  // 1. Registration (and client-side age gate).
  await page.goto("/");
  await expect(page.getByTestId("auth-screen")).toBeVisible();
  await page.getByTestId("auth-mode-register").click();
  await page.getByTestId("auth-email-input").fill(myEmail);
  await page.getByTestId("auth-password-input").fill(PASSWORD);
  await page.getByTestId("auth-birth-day").fill("10");
  await page.getByTestId("auth-birth-month").fill("5");
  await page.getByTestId("auth-birth-year").fill(String(new Date().getFullYear() - 16));
  await page.getByTestId("auth-submit-button").click();
  await expect(page.getByText("Vous devez avoir 18 ans ou plus").first()).toBeVisible();
  await page.getByTestId("auth-birth-year").fill("1993");
  await page.getByTestId("auth-submit-button").click();

  // 2. Onboarding: about → tastes → preferences → photos.
  await expect(page.getByTestId("onboarding-title")).toHaveText("À propos de vous");
  await page.getByTestId("onboarding-next").click();
  await expect(page.getByText("Indiquez votre prénom.")).toBeVisible(); // validation
  await page.getByTestId("profile-first-name").fill("Alice");
  await page.getByTestId("gender-woman").click();
  await page.getByTestId("profile-bio").fill("Curieuse, sincère, gourmande.");
  await page.getByTestId("profile-city").fill("Lyon");
  await page.getByTestId("onboarding-next").click();

  await expect(page.getByTestId("onboarding-title")).toHaveText("Vos goûts");
  await page.getByTestId("interest-music").click();
  await page.getByTestId("interest-travel").click();
  await page.getByTestId("onboarding-next").click();

  await expect(page.getByTestId("onboarding-title")).toHaveText("Vos préférences");
  await page.getByTestId("interested-man").click(); // toggle off men then back on keeps at least one
  await page.getByTestId("interested-man").click();
  await page.getByTestId("onboarding-next").click();

  await expect(page.getByTestId("onboarding-title")).toHaveText("Vos photos");
  await expect(page.getByTestId("onboarding-finish")).toBeDisabled();
  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("photo-add").click();
  await (await chooser).setFiles(photo);
  await expect(page.getByTestId("photo-slot-0")).toBeVisible();
  await page.getByTestId("onboarding-finish").click();

  // 3. Bastien already likes Alice → she sees him first and likes him back.
  const me = await tokensOf(page);
  const meJson = await api("/me", { token: me.accessToken });
  expect(meJson.json.hasProfile).toBe(true);
  expect((await api("/swipes", { method: "POST", token: bastien.token, body: { targetId: meJson.json.id, action: "like" } })).status).toBe(
    200
  );

  await page.reload();
  await expect(page.getByTestId("discover-screen")).toBeVisible();
  await expect(page.getByTestId("swipe-card-name")).toContainText(bastienName);
  await page.getByTestId("swipe-like").click();

  // 4. Match modal → chat.
  await expect(page.getByTestId("match-screen")).toBeVisible();
  await page.getByTestId("match-send-message").click();
  await page.getByTestId("chat-input").fill("Bonjour Bastien !");
  await page.getByTestId("chat-send").click();
  await expect(page.getByText("Bonjour Bastien !")).toBeVisible();

  // 5. Bastien reads it (server side) and answers → Alice receives it live, without reloading.
  const convs = await api("/conversations", { token: bastien.token });
  const conv = (convs.json.items as Json[])[0] as Json;
  expect((conv.lastMessage as Json).body).toBe("Bonjour Bastien !");
  expect(conv.unreadCount).toBe(1);
  expect(
    (
      await api(`/conversations/${conv.id}/messages`, {
        method: "POST",
        token: bastien.token,
        body: { body: "Salut Alice, ravi de te rencontrer" }
      })
    ).status
  ).toBe(201);
  await expect(page.getByText("Salut Alice, ravi de te rencontrer")).toBeVisible();

  // 6. Notifications list the match.
  await back(page);
  await page.getByTestId("tab-messages").click();
  await expect(page.getByTestId(`conversation-${bastienName}`)).toBeVisible();
  await page.getByTestId("messages-screen").getByTestId("open-notifications").click();
  await expect(page.getByTestId("notification-match")).toBeVisible();
  await back(page);

  // 7. Block from the chat: the conversation disappears.
  await page.getByTestId(`conversation-${bastienName}`).click();
  await page.getByTestId("chat-menu").click();
  await page.getByTestId("chat-block").click();
  await expect(page.getByTestId("messages-empty")).toBeVisible();
  expect((await api(`/conversations/${conv.id}/messages`, { token: bastien.token })).status).toBe(404);

  // 8. Account deletion.
  await page.getByTestId("tab-profile").click();
  await page.getByTestId("menu-delete").click();
  await page.getByTestId("delete-password").fill(PASSWORD);
  await page.getByTestId("delete-submit").click();
  await expect(page.getByTestId("auth-screen")).toBeVisible();
  const login = await api("/auth/login", { method: "POST", body: { email: myEmail, password: PASSWORD } });
  expect(login.status).toBe(401);
});
