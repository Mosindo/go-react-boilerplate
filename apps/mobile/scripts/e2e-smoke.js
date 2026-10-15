/* eslint-disable no-console */
"use strict";
// End-to-end smoke test of the whole dating journey against a running API:
// sign up -> profile -> discover -> like -> match -> chat -> notifications -> safety -> delete.
//
//   docker compose up -d && npm run e2e:smoke

const { API_BASE_URL, PASSWORD, assert, createCompleteUser, request } = require("./lib/e2e-helpers");

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);

  const health = await request("/health");
  assert(health.status === 200 && health.json.status === "ok", `health check failed: ${health.status}`);

  const alice = await createCompleteUser({ prefix: "smoke_alice", name: "Alice", gender: "woman", seeks: ["man"], hue: 20 });
  const bob = await createCompleteUser({ prefix: "smoke_bob", name: "Bob", gender: "man", seeks: ["woman"], hue: 140 });
  console.log("[e2e-smoke] onboarding complete for two users");

  const me = await request("/me/profile", { token: alice.token });
  assert(me.json.onboarding.complete === true, "onboarding should be complete");

  const feed = await request("/discover?limit=20", { token: alice.token });
  assert(feed.status === 200, `discover failed: ${feed.status}`);
  assert(feed.json.profiles.some((p) => p.userId === bob.id), "Bob should be proposed to Alice");
  assert(!JSON.stringify(feed.json).includes("latitude"), "discover must not expose coordinates");

  const first = await request("/swipes", { method: "POST", token: alice.token, body: { userId: bob.id, action: "like" } });
  assert(first.status === 200 && first.json.matched === false, "first like must not match");
  const repeat = await request("/swipes", { method: "POST", token: alice.token, body: { userId: bob.id, action: "like" } });
  assert(repeat.status === 409, `a repeated like must be refused (got ${repeat.status})`);

  const second = await request("/swipes", { method: "POST", token: bob.token, body: { userId: alice.id, action: "like" } });
  assert(second.json.matched === true, "second like must create a match");
  const conversationId = second.json.match.conversationId;
  console.log("[e2e-smoke] match created");

  const sent = await request(`/chats/${conversationId}/messages`, { method: "POST", token: bob.token, body: { body: "Salut Alice !" } });
  assert(sent.status === 201, `send failed: ${sent.status}`);
  const inbox = await request("/chats", { token: alice.token });
  const convo = inbox.json.conversations.find((c) => c.id === conversationId);
  assert(convo && convo.unreadCount === 1 && convo.lastMessage.body === "Salut Alice !", "Alice should see one unread message");
  const read = await request(`/chats/${conversationId}/read`, { method: "POST", token: alice.token });
  assert(read.status === 200, "mark read failed");
  const history = await request(`/chats/${conversationId}/messages`, { token: bob.token });
  assert(history.json.peerLastReadAt, "Bob should see that Alice read the message");

  const notifications = await request("/notifications", { token: alice.token });
  assert(notifications.json.notifications.some((n) => n.type === "match"), "Alice should have a match notification");

  const intruder = await createCompleteUser({ prefix: "smoke_eve", name: "Eve", gender: "woman", seeks: ["man"], hue: 90 });
  const spy = await request(`/chats/${conversationId}/messages`, { token: intruder.token });
  assert(spy.status === 404, `a stranger must not read the conversation (got ${spy.status})`);

  const report = await request("/reports", { method: "POST", token: alice.token, body: { userId: bob.id, reason: "spam" } });
  assert(report.status === 202, `report failed: ${report.status}`);
  const block = await request(`/blocks/${bob.id}`, { method: "POST", token: alice.token });
  assert(block.status === 204, `block failed: ${block.status}`);
  const after = await request(`/chats/${conversationId}/messages`, { token: bob.token });
  assert(after.status === 404, "blocking must remove the conversation");
  console.log("[e2e-smoke] safety checks passed");

  for (const user of [alice, bob, intruder]) {
    const deleted = await request("/me", { method: "DELETE", token: user.token, body: { password: PASSWORD } });
    assert(deleted.status === 204, `account deletion failed for ${user.name}: ${deleted.status}`);
    const gone = await request("/me", { token: user.token });
    assert(gone.status === 401, "a deleted account must not authenticate");
  }

  console.log("[e2e-smoke] PASS");
}

main().catch((error) => {
  console.error(`[e2e-smoke] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
