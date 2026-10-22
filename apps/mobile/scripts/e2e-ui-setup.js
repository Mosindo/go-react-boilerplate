/* Prepares two onboarded accounts for the Maestro flow and prints them as env assignments.
 * The primary account already liked the contact, so the contact's like creates a match. */
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, uniqueEmail, request, assert, register, onboard } = require("./api-flow");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);
  const primary = await onboard(await register(uniqueEmail("ui_primary")), { name: "Primary", gender: "woman", lookingFor: "man" });
  const contact = await onboard(await register(uniqueEmail("ui_contact")), {
    name: "Contact",
    gender: "man",
    lookingFor: "woman",
    color: [70, 120, 190]
  });

  const like = await request("/discover/swipes", { method: "POST", token: primary.token, body: { userId: contact.id, action: "like" } });
  assert(like.status === 200, `like: HTTP ${like.status}`);

  const values = { TEST_EMAIL: primary.email, TEST_PASSWORD: PASSWORD, CONTACT_EMAIL: contact.email, CONTACT_NAME: "Contact" };
  const dir = path.join(__dirname, "..", ".e2e");
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(
    path.join(dir, "maestro.env"),
    Object.entries(values)
      .map(([k, v]) => `${k}=${v}`)
      .join("\n") + "\n"
  );
  fs.writeFileSync(
    path.join(dir, "maestro-env.ps1"),
    Object.entries(values)
      .map(([k, v]) => `$env:${k} = "${v}"`)
      .join("\n") + "\n"
  );
  console.log(`[e2e-ui-setup] wrote fixtures to ${dir} (primary: ${primary.email})`);
}

main().catch((error) => {
  console.error(`[e2e-ui-setup] FAILED: ${error.message}`);
  process.exit(1);
});
