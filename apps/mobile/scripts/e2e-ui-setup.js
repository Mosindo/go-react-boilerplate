/* eslint-disable no-console */
// Prepares fixtures for the Maestro UI flow: a primary member who has just been liked by a contact,
// so the first card in discovery is the contact and a like produces an instant match.
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, assert, createMember, randomOrigin, request } = require("./lib/api");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);
  const origin = randomOrigin();
  const primary = await createMember("ui_primary", { name: "Alex", gender: "man", interestedIn: ["woman"], origin });
  const contact = await createMember("ui_contact", { name: "Casey", gender: "woman", interestedIn: ["man"], origin });

  const like = await request("/swipes", { token: contact.token, json: { targetId: primary.id, action: "like" } });
  assert(like.status === 200, `contact like failed: ${like.status}`);

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const env = { TEST_EMAIL: primary.email, TEST_PASSWORD: PASSWORD, CONTACT_NAME: "Casey", API_BASE_URL };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(env, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${primary.email}'\n$env:TEST_PASSWORD='${PASSWORD}'\n$env:CONTACT_NAME='Casey'\n`
  );
  console.log("[e2e-ui-setup] PASS");
  console.log(`[e2e-ui-setup] TEST_EMAIL=${primary.email}`);
}

main().catch((err) => {
  console.error(`[e2e-ui-setup] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
