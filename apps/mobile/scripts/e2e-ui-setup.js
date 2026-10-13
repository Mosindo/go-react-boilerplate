// Prepares two ready-to-use members for the Maestro UI flow. The contact has
// already liked the primary user, so liking them back produces a match.
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, assert, createMember, request } = require("./lib/dating-api");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);
  const primary = await createMember({ prefix: "ui_primary", name: "Alice", gender: "woman", interestedIn: ["man"] });
  const contact = await createMember({ prefix: "ui_contact", name: "Bob", gender: "man", interestedIn: ["woman"] });

  const like = await request("/likes", { method: "POST", token: contact.token, json: { userId: primary.id, action: "like" } });
  assert(like.status === 200, `seed like failed: ${like.status}`);

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const envData = { TEST_EMAIL: primary.email, TEST_PASSWORD: PASSWORD, CONTACT_NAME: "Bob", API_BASE_URL };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(envData, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${primary.email}'\n$env:TEST_PASSWORD='${PASSWORD}'\n$env:CONTACT_NAME='Bob'\n`
  );
  console.log("[e2e-ui-setup] PASS");
  console.log(`[e2e-ui-setup] TEST_EMAIL=${primary.email}`);
}

main().catch((error) => {
  console.error(`[e2e-ui-setup] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
