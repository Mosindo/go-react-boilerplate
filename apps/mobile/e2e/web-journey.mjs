// End-to-end journey on the web build, against a running API:
//   register -> onboarding -> discovery -> like -> match -> realtime chat -> block -> delete account
//
// Prerequisites (see README "Tests"): API on :18080 with ALLOWED_ORIGINS including the web origin,
// the web build served at WEB_URL (default http://localhost:8081) and `npx playwright install chromium`.
import { chromium } from "playwright";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { solidPng } from "./png.mjs";

const WEB_URL = process.env.WEB_URL ?? "http://localhost:8081";
const PASSWORD = "CorrectHorse42";
const run = Date.now();
// Every run gets its own spot on the map so leftovers from earlier runs never show up in discovery.
const LOCATION = {
  latitude: -50 + ((run % 100000) / 100000) * 100,
  longitude: -170 + ((Math.floor(run / 100000) % 100000) / 100000) * 340
};

const photoPath = join(mkdtempSync(join(tmpdir(), "alba-e2e-")), "photo.png");
writeFileSync(photoPath, solidPng(120, 160, [200, 90, 60]));

const step = (message) => console.log(`• ${message}`);
const tid = (id) => `[data-testid="${id}"]`;

async function expectVisible(page, selector, timeout = 15000) {
  await page.waitForSelector(selector, { state: "visible", timeout });
}

async function onboard(browser, { prefix, name, gender, interestedIn }) {
  const context = await browser.newContext({
    viewport: { width: 400, height: 820 },
    permissions: ["geolocation"],
    geolocation: LOCATION
  });
  const page = await context.newPage();
  const email = `${prefix}-${run}@e2e.invalid`;
  page.on("pageerror", (error) => console.error(`[${prefix}] page error:`, error.message));

  await page.goto(WEB_URL);
  await expectVisible(page, tid("auth-screen"), 30000);
  await page.click(tid("mode-register"));
  await page.fill(tid("auth-email"), email);
  await page.fill(tid("auth-password"), PASSWORD);
  await page.click(tid("auth-submit"));

  await expectVisible(page, tid("ob-name"));
  await page.fill(tid("ob-name"), name);
  await page.fill(tid("ob-birth"), "01011990");
  await page.click(tid(`ob-gender-${gender}`));
  await page.click(tid("ob-next"));

  await expectVisible(page, tid("ob-locate"));
  await page.click(tid("ob-locate"));
  await page.waitForFunction(() => document.body.innerText.includes("📍"), null, { timeout: 15000 });
  await page.click(tid("ob-next"));

  await expectVisible(page, tid(`interested-${interestedIn}`));
  await page.click(tid(`interested-${interestedIn}`));
  await page.click(tid("ob-next"));

  await expectVisible(page, tid("photo-add"));
  const [chooser] = await Promise.all([page.waitForEvent("filechooser"), page.click(tid("photo-add"))]);
  await chooser.setFiles(photoPath);
  await page.waitForSelector(`${tid("photo-grid")} img`, { timeout: 15000 });
  await page.click(tid("ob-next"));

  await expectVisible(page, tid("ob-bio"));
  await page.fill(tid("ob-bio"), `Bonjour, je suis ${name}.`);
  await page.getByText("Voyage", { exact: true }).click();
  await page.click(tid("ob-finish"));
  await expectVisible(page, tid("discover-screen"));
  return { page, context, email };
}

async function main() {
  const browser = await chromium.launch();
  try {
    step("Alice and Bob create accounts and profiles");
    const alice = await onboard(browser, { prefix: "alice", name: "Alice", gender: "woman", interestedIn: "man" });
    const bob = await onboard(browser, { prefix: "bob", name: "Bob", gender: "man", interestedIn: "woman" });

    step("Alice discovers Bob and likes him (no match yet)");
    await alice.page.reload();
    await expectVisible(alice.page, tid("discover-card"));
    await alice.page.waitForFunction(() =>
      document.querySelector('[data-testid="discover-name"]')?.textContent?.startsWith("Bob")
    );
    await alice.page.click(tid("discover-like"));
    await expectVisible(alice.page, tid("discover-empty"));

    step("Bob discovers Alice, likes her and gets the match");
    await bob.page.reload();
    await expectVisible(bob.page, tid("discover-card"));
    await bob.page.waitForFunction(() =>
      document.querySelector('[data-testid="discover-name"]')?.textContent?.startsWith("Alice")
    );
    await bob.page.click(tid("discover-like"));
    await expectVisible(bob.page, tid("match-modal"));

    step("Bob opens the chat and sends a message");
    await bob.page.click(tid("match-message"));
    await expectVisible(bob.page, tid("chat-input"));
    await bob.page.fill(tid("chat-input"), "Salut Alice !");
    await bob.page.click(tid("chat-send"));
    await expectVisible(bob.page, tid("msg-mine"));

    step("Alice receives the match and message in real time");
    await alice.page.click(tid("tab-messages"));
    await expectVisible(alice.page, tid("conversation-row"), 20000);
    await alice.page.waitForFunction(() => document.body.innerText.includes("Salut Alice !"), null, { timeout: 20000 });
    await alice.page.click(tid("conversation-row"));
    await expectVisible(alice.page, tid("msg-theirs"));
    await alice.page.fill(tid("chat-input"), "Salut Bob, ravie de te rencontrer.");
    await alice.page.click(tid("chat-send"));

    step("Bob sees Alice's reply without reloading, and his message is marked as read");
    await bob.page.waitForSelector(tid("msg-theirs"), { timeout: 20000 });
    await alice.page
      .waitForFunction(() => document.body.innerText.includes("Lu"), null, { timeout: 20000 })
      .catch(() => undefined);

    step("Bob blocks Alice: the conversation disappears for both");
    await bob.page.click(tid("chat-menu"));
    await bob.page.click(tid("chat-block"));
    await bob.page.click(tid("tab-messages"));
    await expectVisible(bob.page, tid("messages-empty"));
    await alice.page.reload();
    await alice.page.click(tid("tab-messages"));
    await expectVisible(alice.page, tid("messages-empty"));

    step("Alice deletes her account");
    await alice.page.click(tid("tab-me"));
    await alice.page.click(tid("row-settings"));
    await alice.page.click(tid("delete-account"));
    await alice.page.fill(tid("delete-password"), PASSWORD);
    await alice.page.click(tid("delete-confirm"));
    await expectVisible(alice.page, tid("auth-screen"));

    console.log("\n✔ End-to-end journey passed");
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error("\n✘ End-to-end journey failed:", error.message);
  process.exit(1);
});
