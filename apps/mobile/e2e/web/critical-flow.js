// UI end-to-end test of the critical journey on the web build, in Chromium:
// register -> onboarding (real photo upload) -> discovery -> like -> match
// -> realtime chat -> notifications -> block -> account deletion.
//
// Prerequisites: a running API (E2E_API_URL, CORS allowing the web origin) and
// a web export built with the same EXPO_PUBLIC_API_URL:
//   EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
//   E2E_API_URL=http://localhost:18080 npm run e2e:web
const fs = require("fs");
const os = require("os");
const path = require("path");
const { chromium } = require("playwright");
const { createMember, expectStatus, makePng, request, uniqueEmail } = require("../fixtures");
const { startStaticServer } = require("./server");

const WEB_PORT = Number(process.env.E2E_WEB_PORT || 8081);
const WEB_URL = `http://localhost:${WEB_PORT}`;
const REPORT_DIR = path.join(__dirname, "..", "reports");
const PASSWORD = "Password123";

async function main() {
  fs.mkdirSync(REPORT_DIR, { recursive: true });
  const server = await startStaticServer(path.join(__dirname, "..", "..", "dist"), WEB_PORT);
  const browser = await chromium.launch();
  const context = await browser.newContext({ viewport: { width: 420, height: 860 }, locale: "fr-FR" });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  const tid = (id) => page.getByTestId(id);
  const shot = (name) => page.screenshot({ path: path.join(REPORT_DIR, `${name}.png`) });
  const token = Math.random().toString(36).slice(2, 8);
  const email = uniqueEmail("web_camille");
  let theo = null;

  const step = async (name, fn) => {
    process.stdout.write(`- ${name} ... `);
    await fn();
    console.log("ok");
  };

  try {
    await page.goto(WEB_URL);

    await step("register", async () => {
      await tid("welcome-register").click();
      await tid("register-email").fill(email);
      await tid("register-password").fill(PASSWORD);
      await tid("register-adult").click();
      await tid("register-submit").click();
      await tid("onboarding-screen").waitFor();
    });

    await step("onboarding: basics", async () => {
      await tid("onboarding-first-name").fill("Camille");
      await tid("birth-day").fill("12");
      await tid("birth-month").fill("03");
      await tid("birth-year").fill("1996");
      await tid("gender-woman").click();
      await shot("01-onboarding-basics");
      await tid("onboarding-next").click();
      await page.getByText("Qui aimeriez-vous rencontrer ?").waitFor();
    });

    await step("onboarding: preferences", async () => {
      await tid("interested-man").click();
      await tid("onboarding-next").click();
      await page.getByText("Montrez-vous").waitFor();
    });

    await step("onboarding: photo upload", async () => {
      const file = path.join(os.tmpdir(), `lueur-e2e-${token}.png`);
      fs.writeFileSync(file, makePng(480, 600, 40));
      const chooser = page.waitForEvent("filechooser");
      await tid("photo-add").click();
      await (await chooser).setFiles(file);
      await tid("photo-slot-0").waitFor({ timeout: 20_000 });
      await shot("02-onboarding-photos");
      await tid("onboarding-next").click();
    });

    await step("onboarding: about and interests", async () => {
      await tid("onboarding-bio").fill(`Toujours partante pour un café. ${token}`);
      await tid("interest-hiking").click();
      await tid("interest-coffee").click();
      await tid("onboarding-next").click();
      await tid("onboarding-skip-location").click();
      await tid("discover-screen").waitFor();
    });

    await step("another member likes Camille (API)", async () => {
      theo = await createMember({ prefix: "web_theo", firstName: "Théo", gender: "man", interestedIn: ["woman"], withLocation: false });
      const feed = expectStatus(await request("/discovery?limit=20", { token: theo.token }), 200, "discovery");
      const camille = feed.profiles.find((p) => p.bio.includes(token));
      if (!camille) throw new Error("Camille not discoverable by Théo");
      theo.camilleId = camille.userId;
      expectStatus(
        await request("/swipes", { method: "POST", token: theo.token, body: { targetUserId: camille.userId, decision: "like" } }),
        200,
        "theo likes camille"
      );
    });

    await step("discover and like -> match", async () => {
      await page.reload();
      await tid("swipe-card").first().waitFor();
      // Members who already liked you rank first.
      for (let i = 0; i < 20; i++) {
        const name = await tid("swipe-card-open").last().innerText();
        if (name.includes("Théo")) break;
        await tid("discover-pass").click();
        await page.waitForTimeout(400);
      }
      await shot("03-discover");
      await tid("discover-like").click();
      await tid("match-celebration").waitFor();
      await page.waitForTimeout(700);
      await shot("04-match");
      await tid("match-message").click();
      await tid("chat-screen").waitFor();
    });

    await step("chat: send, realtime reply and read receipt", async () => {
      await tid("chat-input").fill("Bonjour Théo ! Une rando ce week-end ?");
      await tid("chat-send").click();
      await page.getByText("Bonjour Théo ! Une rando ce week-end ?").waitFor();
      expectStatus(
        await request(`/chats/${theo.camilleId}/messages`, {
          method: "POST",
          token: theo.token,
          body: { content: "Avec plaisir, samedi ?" }
        }),
        201,
        "theo replies"
      );
      await page.getByText("Avec plaisir, samedi ?").waitFor({ timeout: 8_000 });
      await request(`/chats/${theo.camilleId}/read`, { method: "POST", token: theo.token });
      await shot("05-chat");
    });

    await step("matches and notifications", async () => {
      await page.getByLabel("Retour").first().click();
      await tid("tab-matches").click();
      await tid(`chat-row-${theo.userId}`).waitFor();
      await tid("tab-activity").click();
      await page.getByText("Nouveau match").first().waitFor();
      await shot("06-activity");
      const readAll = tid("activity-read-all");
      if (await readAll.isVisible()) {
        await readAll.click();
      }
    });

    await step("block from the conversation", async () => {
      await tid("tab-matches").click();
      await tid(`chat-row-${theo.userId}`).click();
      await tid("chat-safety").click();
      await tid("safety-block").click();
      await tid("confirm-accept").click();
      await tid("matches-screen").waitFor();
      await page.waitForTimeout(500);
      if (await tid(`chat-row-${theo.userId}`).count()) throw new Error("blocked conversation still listed");
      expectStatus(await request(`/chats/${theo.camilleId}/messages`, { token: theo.token }), 404, "blocked chat for theo");
    });

    await step("profile tab and account deletion", async () => {
      await tid("tab-me").click();
      await tid("me-screen").waitFor();
      await page.waitForTimeout(800);
      await shot("07-me");
      await tid("me-account").click();
      await tid("delete-password").fill(PASSWORD);
      await tid("delete-account").click();
      await tid("confirm-accept").click();
      await tid("welcome-screen").waitFor();
      const login = await request("/auth/login", { method: "POST", body: { email, password: PASSWORD } });
      if (login.status !== 401) throw new Error(`deleted account can still log in (${login.status})`);
    });

    await step("dark mode renders", async () => {
      const dark = await browser.newPage({ colorScheme: "dark", viewport: { width: 420, height: 860 } });
      await dark.goto(WEB_URL);
      await dark.getByTestId("welcome-screen").waitFor();
      await dark.screenshot({ path: path.join(REPORT_DIR, "08-welcome-dark.png") });
      await dark.close();
    });

    if (errors.length) {
      throw new Error(`page errors: ${errors.join(" | ")}`);
    }
    console.log(`[e2e-web] PASS (screenshots in ${REPORT_DIR})`);
  } catch (error) {
    await shot("failure").catch(() => undefined);
    console.error(`\n[e2e-web] FAIL: ${error.message}`);
    process.exitCode = 1;
  } finally {
    if (theo) {
      await request("/me", { method: "DELETE", token: theo.token, body: { password: PASSWORD } });
    }
    await browser.close();
    server.close();
  }
}

main();
