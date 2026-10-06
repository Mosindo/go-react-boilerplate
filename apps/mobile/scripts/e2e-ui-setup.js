/* eslint-disable no-console */
// Prepares fixtures for the Maestro UI flow: a fully onboarded test user (the
// flow signs in with it) and a "contact" who already liked that user, so
// liking the contact back in the UI produces a match.
// (Onboarding itself needs a photo from the device gallery, which Maestro cannot
// provide portably, so it is covered by the API smoke test and the Go suite.)

const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, request, expectStatus, registerAndOnboard } = require("./lib-e2e");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);

  const contact = await registerAndOnboard({ prefix: "ui_contact", name: "Camille", gender: "woman", interestedIn: ["man"], age: 28 });
  const primary = await registerAndOnboard({ prefix: "ui_primary", name: "Alex", gender: "man", interestedIn: ["woman"], age: 30 });
  const testEmail = primary.email;
  expectStatus(
    await request("/swipes", { method: "POST", token: contact.token, body: { userId: primary.id, action: "like" } }),
    201,
    "contact likes the test user"
  );

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const envData = { TEST_EMAIL: testEmail, TEST_PASSWORD: PASSWORD, CONTACT_NAME: "Camille", API_BASE_URL };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(envData, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${testEmail}'\n$env:TEST_PASSWORD='${PASSWORD}'\n$env:CONTACT_NAME='Camille'\n`
  );
  fs.writeFileSync(
    path.join(outDir, "maestro-env.sh"),
    `export TEST_EMAIL='${testEmail}'\nexport TEST_PASSWORD='${PASSWORD}'\nexport CONTACT_NAME='Camille'\n`
  );

  expectStatus(await request("/health"), 200, "health");

  console.log("[e2e-ui-setup] PASS");
  console.log(`[e2e-ui-setup] TEST_EMAIL=${testEmail}`);
  console.log(`[e2e-ui-setup] env files: ${outDir}`);
}

main().catch((err) => {
  console.error(`[e2e-ui-setup] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
