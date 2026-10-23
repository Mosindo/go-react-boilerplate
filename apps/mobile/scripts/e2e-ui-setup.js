/**
 * Seeds the live API for the Maestro UI flow: two onboarded users who matched, with one message
 * from the contact. Writes .e2e/maestro-env.json and .e2e/maestro-env.ps1.
 */
const fs = require("fs");
const path = require("path");
const {
  API_URL,
  PASSWORD,
  assert,
  call,
  expectStatus,
  onboardUser,
  preferenceInput,
  profileInput,
  registerUser
} = require("./lib/api");

const CONTACT_NAME = "Camille";

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_URL}`);
  const primary = await registerUser("ui_primary");
  const contact = await registerUser("ui_contact");
  await onboardUser(
    primary,
    profileInput("Alex", "man", "1992-02-02"),
    preferenceInput(["woman"]),
    3
  );
  await onboardUser(
    contact,
    profileInput(CONTACT_NAME, "woman", "1994-07-21"),
    preferenceInput(["man"]),
    4
  );

  for (const [from, to] of [
    [primary, contact],
    [contact, primary]
  ]) {
    const swipe = expectStatus(
      await call("/swipes", {
        method: "POST",
        token: from.token,
        body: { targetUserId: to.id, action: "like" }
      }),
      200,
      "seed like"
    );
    if (from === contact) assert(swipe.matched && swipe.match?.id, "seed users should match");
    if (from === contact) {
      expectStatus(
        await call(`/matches/${swipe.match.id}/messages`, {
          method: "POST",
          token: contact.token,
          body: { content: "Salut, ravie de ce match !" }
        }),
        201,
        "seed message"
      );
    }
  }

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const env = {
    TEST_EMAIL: primary.email,
    TEST_PASSWORD: PASSWORD,
    CONTACT_NAME,
    API_URL
  };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(env, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${env.TEST_EMAIL}'\n$env:TEST_PASSWORD='${env.TEST_PASSWORD}'\n$env:CONTACT_NAME='${CONTACT_NAME}'\n`
  );
  console.log("[e2e-ui-setup] PASS");
  console.log(`[e2e-ui-setup] TEST_EMAIL=${primary.email}`);
  console.log(`[e2e-ui-setup] CONTACT_NAME=${CONTACT_NAME}`);
}

main().catch((err) => {
  console.error(`[e2e-ui-setup] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
