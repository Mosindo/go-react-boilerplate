// API smoke test of the critical path, run against a live API:
// register -> profile -> photo -> discover -> like -> match -> chat -> notifications.
const { API_BASE_URL, PASSWORD, assert, createMember, request } = require("./lib/dating-api");

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);

  const health = await request("/health");
  assert(health.status === 200, `health failed: ${health.status}`);

  const alice = await createMember({ prefix: "smoke_a", name: "Alice", gender: "woman", interestedIn: ["man"] });
  const bob = await createMember({ prefix: "smoke_b", name: "Bob", gender: "man", interestedIn: ["woman"] });

  const login = await request("/auth/login", { method: "POST", json: { email: alice.email, password: PASSWORD } });
  assert(login.status === 200 && login.payload.user.id === alice.id, "login failed");

  const me = await request("/me", { token: alice.token });
  assert(me.status === 200 && me.payload.email === alice.email, "/me failed");

  const deck = await request("/discover?limit=20", { token: alice.token });
  assert(deck.status === 200, `discover failed: ${deck.status}`);
  assert(
    deck.payload.profiles.some((p) => p.id === bob.id),
    "bob should be discoverable by alice"
  );
  assert(!JSON.stringify(deck.payload).includes("latitude"), "discover must not expose coordinates");

  const first = await request("/likes", { method: "POST", token: alice.token, json: { userId: bob.id, action: "like" } });
  assert(first.status === 200 && first.payload.matched === false, "first like must not match");
  const repeat = await request("/likes", { method: "POST", token: alice.token, json: { userId: bob.id, action: "like" } });
  assert(repeat.status === 409, `repeated like must conflict, got ${repeat.status}`);

  const second = await request("/likes", { method: "POST", token: bob.token, json: { userId: alice.id, action: "like" } });
  assert(second.status === 200 && second.payload.matched === true, "reciprocal like must match");
  const conversationId = second.payload.match.conversationId;

  const sent = await request(`/conversations/${conversationId}/messages`, {
    method: "POST",
    token: alice.token,
    json: { body: "Bonjour Bob !" }
  });
  assert(sent.status === 201, `send failed: ${sent.status}`);

  const inbox = await request("/conversations", { token: bob.token });
  assert(inbox.payload.conversations[0]?.unreadCount === 1, "bob should have one unread message");
  const read = await request(`/conversations/${conversationId}/read`, { method: "POST", token: bob.token });
  assert(read.status === 200, "mark read failed");

  const notifications = await request("/notifications", { token: bob.token });
  assert(
    notifications.payload.notifications.some((n) => n.type === "match"),
    "bob should have a match notification"
  );

  const intruder = await createMember({ prefix: "smoke_c", name: "Chloe", gender: "woman", interestedIn: ["man"] });
  const forbidden = await request(`/conversations/${conversationId}/messages`, { token: intruder.token });
  assert(forbidden.status === 404, `outsider must not read the conversation, got ${forbidden.status}`);

  console.log("[e2e-smoke] PASS");
}

main().catch((error) => {
  console.error(`[e2e-smoke] FAIL ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
