/* eslint-disable no-console */
"use strict";
// Prepares fixtures for the Maestro UI flow: a fully onboarded user, and a contact who already
// liked them. In the app, the user then only has to log in, like the contact and chat.

const fs = require("fs");
const path = require("path");
const { API_BASE_URL, PASSWORD, createCompleteUser, request } = require("./lib/e2e-helpers");

async function main() {
  console.log(`[e2e-ui-setup] API: ${API_BASE_URL}`);

  const user = await createCompleteUser({ prefix: "ui_user", name: "Samuel", gender: "man", seeks: ["woman"], hue: 30 });
  const contact = await createCompleteUser({ prefix: "ui_contact", name: "Clara", gender: "woman", seeks: ["man"], hue: 170 });

  const like = await request("/swipes", { method: "POST", token: contact.token, body: { userId: user.id, action: "like" } });
  if (like.status !== 200) {
    throw new Error(`contact like failed: ${like.status} ${JSON.stringify(like.json)}`);
  }

  const outDir = path.join(process.cwd(), ".e2e");
  fs.mkdirSync(outDir, { recursive: true });
  const env = { TEST_EMAIL: user.email, TEST_PASSWORD: PASSWORD, CONTACT_NAME: contact.name, API_BASE_URL };
  fs.writeFileSync(path.join(outDir, "maestro-env.json"), JSON.stringify(env, null, 2));

  console.log("[e2e-ui-setup] PASS");
  console.log(`[e2e-ui-setup] TEST_EMAIL=${user.email}`);
  console.log(`[e2e-ui-setup] CONTACT_NAME=${contact.name}`);
}

main().catch((error) => {
  console.error(`[e2e-ui-setup] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
