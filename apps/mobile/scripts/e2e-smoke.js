// Full product journey against a running API:
// register -> profile -> photos -> discover -> like -> match -> chat -> read -> notifications -> block -> delete.
const { API_BASE_URL, PASSWORD, assert, onboard, register, request, uniqueEmail } = require("./lib/api");

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);

  const health = await request("/health");
  assert(health.status === 200, `health failed: ${health.status}`);

  const alice = await register(uniqueEmail("alice"));
  const bob = await register(uniqueEmail("bob"));

  const login = await request("/auth/login", { method: "POST", body: { email: alice.email, password: PASSWORD } });
  assert(login.status === 200 && login.payload.user.id === alice.id, `login failed: ${login.status}`);

  const before = await request("/me/profile", { token: alice.token });
  assert(before.payload.isComplete === false, "a new account must start with an incomplete profile");

  await onboard(alice, { name: "Alice", gender: "woman", seeking: ["man"] });
  await onboard(bob, { name: "Bob", gender: "man", seeking: ["woman"] });
  const after = await request("/me/profile", { token: alice.token });
  assert(after.payload.isComplete === true, "profile should be complete after onboarding");

  const deck = await request("/discover?limit=20", { token: alice.token });
  assert(deck.status === 200, `discover failed: ${deck.status}`);
  assert(deck.payload.profiles.some((p) => p.id === bob.id), "alice should discover bob");
  assert(!JSON.stringify(deck.payload).includes("latitude"), "discover must never expose coordinates");

  const first = await request("/swipes", { method: "POST", token: alice.token, body: { targetId: bob.id, action: "like" } });
  assert(first.status === 200 && first.payload.matched === false, "a one-sided like must not match");

  const second = await request("/swipes", { method: "POST", token: bob.token, body: { targetId: alice.id, action: "like" } });
  assert(second.status === 200 && second.payload.matched === true, "a reciprocal like must match");
  const matchId = second.payload.matchId;

  const sent = await request(`/conversations/${matchId}/messages`, { method: "POST", token: alice.token, body: { body: "Salut Bob !" } });
  assert(sent.status === 201, `send message failed: ${sent.status}`);

  const inbox = await request("/conversations", { token: bob.token });
  assert(inbox.payload.conversations[0].unreadCount === 1, "bob should see one unread message");

  const read = await request(`/conversations/${matchId}/read`, { method: "POST", token: bob.token });
  assert(read.status === 204, `mark read failed: ${read.status}`);
  const thread = await request(`/conversations/${matchId}/messages`, { token: alice.token });
  assert(thread.payload.messages[0].readAt, "alice should see the read receipt");

  const notifications = await request("/notifications", { token: bob.token });
  assert(notifications.payload.notifications.some((n) => n.type === "match"), "bob should have a match notification");
  const readAll = await request("/notifications/read-all", { method: "POST", token: bob.token });
  assert(readAll.status === 204, `read-all failed: ${readAll.status}`);

  const block = await request("/blocks", { method: "POST", token: alice.token, body: { userId: bob.id } });
  assert(block.status === 204, `block failed: ${block.status}`);
  const gone = await request(`/conversations/${matchId}/messages`, { token: bob.token });
  assert(gone.status === 404, "blocking must dissolve the conversation");

  const del = await request("/me", { method: "DELETE", token: alice.token, body: { password: PASSWORD } });
  assert(del.status === 204, `account deletion failed: ${del.status}`);
  const relogin = await request("/auth/login", { method: "POST", body: { email: alice.email, password: PASSWORD } });
  assert(relogin.status === 401, "a deleted account must not be able to log in");

  console.log("[e2e-smoke] PASS register/profile/photos/discover/like/match/chat/read/notifications/block/delete");
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
