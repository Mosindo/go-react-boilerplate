// API smoke test of the whole product loop against a running API:
// register → profile → photo → discover → like → match → chat → read → notifications → block → delete account.
const { API_BASE_URL, PASSWORD, assert, createOnboardedUser, expectStatus, request } = require("./lib/api-flow");

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);
  expectStatus(await request("/health"), 200, "health");

  const alice = await createOnboardedUser({ label: "alice", gender: "woman", interestedIn: ["man"] });
  const bob = await createOnboardedUser({ label: "bob", gender: "man", interestedIn: ["woman"] });

  const me = expectStatus(await request("/profile/me", { token: alice.token }), 200, "my profile");
  assert(me.completion.complete === true, "onboarding must be complete");

  const feed = expectStatus(await request("/discover?limit=30", { token: alice.token }), 200, "discover");
  assert(
    feed.profiles.some((p) => p.id === bob.id),
    "Bob must appear in Alice's feed"
  );
  assert(
    feed.profiles.every((p) => p.latitude === undefined && p.birthDate === undefined),
    "profiles must not leak coordinates or birth date"
  );

  const first = expectStatus(
    await request("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.id, action: "like" } }),
    200,
    "like"
  );
  assert(first.matched === false, "one-sided like must not match");
  expectStatus(
    await request("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.id, action: "like" } }),
    409,
    "duplicate like"
  );

  const second = expectStatus(
    await request("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.id, action: "like" } }),
    200,
    "like back"
  );
  assert(second.matched === true && second.match.conversationId, "reciprocal like must match");
  const conversationId = second.match.conversationId;

  expectStatus(
    await request(`/conversations/${conversationId}/messages`, { method: "POST", token: bob.token, body: { content: "Salut Alice !" } }),
    201,
    "send"
  );
  const conversations = expectStatus(await request("/conversations", { token: alice.token }), 200, "conversations");
  assert(conversations.conversations[0].unreadCount === 1, "Alice must have one unread message");
  expectStatus(await request(`/conversations/${conversationId}/read`, { method: "POST", token: alice.token }), 200, "mark read");
  const history = expectStatus(await request(`/conversations/${conversationId}/messages`, { token: bob.token }), 200, "history");
  assert(history.messages.length === 1 && history.messages[0].readAt, "Bob must see the read receipt");

  const outsider = await createOnboardedUser({ label: "carl", gender: "man", interestedIn: ["woman"] });
  expectStatus(await request(`/conversations/${conversationId}/messages`, { token: outsider.token }), 404, "outsider cannot read");

  const notifications = expectStatus(await request("/notifications", { token: alice.token }), 200, "notifications");
  assert(
    notifications.notifications.some((n) => n.type === "match"),
    "Alice must have a match notification"
  );

  expectStatus(await request("/blocks", { method: "POST", token: alice.token, body: { userId: bob.id } }), 204, "block");
  expectStatus(await request(`/conversations/${conversationId}/messages`, { token: bob.token }), 404, "blocked conversation");

  for (const user of [alice, bob, outsider]) {
    expectStatus(await request("/me", { method: "DELETE", token: user.token, body: { password: PASSWORD } }), 204, "delete account");
  }
  console.log("[e2e-smoke] PASS");
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
