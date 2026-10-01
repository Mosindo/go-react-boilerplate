/* eslint-disable no-console */
// End-to-end API smoke test of the critical dating journey, run against a live API:
//   register -> profile + photo -> discover -> like -> match -> chat -> read -> notifications -> safety -> deletion
const { API_BASE_URL, PASSWORD, assert, createMember, expectStatus, randomSpot, request, uniqueEmail } = require("./lib/api");

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);
  const health = await request("/health");
  assert(health.status === 200, `health check failed: ${health.status}`);

  const spot = randomSpot();
  const alice = await createMember({ name: "Alice", gender: "woman", interestedIn: ["man"], ...spot, interests: ["music", "art"] });
  const bob = await createMember({ name: "Bob", gender: "man", interestedIn: ["woman"], ...spot, interests: ["music", "travel"] });
  const eve = await createMember({ name: "Eve", gender: "woman", interestedIn: ["man"], ...spot });
  console.log("[e2e-smoke] members onboarded");

  const me = await expectStatus("/me", request("/me", { token: alice.token }), 200);
  assert(me.profileComplete === true, "profile should be complete after onboarding");

  // discovery
  const discover = await expectStatus("discover", request("/discover?limit=20", { token: alice.token }), 200);
  const ids = discover.profiles.map((p) => p.id);
  assert(ids.includes(bob.id), "alice should discover bob");
  assert(!ids.includes(eve.id), "alice should not see eve (preferences)");
  assert(!JSON.stringify(discover).includes("@e2e.invalid"), "discovery must not leak emails");
  console.log("[e2e-smoke] discovery ok");

  // matching
  const like1 = await expectStatus(
    "like 1",
    request("/swipes", { method: "POST", token: alice.token, body: { userId: bob.id, action: "like" } }),
    200
  );
  assert(like1.matched === false, "first like must not match");
  const again = await expectStatus(
    "like repeat",
    request("/swipes", { method: "POST", token: alice.token, body: { userId: bob.id, action: "like" } }),
    200
  );
  assert(again.alreadySwiped === true, "repeat like must be idempotent");
  const like2 = await expectStatus(
    "like 2",
    request("/swipes", { method: "POST", token: bob.token, body: { userId: alice.id, action: "like" } }),
    200
  );
  assert(like2.matched === true && like2.match.conversationId, "reciprocal like must match");
  const conversationId = like2.match.conversationId;
  console.log("[e2e-smoke] match ok");

  // chat + permissions
  await expectStatus(
    "send",
    request(`/conversations/${conversationId}/messages`, { method: "POST", token: alice.token, body: { body: "Salut Bob !" } }),
    201
  );
  await expectStatus("outsider read", request(`/conversations/${conversationId}/messages`, { token: eve.token }), 404);
  const inbox = await expectStatus("bob inbox", request("/conversations", { token: bob.token }), 200);
  assert(inbox.totalUnread === 1 && inbox.conversations[0].lastMessage.body === "Salut Bob !", "bob should see 1 unread message");
  await expectStatus("mark read", request(`/conversations/${conversationId}/read`, { method: "POST", token: bob.token }), 200);
  const thread = await expectStatus("alice thread", request(`/conversations/${conversationId}/messages`, { token: alice.token }), 200);
  assert(thread.messages[0].readAt, "alice should see the read receipt");
  console.log("[e2e-smoke] chat ok");

  // notifications
  const notifs = await expectStatus("notifications", request("/notifications", { token: bob.token }), 200);
  assert(
    notifs.notifications.some((n) => n.type === "match"),
    "bob should have a match notification"
  );
  await expectStatus("read all", request("/notifications/read-all", { method: "POST", token: bob.token }), 200);
  console.log("[e2e-smoke] notifications ok");

  // safety
  await expectStatus("report", request("/reports", { method: "POST", token: alice.token, body: { userId: eve.id, reason: "spam" } }), 201);
  await expectStatus("block", request("/blocks", { method: "POST", token: alice.token, body: { userId: bob.id } }), 204);
  await expectStatus("chat closed after block", request(`/conversations/${conversationId}/messages`, { token: bob.token }), 404);
  console.log("[e2e-smoke] safety ok");

  // account deletion
  await expectStatus("delete wrong password", request("/me", { method: "DELETE", token: bob.token, body: { password: "nope-nope" } }), 403);
  await expectStatus("delete", request("/me", { method: "DELETE", token: bob.token, body: { password: PASSWORD } }), 204);
  await expectStatus("token revoked", request("/me", { token: bob.token }), 401);
  const unknown = await request("/auth/login", { method: "POST", body: { email: uniqueEmail("ghost"), password: PASSWORD } });
  assert(unknown.status === 401, "unknown account must not log in");
  console.log("[e2e-smoke] account deletion ok");

  // cleanup the other members so reruns stay tidy
  for (const member of [alice, eve]) {
    await request("/me", { method: "DELETE", token: member.token, body: { password: PASSWORD } });
  }
  console.log("[e2e-smoke] PASS");
}

main().catch((error) => {
  console.error(`[e2e-smoke] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
