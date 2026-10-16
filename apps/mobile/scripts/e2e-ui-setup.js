// Prepares fixtures for the Maestro UI flow: the "primary" user logs in through the UI;
// "Camille" has already liked them, so a single like in the app produces a match.
const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, assert, onboard, register, request, uniqueEmail } = require("./lib/api");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);

  const primary = await register(uniqueEmail("ui_primary"));
  const camille = await register(uniqueEmail("ui_camille"));
  await onboard(primary, { name: "Hugo", gender: "man", seeking: ["woman"] });
  await onboard(camille, { name: "Camille", gender: "woman", seeking: ["man"] });

  const like = await request("/swipes", { method: "POST", token: camille.token, body: { targetId: primary.id, action: "like" } });
  assert(like.status === 200, `seed like failed: ${like.status}`);

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });

  const envData = { TEST_EMAIL: primary.email, TEST_PASSWORD: PASSWORD, MATCH_NAME: "Camille", API_BASE_URL };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(envData, null, 2));
  fs.writeFileSync(
    path.join(outDir, "maestro-env.ps1"),
    `$env:TEST_EMAIL='${primary.email}'\n$env:TEST_PASSWORD='${PASSWORD}'\n$env:MATCH_NAME='Camille'\n`
  );
  fs.writeFileSync(
    path.join(outDir, "maestro-env.sh"),
    `export TEST_EMAIL='${primary.email}'\nexport TEST_PASSWORD='${PASSWORD}'\nexport MATCH_NAME='Camille'\n`
  );

  console.log("[e2e-ui-setup] PASS");
  console.log(`[e2e-ui-setup] TEST_EMAIL=${primary.email}`);
}

main().catch((err) => {
  console.error(`[e2e-ui-setup] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
