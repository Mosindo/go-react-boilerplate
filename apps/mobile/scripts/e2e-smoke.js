/**
 * API smoke test for a running stack (docker compose or local):
 * register -> profile -> photo -> discover -> like -> match -> chat -> notifications -> safety.
 */
const { API_BASE_URL, assert, createMember, expectStatus, request } = require("./lib/dating-api");

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);
  await expectStatus("health", 200, request("/health"));

  const alice = await createMember("alice", { name: "Alice", gender: "woman", seeking: "man" });
  const bob = await createMember("bob", { name: "Bob", gender: "man", seeking: "woman" });
  console.log("[e2e-smoke] users registered with profile and photo");

  const deck = await expectStatus("discover", 200, request("/discover?limit=20", { token: alice.token }));
  assert(
    deck.profiles.some((p) => p.userId === bob.id),
    "Bob should be discoverable by Alice"
  );
  assert(
    deck.profiles.every((p) => !("latitude" in p) && !("email" in p)),
    "public profiles must not leak private fields"
  );

  const first = await expectStatus(
    "like",
    200,
    request("/swipes", { method: "POST", token: alice.token, body: { userId: bob.id, action: "like" } })
  );
  assert(first.matched === false, "a one-sided like must not match");
  const second = await expectStatus(
    "like back",
    200,
    request("/swipes", { method: "POST", token: bob.token, body: { userId: alice.id, action: "like" } })
  );
  assert(second.matched === true, "a reciprocal like must match");
  const conversationId = second.match.conversationId;
  console.log("[e2e-smoke] match created");

  await expectStatus(
    "send",
    201,
    request(`/conversations/${conversationId}/messages`, {
      method: "POST",
      token: alice.token,
      body: { content: "Salut !" }
    })
  );
  const thread = await expectStatus(
    "read thread",
    200,
    request(`/conversations/${conversationId}/messages`, { token: bob.token })
  );
  assert(thread.messages[0].content === "Salut !", "Bob should read Alice's message");
  const list = await expectStatus("conversations", 200, request("/conversations", { token: bob.token }));
  assert(list.unreadTotal === 1, "Bob should have one unread message");
  await expectStatus(
    "mark read",
    200,
    request(`/conversations/${conversationId}/read`, { method: "POST", token: bob.token })
  );

  const outsider = await createMember("eve", { name: "Eve", gender: "man", seeking: "woman" });
  await expectStatus(
    "outsider cannot read",
    404,
    request(`/conversations/${conversationId}/messages`, { token: outsider.token })
  );

  const notifications = await expectStatus("notifications", 200, request("/notifications", { token: bob.token }));
  assert(
    notifications.notifications.some((n) => n.type === "match"),
    "Bob should have a match notification"
  );

  await expectStatus(
    "block",
    204,
    request("/blocks", { method: "POST", token: bob.token, body: { userId: alice.id } })
  );
  await expectStatus(
    "blocked conversation",
    404,
    request(`/conversations/${conversationId}/messages`, { token: alice.token })
  );

  for (const member of [alice, bob, outsider]) {
    await expectStatus(
      "delete account",
      204,
      request("/me", {
        method: "DELETE",
        token: member.token,
        body: { password: require("./lib/dating-api").PASSWORD }
      })
    );
  }
  console.log("[e2e-smoke] PASS");
}

main().catch((error) => {
  console.error(`[e2e-smoke] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
