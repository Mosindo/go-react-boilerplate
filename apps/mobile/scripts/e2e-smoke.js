/* API smoke test for the dating flow. Run against a started API: `npm run e2e:smoke`.
 * Use an empty database or unique locations: discovery is shared by every account. */
const { API_BASE_URL, PASSWORD, uniqueEmail, request, assert, register, onboard } = require("./api-flow");

async function step(label, fn) {
  process.stdout.write(`- ${label} ... `);
  const result = await fn();
  console.log("ok");
  return result;
}

async function main() {
  console.log(`[smoke] API: ${API_BASE_URL}`);

  await step("health", async () => assert((await request("/health")).status === 200, "health is not 200"));

  const ana = await step("register Ana", () => register(uniqueEmail("smoke_ana")));
  const ben = await step("register Ben", () => register(uniqueEmail("smoke_ben")));

  await step("login", async () => {
    const res = await request("/auth/login", { method: "POST", body: { email: ana.email, password: PASSWORD } });
    assert(res.status === 200 && res.payload.accessToken, `login: HTTP ${res.status}`);
  });

  await step("profile, preferences and photo", async () => {
    await onboard(ana, { name: "Ana", gender: "woman", lookingFor: "man" });
    await onboard(ben, { name: "Ben", gender: "man", lookingFor: "woman", color: [70, 120, 190] });
    const me = await request("/me", { token: ana.token });
    assert(me.payload.profileComplete === true, "profile should be complete");
  });

  await step("discover loads and hides private data", async () => {
    const res = await request("/discover?limit=20", { token: ana.token });
    assert(res.status === 200, `discover: HTTP ${res.status}`);
    const card = res.payload.profiles.find((p) => p.userId === ben.id);
    assert(card, "Ben should be discoverable by Ana");
    assert(!("birthdate" in card) && !("latitude" in card) && !("email" in card), "private fields leaked");
  });

  const match = await step("like + like back = match", async () => {
    let res = await request("/discover/swipes", { method: "POST", token: ana.token, body: { userId: ben.id, action: "like" } });
    assert(res.status === 200 && res.payload.matched === false, "first like must not match");
    res = await request("/discover/swipes", { method: "POST", token: ben.token, body: { userId: ana.id, action: "like" } });
    assert(res.status === 200 && res.payload.matched === true, "second like must match");
    return res.payload.match;
  });

  await step("chat send / read", async () => {
    const path = `/conversations/${match.conversationId}`;
    let res = await request(`${path}/messages`, { method: "POST", token: ana.token, body: { content: "Salut Ben !" } });
    assert(res.status === 201, `send: HTTP ${res.status}`);
    res = await request("/conversations", { token: ben.token });
    assert(res.payload.conversations[0].unreadCount === 1, "Ben should have one unread message");
    res = await request(`${path}/read`, { method: "POST", token: ben.token });
    assert(res.payload.read === 1, "read should mark one message");
    res = await request(`${path}/messages`, { token: ana.token });
    assert(res.payload.messages[0].readAt, "Ana should see the read receipt");
  });

  await step("notifications", async () => {
    const res = await request("/notifications", { token: ben.token });
    assert(res.status === 200 && res.payload.notifications.some((n) => n.type === "match"), "match notification expected");
  });

  await step("cleanup (delete accounts)", async () => {
    for (const account of [ana, ben]) {
      const res = await request("/me", { method: "DELETE", token: account.token, body: { password: PASSWORD } });
      assert(res.status === 204, `delete ${account.email}: HTTP ${res.status}`);
    }
  });

  console.log("[smoke] all checks passed");
}

main().catch((error) => {
  console.error(`\n[smoke] FAILED: ${error.message}`);
  process.exit(1);
});
