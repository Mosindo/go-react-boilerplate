// End-to-end API smoke test of the critical journey:
// register -> profile -> photo -> discovery -> like -> match -> chat -> notifications
// -> block -> account deletion. Run against a live API:
//   E2E_API_URL=http://localhost:18080 npm run e2e:api
const { API_BASE_URL, createMember, expectStatus, request } = require("../e2e/fixtures");

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function step(name, fn) {
  process.stdout.write(`- ${name} ... `);
  const result = await fn();
  console.log("ok");
  return result;
}

async function main() {
  console.log(`[e2e-smoke] API ${API_BASE_URL}`);
  await step("health", async () => expectStatus(await request("/health"), 200, "health"));

  const nora = await step("onboard Nora", () =>
    createMember({ prefix: "smoke_nora", firstName: "Nora", gender: "woman", interestedIn: ["man"] })
  );
  const theo = await step("onboard Théo", () =>
    createMember({ prefix: "smoke_theo", firstName: "Théo", gender: "man", interestedIn: ["woman"] })
  );

  await step("discovery shows the other member", async () => {
    const feed = expectStatus(await request("/discovery?limit=20", { token: nora.token }), 200, "discovery");
    // The shared database may contain other members: walk the feed until Théo shows up.
    let found = feed.profiles.some((p) => p.userId === theo.userId);
    for (let i = 0; !found && i < 10; i++) {
      for (const p of feed.profiles) {
        await request("/swipes", { method: "POST", token: nora.token, body: { targetUserId: p.userId, decision: "pass" } });
      }
      const next = expectStatus(await request("/discovery?limit=20", { token: nora.token }), 200, "discovery");
      found = next.profiles.some((p) => p.userId === theo.userId);
      if (next.profiles.length === 0) break;
      feed.profiles = next.profiles;
    }
    assert(found, "Théo should be discoverable by Nora");
  });

  await step("one-sided like does not match", async () => {
    const res = expectStatus(
      await request("/swipes", { method: "POST", token: nora.token, body: { targetUserId: theo.userId, decision: "like" } }),
      200,
      "like"
    );
    assert(res.matched === false, "no match expected yet");
  });

  await step("mutual like creates a match", async () => {
    const res = expectStatus(
      await request("/swipes", { method: "POST", token: theo.token, body: { targetUserId: nora.userId, decision: "like" } }),
      201,
      "like back"
    );
    assert(res.matched && res.match.user.userId === nora.userId, "match expected");
  });

  await step("chat send and read", async () => {
    expectStatus(
      await request(`/chats/${theo.userId}/messages`, { method: "POST", token: nora.token, body: { content: "Salut Théo !" } }),
      201,
      "send"
    );
    const chats = expectStatus(await request("/chats", { token: theo.token }), 200, "chats");
    const chat = chats.chats.find((c) => c.user.userId === nora.userId);
    assert(chat && chat.unreadCount === 1, "unread message expected");
    expectStatus(await request(`/chats/${nora.userId}/read`, { method: "POST", token: theo.token }), 200, "read");
    const thread = expectStatus(await request(`/chats/${theo.userId}/messages`, { token: nora.token }), 200, "thread");
    assert(thread.messages.length === 1 && thread.otherLastReadAt, "read receipt expected");
  });

  await step("notifications read flow", async () => {
    const list = expectStatus(await request("/notifications", { token: theo.token }), 200, "notifications");
    assert(
      list.notifications.some((n) => n.type === "match.created"),
      "match notification expected"
    );
    assert(list.unreadCount > 0, "unread notifications expected");
    expectStatus(await request("/notifications/read-all", { method: "POST", token: theo.token }), 204, "read all");
    const after = expectStatus(await request("/notifications/unread-count", { token: theo.token }), 200, "unread count");
    assert(after.unreadCount === 0, "all notifications should be read");
  });

  await step("block removes the conversation", async () => {
    expectStatus(await request("/blocks", { method: "POST", token: theo.token, body: { userId: nora.userId } }), 204, "block");
    expectStatus(await request(`/chats/${theo.userId}/messages`, { token: nora.token }), 404, "chat after block");
  });

  await step("account deletion", async () => {
    for (const m of [nora, theo]) {
      expectStatus(await request("/me", { method: "DELETE", token: m.token, body: { password: "Password123" } }), 204, "delete");
      expectStatus(await request("/me", { token: m.token }), 401, "token after delete");
    }
  });

  console.log("[e2e-smoke] PASS");
}

main().catch((error) => {
  console.error(`\n[e2e-smoke] FAIL: ${error.message}`);
  process.exit(1);
});
