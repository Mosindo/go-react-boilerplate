/* eslint-disable no-console */
// Critical-path smoke test against a running API (same journey the app performs):
// register -> profile -> photo -> discover -> like -> match -> chat -> notifications -> realtime.
//
//   MOBILE_E2E_API_URL=http://<LAN_IP>:18080 npm run e2e:smoke

const { API_BASE_URL, request, assert, expectStatus, registerAndOnboard } = require("./lib-e2e");

function waitForEvent(socket, type, timeoutMs = 5000) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`timed out waiting for websocket event "${type}"`)), timeoutMs);
    socket.addEventListener("message", function onMessage(message) {
      const event = JSON.parse(String(message.data));
      if (event.type === type) {
        clearTimeout(timer);
        socket.removeEventListener("message", onMessage);
        resolve(event);
      }
    });
  });
}

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);

  const health = await request("/health");
  expectStatus(health, 200, "health");

  const alice = await registerAndOnboard({ prefix: "smoke_a", name: "Alice", gender: "woman", interestedIn: ["man"] });
  const bob = await registerAndOnboard({ prefix: "smoke_b", name: "Bob", gender: "man", interestedIn: ["woman"] });

  const me = expectStatus(await request("/profile", { token: alice.token }), 200, "profile");
  assert(me.onboarding.complete === true, "onboarding should be complete");

  const feed = expectStatus(await request("/discover?limit=20", { token: alice.token }), 200, "discover");
  assert(
    feed.profiles.some((p) => p.userId === bob.id),
    "Alice should discover Bob"
  );
  assert(!JSON.stringify(feed).includes("latitude"), "discovery must not expose coordinates");

  // Bob listens in real time before the match happens.
  const socket = new WebSocket(`${API_BASE_URL.replace(/^http/, "ws")}/ws`);
  await new Promise((resolve, reject) => {
    socket.addEventListener("open", resolve, { once: true });
    socket.addEventListener("error", () => reject(new Error("websocket connection failed")), { once: true });
  });
  socket.send(JSON.stringify({ type: "auth", token: bob.token }));
  await waitForEvent(socket, "ready");

  const first = expectStatus(
    await request("/swipes", { method: "POST", token: alice.token, body: { userId: bob.id, action: "like" } }),
    201,
    "alice likes bob"
  );
  assert(first.matched === false, "one-sided like must not match");

  const matchEvent = waitForEvent(socket, "match.new");
  const second = expectStatus(
    await request("/swipes", { method: "POST", token: bob.token, body: { userId: alice.id, action: "like" } }),
    201,
    "bob likes alice"
  );
  assert(second.matched === true && second.matchId, "reciprocal like must match");
  await matchEvent;

  const messageEvent = waitForEvent(socket, "message.new");
  expectStatus(
    await request(`/matches/${second.matchId}/messages`, { method: "POST", token: alice.token, body: { body: "Salut Bob, e2e smoke" } }),
    201,
    "send message"
  );
  const pushed = await messageEvent;
  assert(pushed.data.body === "Salut Bob, e2e smoke", "message must arrive in real time");
  socket.close();

  const thread = expectStatus(await request(`/matches/${second.matchId}/messages`, { token: bob.token }), 200, "read thread");
  assert(thread.messages.length === 1, "thread should contain the message");
  expectStatus(await request(`/matches/${second.matchId}/read`, { method: "POST", token: bob.token }), 204, "mark read");

  const notifications = expectStatus(await request("/notifications", { token: bob.token }), 200, "notifications");
  assert(
    notifications.notifications.some((n) => n.type === "new_match"),
    "Bob should have a match notification"
  );
  expectStatus(await request("/notifications/read-all", { method: "POST", token: bob.token }), 204, "read all notifications");

  console.log("[e2e-smoke] PASS register/profile/photo/discover/like/match/chat/realtime/notifications");
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
