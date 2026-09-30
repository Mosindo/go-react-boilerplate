// API smoke test of the critical journey against a running API:
// register -> profile -> photo -> discovery -> like -> match -> chat -> read
// -> notifications -> block. Usage: MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:api
const { API_BASE_URL, assert, completeProfile, expectStatus, register, request } = require("./lib/api-fixtures");

async function main() {
  process.stdout.write(`[e2e-smoke] API: ${API_BASE_URL}\n`);
  await expectStatus(request("/health"), 200, "health");

  const alice = await register("smoke_alice");
  const bob = await register("smoke_bob");
  await expectStatus(request("/discovery", { token: alice.token }), 409, "discovery requires a complete profile");

  await completeProfile(alice, { firstName: "Alice", birthdate: "1995-06-15", gender: "woman", interestedIn: ["man"] });
  await completeProfile(bob, { firstName: "Bob", birthdate: "1993-03-03", gender: "man", interestedIn: ["woman"] });

  const deck = await expectStatus(request("/discovery?limit=20", { token: alice.token }), 200, "discovery");
  assert(deck.profiles.some((p) => p.userId === bob.userId), "alice should discover bob");

  const first = await expectStatus(request("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.userId, action: "like" } }), 200, "alice likes bob");
  assert(!first.matched, "no match before reciprocity");
  await expectStatus(request("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.userId, action: "like" } }), 409, "duplicate like");
  const second = await expectStatus(request("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.userId, action: "like" } }), 200, "bob likes alice");
  assert(second.matched && second.match.conversationId, "reciprocal like must match");

  const conversationId = second.match.conversationId;
  await expectStatus(request(`/conversations/${conversationId}/messages`, { method: "POST", token: bob.token, body: { body: "Salut Alice !" } }), 201, "send");
  const messages = await expectStatus(request(`/conversations/${conversationId}/messages`, { token: alice.token }), 200, "list messages");
  assert(messages.messages[0].body === "Salut Alice !", "alice should read bob's message");
  await expectStatus(request(`/conversations/${conversationId}/read`, { method: "POST", token: alice.token }), 204, "mark read");

  const notifications = await expectStatus(request("/notifications", { token: alice.token }), 200, "notifications");
  assert(notifications.notifications.some((n) => n.type === "match"), "alice should have a match notification");

  const outsider = await register("smoke_outsider");
  await expectStatus(request(`/conversations/${conversationId}/messages`, { token: outsider.token }), 404, "outsider cannot read");

  await expectStatus(request("/blocks", { method: "POST", token: alice.token, body: { userId: bob.userId } }), 204, "block");
  await expectStatus(request(`/conversations/${conversationId}/messages`, { token: bob.token }), 404, "conversation removed after block");

  for (const user of [alice, bob, outsider]) {
    await expectStatus(request("/me", { method: "DELETE", token: user.token, body: { password: "Password123" } }), 204, "delete account");
  }
  process.stdout.write("[e2e-smoke] PASS\n");
}

main().catch((err) => {
  process.stderr.write(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}\n`);
  process.exit(1);
});
