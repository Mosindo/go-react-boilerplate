// API smoke test against a running server: register -> profile -> discover -> like -> match -> chat -> notifications.
// Usage: E2E_API_URL=http://localhost:18080 npm run e2e:api
const { API_URL, assert, request, createCompleteUser } = require("./lib-e2e");

async function main() {
  console.log(`API smoke test against ${API_URL}`);
  const health = await request("GET", "/health");
  assert(health.status === 200, `health: ${health.status}`);

  // Remote, isolated area so the test never meets real users.
  const area = { latitude: -43.53, longitude: 172.64 };
  const alice = await createCompleteUser({
    firstName: "Alice",
    gender: "woman",
    interestedIn: ["man"],
    ...area
  });
  const bob = await createCompleteUser({ firstName: "Bob", gender: "man", interestedIn: ["woman"], ...area });
  console.log("✓ register + complete profile (x2)");

  const feed = await request("GET", "/discover?limit=10", { token: alice.token });
  assert(
    feed.status === 200 && feed.json.profiles.some((p) => p.userId === bob.id),
    "Bob should appear in Alice's discovery"
  );
  assert(!JSON.stringify(feed.json).includes("latitude"), "discovery must not leak coordinates");
  console.log("✓ discovery");

  let swipe = await request("POST", "/swipes", {
    token: alice.token,
    body: { userId: bob.id, action: "like" }
  });
  assert(swipe.status === 200 && swipe.json.matched === false, "first like must not match");
  swipe = await request("POST", "/swipes", { token: bob.token, body: { userId: alice.id, action: "like" } });
  assert(swipe.status === 200 && swipe.json.matched === true, "mutual like must match");
  const conversationId = swipe.json.match.conversationId;
  console.log("✓ like -> match");

  const sent = await request("POST", `/conversations/${conversationId}/messages`, {
    token: alice.token,
    body: { body: "Salut Bob !" }
  });
  assert(sent.status === 201, `send message: ${sent.status}`);
  const history = await request("GET", `/conversations/${conversationId}/messages`, { token: bob.token });
  assert(
    history.status === 200 && history.json.messages[0].body === "Salut Bob !",
    "Bob should read Alice's message"
  );
  const convs = await request("GET", "/conversations", { token: bob.token });
  assert(convs.json.conversations[0].unreadCount === 1, "Bob has one unread message");
  await request("POST", `/conversations/${conversationId}/read`, { token: bob.token });
  const after = await request("GET", "/conversations/unread-count", { token: bob.token });
  assert(after.json.unreadCount === 0, "unread cleared after read");
  console.log("✓ chat send / read / unread");

  const intruder = await createCompleteUser({
    firstName: "Eve",
    gender: "woman",
    interestedIn: ["man"],
    ...area
  });
  const spy = await request("GET", `/conversations/${conversationId}/messages`, { token: intruder.token });
  assert(spy.status === 404, `non-participant must not read messages (got ${spy.status})`);
  console.log("✓ chat permissions");

  const notifs = await request("GET", "/notifications", { token: bob.token });
  assert(
    notifs.status === 200 && notifs.json.notifications.some((n) => n.type === "match"),
    "match notification expected"
  );
  console.log("✓ notifications");

  const del = await request("DELETE", "/me", { token: intruder.token, body: { password: "Password123!" } });
  assert(del.status === 204, "account deletion");
  console.log("\nAPI smoke test passed.");
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
