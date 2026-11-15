/* API smoke test against a running stack: register → profile → photo →
 * discovery → like → match → chat → read → notifications → block. */
const { createCompleteUser, expectStatus, request } = require("./lib");

async function main() {
  expectStatus(await request("GET", "/health"), 200, "health");
  const alice = await createCompleteUser({ name: "Alice", gender: "woman", seeking: ["man"] });
  const bob = await createCompleteUser({ name: "Bob", gender: "man", seeking: ["woman"] });

  const feed = expectStatus(await request("GET", "/discovery?limit=30", { token: alice.token }), 200, "discovery");
  if (!feed.profiles.some((p) => p.userId === bob.id)) throw new Error("bob missing from alice's discovery");

  const first = expectStatus(
    await request("POST", "/swipes", { token: alice.token, body: { targetUserId: bob.id, action: "like" } }),
    200,
    "alice likes bob"
  );
  if (first.matched) throw new Error("one-sided like must not match");
  const second = expectStatus(
    await request("POST", "/swipes", { token: bob.token, body: { targetUserId: alice.id, action: "like" } }),
    200,
    "bob likes alice"
  );
  if (!second.matched) throw new Error("mutual like must match");
  const conversationId = second.match.conversationId;

  expectStatus(
    await request("POST", `/conversations/${conversationId}/messages`, { token: bob.token, body: { body: "Salut Alice !" } }),
    201,
    "send"
  );
  const page = expectStatus(
    await request("GET", `/conversations/${conversationId}/messages`, { token: alice.token }),
    200,
    "read messages"
  );
  if (page.messages[0]?.body !== "Salut Alice !") throw new Error("message not delivered");
  expectStatus(await request("POST", `/conversations/${conversationId}/read`, { token: alice.token }), 200, "mark read");

  const notifications = expectStatus(await request("GET", "/notifications", { token: alice.token }), 200, "notifications");
  if (!notifications.notifications.some((n) => n.type === "match")) throw new Error("match notification missing");

  expectStatus(await request("POST", "/blocks", { token: alice.token, body: { userId: bob.id } }), 204, "block");
  expectStatus(
    await request("GET", `/conversations/${conversationId}/messages`, { token: bob.token }),
    404,
    "conversation gone after block"
  );

  for (const u of [alice, bob]) {
    expectStatus(await request("DELETE", "/me", { token: u.token, body: { password: u.password } }), 204, `delete ${u.name}`);
  }
  console.log("[e2e:api] OK — register, profile, photo, discovery, like, match, chat, read, notifications, block, delete");
}

main().catch((error) => {
  console.error("[e2e:api] FAILED:", error.message);
  process.exit(1);
});
