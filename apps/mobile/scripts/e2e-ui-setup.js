// Prepares fixtures for the native Maestro flow: a fully onboarded primary user who has already been liked by
// a second user, so the UI flow can like back, get a match and chat. Writes .e2e/maestro-env.{json,ps1}.
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, createOnboardedUser, expectStatus, request } = require("./lib/api-flow");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);
  const primary = await createOnboardedUser({ label: "primary", gender: "woman", interestedIn: ["man"] });
  const contact = await createOnboardedUser({ label: "contact", gender: "man", interestedIn: ["woman"] });
  expectStatus(
    await request("/swipes", { method: "POST", token: contact.token, body: { targetUserId: primary.id, action: "like" } }),
    200,
    "contact likes primary"
  );

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const env = { TEST_EMAIL: primary.email, TEST_PASSWORD: PASSWORD, CONTACT_NAME: contact.name, API_BASE_URL };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(env, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${env.TEST_EMAIL}'\n$env:TEST_PASSWORD='${env.TEST_PASSWORD}'\n$env:CONTACT_NAME='${env.CONTACT_NAME}'\n`
  );
  fs.writeFileSync(
    path.join(outDir, "maestro-env.sh"),
    `export TEST_EMAIL='${env.TEST_EMAIL}'\nexport TEST_PASSWORD='${env.TEST_PASSWORD}'\nexport CONTACT_NAME='${env.CONTACT_NAME}'\n`
  );
  console.log(`[e2e-ui-setup] PASS — primary ${primary.email}, contact ${contact.name}`);
}

main().catch((err) => {
  console.error(`[e2e-ui-setup] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
