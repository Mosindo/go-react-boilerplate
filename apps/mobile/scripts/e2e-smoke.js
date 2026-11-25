// API smoke test of the critical dating flow:
// register → profile → photo → discover → like → match → chat → read → block → delete.
// Usage: E2E_API_URL=http://localhost:18080 npm run e2e:smoke
const { API_URL, PASSWORD, assert, expectStatus, onboard, register, request } = require("../e2e/lib");

async function main() {
  console.log(`[e2e-smoke] API ${API_URL}`);
  await expectStatus(request("/health"), 200, "health");

  // Coordinates in an empty area so demo seed data does not interfere.
  const area = { latitude: -45.01 - Math.random(), longitude: -140.01 - Math.random() };
  const alice = await register("alice");
  const bob = await register("bob");
  await onboard(alice, { firstName: "Alice", gender: "woman", interestedIn: ["man"], ...area });
  await onboard(bob, { firstName: "Bob", gender: "man", interestedIn: ["woman"], ...area });
  console.log("[e2e-smoke] users onboarded");

  const login = await expectStatus(
    request("/auth/login", { method: "POST", body: { email: alice.email, password: PASSWORD } }),
    200,
    "login"
  );
  assert(login.user.id === alice.id, "login returns the same user");

  const deck = await expectStatus(request("/discover", { token: alice.token }), 200, "discover");
  assert(
    deck.profiles.some((p) => p.userId === bob.id),
    "bob is in alice's deck"
  );
  assert(!JSON.stringify(deck).includes("@"), "discovery never exposes emails");

  const first = await expectStatus(
    request("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.id, action: "like" } }),
    200,
    "like"
  );
  assert(first.match === null, "no match on one-sided like");
  const second = await expectStatus(
    request("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.id, action: "like" } }),
    200,
    "like back"
  );
  assert(second.match && second.match.conversationId, "mutual like creates a match");
  const conversationId = second.match.conversationId;
  console.log("[e2e-smoke] match created");

  await expectStatus(
    request(`/conversations/${conversationId}/messages`, { method: "POST", token: bob.token, body: { body: "Bonjour Alice !" } }),
    201,
    "send"
  );
  const inbox = await expectStatus(request("/conversations", { token: alice.token }), 200, "conversations");
  assert(inbox.conversations[0]?.unreadCount === 1, "alice has one unread message");
  await expectStatus(request(`/conversations/${conversationId}/read`, { method: "POST", token: alice.token }), 200, "read");
  const thread = await expectStatus(request(`/conversations/${conversationId}/messages`, { token: bob.token }), 200, "messages");
  assert(thread.peerLastReadAt !== null, "bob sees the read receipt");

  const notifications = await expectStatus(request("/notifications", { token: alice.token }), 200, "notifications");
  assert(
    notifications.notifications.some((n) => n.type === "match"),
    "alice got a match notification"
  );

  const outsider = await register("eve");
  await expectStatus(request(`/conversations/${conversationId}/messages`, { token: outsider.token }), 404, "outsider cannot read");

  await expectStatus(request("/blocks", { method: "POST", token: alice.token, body: { userId: bob.id } }), 204, "block");
  await expectStatus(request(`/conversations/${conversationId}/messages`, { token: bob.token }), 404, "chat closed after block");
  console.log("[e2e-smoke] block enforced");

  for (const u of [alice, bob, outsider]) {
    await expectStatus(request("/me", { method: "DELETE", token: u.token, body: { password: PASSWORD } }), 204, "delete account");
  }
  console.log("[e2e-smoke] OK");
}

main().catch((error) => {
  console.error(`[e2e-smoke] FAILED: ${error.message}`);
  process.exit(1);
});
