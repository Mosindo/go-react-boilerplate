// Prepares fixtures for the native Maestro flow: a complete test account and
// a contact who already likes it, so liking the contact in the UI matches.
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, completeProfile, expectStatus, register, request } = require("./lib/api-fixtures");

async function main() {
  process.stdout.write(`[e2e-ui-setup] API: ${API_BASE_URL}\n`);
  const primary = await register("maestro_primary");
  const contact = await register("maestro_contact");
  await completeProfile(primary, { firstName: "Maestro", birthdate: "1994-01-01", gender: "woman", interestedIn: ["man"] });
  await completeProfile(contact, { firstName: "Camille", birthdate: "1995-02-02", gender: "man", interestedIn: ["woman"] });
  await expectStatus(request("/swipes", { method: "POST", token: contact.token, body: { targetUserId: primary.userId, action: "like" } }), 200, "contact likes primary");

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const env = { TEST_EMAIL: primary.email, TEST_PASSWORD: PASSWORD, CONTACT_NAME: "Camille" };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(env, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    Object.entries(env)
      .map(([k, v]) => `$env:${k}='${v}'`)
      .join("\n") + "\n"
  );
  process.stdout.write(`[e2e-ui-setup] PASS TEST_EMAIL=${primary.email}\n`);
}

main().catch((err) => {
  process.stderr.write(`[e2e-ui-setup] FAIL ${err instanceof Error ? err.message : String(err)}\n`);
  process.exit(1);
});
