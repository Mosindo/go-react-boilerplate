// Captures the main screens for visual review: node e2e/web/screenshots.mjs <outDir> [dark]
import { chromium, devices } from "@playwright/test";
import path from "path";
import { execSync } from "child_process";

const out = process.argv[2];
const scheme = process.argv[3] === "dark" ? "dark" : "light";
const WEB = process.env.E2E_WEB_URL ?? "http://localhost:8099";
const photo = path.resolve("e2e/web/fixtures/portrait.png");

const browser = await chromium.launch();
const page = await browser.newPage({ ...devices["Pixel 7"], colorScheme: scheme });
const shot = (name) => page.screenshot({ path: `${out}/${scheme}-${name}.png` });
const email = `shots_${Date.now()}@e2e.test`;

await page.goto(WEB);
await page.getByTestId("welcome-signup").waitFor();
await shot("01-welcome");
await page.getByTestId("welcome-signup").click();
await page.getByTestId("signup-email").fill(email);
await page.getByTestId("signup-password").fill("Password123");
await page.getByTestId("signup-adult").click();
await shot("02-signup");
await page.getByTestId("signup-submit").click();
await page.getByTestId("onboarding-firstname").fill("Jordan");
await page.getByTestId("onboarding-next").click();
await page.getByTestId("onboarding-birthdate").fill("02021994");
await page.getByTestId("onboarding-next").click();
await page.getByTestId("gender-man").click();
await page.getByTestId("onboarding-next").click();
await page.getByTestId("seeking-woman").click();
await shot("03-onboarding-seeking");
await page.getByTestId("onboarding-next").click();
const chooser = page.waitForEvent("filechooser");
await page.getByTestId("photo-add").click();
await (await chooser).setFiles(photo);
await page.getByTestId("photo-slot-0").waitFor();
await page.waitForTimeout(500);
await shot("04-onboarding-photos");
await page.getByTestId("onboarding-next").click();
await page.getByTestId("onboarding-next").click();
await page.getByTestId("interest-cooking").click();
await page.getByTestId("interest-travel").click();
await shot("05-onboarding-about");
await page.getByTestId("onboarding-next").click();
await page.getByTestId("profile-card").first().waitFor();
await page.waitForTimeout(800);
await shot("06-discover");
await page.getByTestId("profile-card-details").last().click();
await page.getByTestId("profile-detail").waitFor();
await page.waitForTimeout(600);
await shot("07-profile-detail");
await page.getByTestId("detail-like").click();
await page.getByTestId("discover-screen").waitFor();

// Demo members like us (seed -like-email), so our next like is a match.
if (process.env.SEED_BIN) {
  execSync(`${process.env.SEED_BIN} -like-email ${email}`, { stdio: "ignore" });
}
await page.getByTestId("discover-like").click();
await page.getByTestId("match-modal").waitFor();
await page.waitForTimeout(700);
await shot("08-match");
await page.getByTestId("match-message").click();
await page.getByTestId("chat-input").fill("Hello ! Ta photo de rando m'a donné envie de partir en week-end 🙂");
await page.getByTestId("chat-send").click();
await page.getByText("Hello !", { exact: false }).waitFor();
await page.getByTestId("chat-input").fill("Tu connais de bons coins près de Paris ?");
await page.getByTestId("chat-send").click();
await page.waitForTimeout(600);
await shot("09-chat");
await page.getByRole("button", { name: /retour|back/i }).first().click();
await page.getByTestId("tab-messages").click();
await page.waitForTimeout(700);
await shot("10-messages");
await page.getByTestId("tab-notifications").click();
await page.waitForTimeout(700);
await shot("11-notifications");
await page.getByTestId("tab-me").click();
await page.waitForTimeout(700);
await shot("12-me");
await page.getByTestId("me-settings").click();
await page.waitForTimeout(700);
await shot("13-settings");
await browser.close();
