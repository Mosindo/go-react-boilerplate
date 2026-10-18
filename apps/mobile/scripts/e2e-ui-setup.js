/**
 * Prepares fixtures for the Maestro device flow: one fully onboarded user who
 * already matched with a contact that wrote to them. Writes env files that
 * run-maestro.ps1 loads.
 */
const fs = require("node:fs");
const path = require("node:path");
const { API_BASE_URL, PASSWORD, createMember, expectStatus, request } = require("./lib/dating-api");

const NOTIFICATION_TITLE = "C'est un match !";

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);
  const primary = await createMember("ui_primary", { name: "Claire", gender: "woman", seeking: "man" });
  const contact = await createMember("ui_contact", { name: "Marc", gender: "man", seeking: "woman" });

  await expectStatus(
    "like",
    200,
    request("/swipes", { method: "POST", token: primary.token, body: { userId: contact.id, action: "like" } })
  );
  const match = await expectStatus(
    "match",
    200,
    request("/swipes", { method: "POST", token: contact.token, body: { userId: primary.id, action: "like" } })
  );
  await expectStatus(
    "message",
    201,
    request(`/conversations/${match.match.conversationId}/messages`, {
      method: "POST",
      token: contact.token,
      body: { content: "Bonjour, ravi de ce match !" }
    })
  );

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const envData = {
    TEST_EMAIL: primary.email,
    TEST_PASSWORD: PASSWORD,
    CONTACT_EMAIL: contact.email,
    NOTIFICATION_TITLE,
    API_BASE_URL
  };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(envData, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${primary.email}'\n$env:TEST_PASSWORD='${PASSWORD}'\n$env:CONTACT_EMAIL='${contact.email}'\n$env:NOTIFICATION_TITLE='${NOTIFICATION_TITLE}'\n`
  );
  console.log(`[e2e-ui-setup] PASS TEST_EMAIL=${primary.email}`);
}

main().catch((error) => {
  console.error(`[e2e-ui-setup] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
