/* eslint-disable no-console */
// Creates the fixtures used by the Maestro UI flow: a fully onboarded member who
// is already matched with a second member who sent a first message.
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, createMember, expectStatus, randomSpot, request } = require("./lib/api");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);
  const spot = randomSpot();
  const tester = await createMember({ name: "Testeuse", gender: "woman", interestedIn: ["man"], ...spot });
  const contact = await createMember({ name: "Contact", gender: "man", interestedIn: ["woman"], ...spot });

  await expectStatus(
    "like",
    request("/swipes", { method: "POST", token: tester.token, body: { userId: contact.id, action: "like" } }),
    200
  );
  const match = await expectStatus(
    "match",
    request("/swipes", { method: "POST", token: contact.token, body: { userId: tester.id, action: "like" } }),
    200
  );
  await expectStatus(
    "first message",
    request(`/conversations/${match.match.conversationId}/messages`, {
      method: "POST",
      token: contact.token,
      body: { body: "Bonjour depuis le setup E2E" }
    }),
    201
  );

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const data = { TEST_EMAIL: tester.email, TEST_PASSWORD: PASSWORD, CONTACT_NAME: "Contact", API_BASE_URL };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(data, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${data.TEST_EMAIL}'\n$env:TEST_PASSWORD='${data.TEST_PASSWORD}'\n$env:CONTACT_NAME='${data.CONTACT_NAME}'\n`
  );
  console.log(`[e2e-ui-setup] PASS TEST_EMAIL=${tester.email}`);
}

main().catch((error) => {
  console.error(`[e2e-ui-setup] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
