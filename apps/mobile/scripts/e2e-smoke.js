// API smoke test of the critical path against a running API:
// register → profile → discovery → like → match → chat → notifications → block → delete.
const { API_URL, call, expect, registerMember, onboardMember } = require("../e2e/fixtures");

async function main() {
  console.log(`[smoke] API ${API_URL}`);
  expect(await call("GET", "/health"), 200, "health");

  const lat = 45.76 + Math.random() * 0.5;
  const lng = 4.83 + Math.random() * 0.5;
  const a = await registerMember("smoke_a");
  const b = await registerMember("smoke_b");
  await onboardMember(a, { firstName: "Ana", gender: "woman", interestedIn: ["man"], lat, lng });
  await onboardMember(b, { firstName: "Bruno", gender: "man", interestedIn: ["woman"], lat: lat + 0.01, lng });

  const feed = expect(await call("GET", "/discover?limit=25", { token: a.token }), 200, "discover");
  if (!feed.profiles.some((p) => p.userId === b.id)) throw new Error("A should discover B");

  expect(
    await call("POST", "/swipes", { token: a.token, body: { targetUserId: b.id, direction: "like" } }),
    200,
    "A likes B"
  );
  const match = expect(
    await call("POST", "/swipes", { token: b.token, body: { targetUserId: a.id, direction: "like" } }),
    201,
    "B likes A"
  );
  if (!match.matched) throw new Error("expected a match");
  const matchId = match.match.matchId;

  expect(
    await call("POST", `/conversations/${matchId}/messages`, { token: a.token, body: { body: "Salut Bruno !" } }),
    201,
    "send"
  );
  const page = expect(
    await call("GET", `/conversations/${matchId}/messages`, { token: b.token }),
    200,
    "read messages"
  );
  if (page.messages.at(-1)?.body !== "Salut Bruno !") throw new Error("B should read A's message");
  expect(await call("POST", `/conversations/${matchId}/read`, { token: b.token }), 204, "mark read");

  const notifs = expect(await call("GET", "/notifications", { token: b.token }), 200, "notifications");
  if (!notifs.notifications.some((n) => n.type === "match")) throw new Error("B should have a match notification");
  expect(await call("POST", "/notifications/read-all", { token: b.token }), 204, "read all");

  expect(await call("POST", "/blocks", { token: a.token, body: { userId: b.id } }), 204, "block");
  expect(
    await call("GET", `/conversations/${matchId}/messages`, { token: b.token }),
    404,
    "conversation gone after block"
  );

  expect(await call("DELETE", "/me", { token: a.token, body: { password: a.password } }), 204, "delete A");
  expect(await call("DELETE", "/me", { token: b.token, body: { password: b.password } }), 204, "delete B");
  expect(await call("GET", "/me", { token: a.token }), 401, "token after deletion");

  console.log("[smoke] PASS register/profile/photo/discover/like/match/chat/read/notifications/block/delete");
}

main().catch((error) => {
  console.error(`[smoke] FAIL ${error.message}`);
  process.exit(1);
});
