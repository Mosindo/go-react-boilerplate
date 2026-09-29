// Creates the fixtures for the Maestro critical flow (e2e/maestro/critical-flow.yaml):
//  - "primary": a fully onboarded user the UI test signs in with
//  - "contact": a nearby, compatible user who already liked the primary user, so liking back yields a match.
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, assert, completeProfile, registerUser, request, uniqueEmail } = require("./lib/fixtures");

const CONTACT_NAME = "Marco";

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);

  const primaryEmail = uniqueEmail("ui_primary");
  const contactEmail = uniqueEmail("ui_contact");

  const primary = await registerUser(primaryEmail);
  const contact = await registerUser(contactEmail);
  await completeProfile(primary, { firstName: "Ana", gender: "woman", interestedIn: ["man"] });
  await completeProfile(contact, { firstName: CONTACT_NAME, gender: "man", interestedIn: ["woman"] });

  const like = await request("/swipes", {
    method: "POST",
    token: contact.accessToken,
    body: { userId: primary.user.id, action: "like" }
  });
  assert(like.status === 200, `contact like failed: ${like.status} ${JSON.stringify(like.payload)}`);

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });

  const envData = {
    TEST_EMAIL: primaryEmail,
    TEST_PASSWORD: PASSWORD,
    CONTACT_NAME,
    API_BASE_URL
  };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(envData, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${primaryEmail}'\n$env:TEST_PASSWORD='${PASSWORD}'\n$env:CONTACT_NAME='${CONTACT_NAME}'\n`
  );

  console.log("[e2e-ui-setup] PASS");
  console.log(`[e2e-ui-setup] TEST_EMAIL=${primaryEmail}`);
  console.log(`[e2e-ui-setup] CONTACT_NAME=${CONTACT_NAME}`);
}

main().catch((err) => {
  console.error(`[e2e-ui-setup] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
