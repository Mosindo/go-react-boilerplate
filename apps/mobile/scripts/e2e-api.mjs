// API smoke test of the critical journey against a running API:
// register -> profile -> discovery -> like -> match -> conversation -> notifications -> safety -> deletion.
import { API_URL, api, assert, createMember, expectStatus } from "./lib.mjs";

async function main() {
  console.log(`[e2e-api] API ${API_URL}`);
  await expectStatus(api("/health"), 200, "health");

  const alice = await createMember({ name: "Alice", gender: "woman", interestedIn: ["man"] });
  const bob = await createMember({ name: "Bob", gender: "man", interestedIn: ["woman"] });
  console.log("[e2e-api] members onboarded");

  const me = await expectStatus(api("/me", { token: alice.token }), 200, "me");
  assert(me.profileComplete, "profile must be complete");

  const deck = await expectStatus(api("/discovery?limit=25", { token: alice.token }), 200, "discovery");
  assert(
    deck.profiles.some((p) => p.userId === bob.id),
    "Bob must be suggested to Alice"
  );

  const first = await expectStatus(api("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.id, action: "like" } }), 200, "like");
  assert(!first.matched, "one like is not a match");
  const likes = await expectStatus(api("/likes/received", { token: bob.token }), 200, "likes received");
  assert(
    likes.profiles.some((p) => p.userId === alice.id),
    "Bob sees Alice's like"
  );
  const second = await expectStatus(api("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.id, action: "like" } }), 200, "like back");
  assert(second.matched && second.conversationId, "mutual like creates a match");
  await expectStatus(api("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.id, action: "like" } }), 409, "duplicate like");
  console.log("[e2e-api] match created");

  const conv = second.conversationId;
  await expectStatus(api(`/conversations/${conv}/messages`, { method: "POST", token: alice.token, body: { body: "Salut Bob !" } }), 201, "send");
  const list = await expectStatus(api("/conversations?filter=active", { token: bob.token }), 200, "conversations");
  assert(list.conversations[0]?.unreadCount === 1, "Bob has one unread message");
  const notifs = await expectStatus(api("/notifications", { token: bob.token }), 200, "notifications");
  assert(notifs.unreadCount >= 2, "match + message notifications");
  await expectStatus(api(`/conversations/${conv}/read`, { method: "POST", token: bob.token }), 200, "read");
  const thread = await expectStatus(api(`/conversations/${conv}/messages`, { token: alice.token }), 200, "messages");
  assert(thread.otherLastReadAt, "read receipt visible to Alice");

  const intruder = await createMember({ name: "Mallory", gender: "man", interestedIn: ["woman"] });
  await expectStatus(api(`/conversations/${conv}/messages`, { token: intruder.token }), 404, "intruder cannot read");

  await expectStatus(api("/blocks", { method: "POST", token: alice.token, body: { userId: bob.id } }), 204, "block");
  await expectStatus(api(`/conversations/${conv}/messages`, { token: bob.token }), 404, "conversation gone after block");
  await expectStatus(api("/reports", { method: "POST", token: alice.token, body: { userId: intruder.id, reason: "spam" } }), 201, "report");

  for (const member of [alice, bob, intruder]) {
    await expectStatus(api("/me", { method: "DELETE", token: member.token, body: { password: "Password123" } }), 204, `delete ${member.name}`);
  }
  await expectStatus(api("/me", { token: alice.token }), 401, "deleted account token");
  console.log("[e2e-api] OK");
}

main().catch((error) => {
  console.error(`[e2e-api] FAILED: ${error.message}`);
  process.exit(1);
});
