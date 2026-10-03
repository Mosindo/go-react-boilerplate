/* eslint-disable no-console */
// Real end-to-end journey against a running API:
// register → profile → photo → discover → like → match → chat (REST + WebSocket) → block → delete account.
const { API_BASE_URL, PASSWORD, assert, createMember, randomOrigin, request } = require("./lib/api");

function openSocket(token) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`${API_BASE_URL.replace(/^http/, "ws")}/ws`);
    const events = [];
    const waiters = [];
    ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
    ws.onerror = () => reject(new Error("websocket error"));
    ws.onmessage = (raw) => {
      const event = JSON.parse(String(raw.data));
      events.push(event);
      waiters.splice(0).forEach((w) => w());
      if (event.type === "ready") {
        resolve({
          close: () => ws.close(),
          next: async (type, timeoutMs = 4000) => {
            const deadline = Date.now() + timeoutMs;
            for (;;) {
              const found = events.find((e) => e.type === type);
              if (found) return found;
              if (Date.now() > deadline) throw new Error(`timed out waiting for ${type}`);
              await new Promise((r) => {
                waiters.push(r);
                setTimeout(r, 200);
              });
            }
          }
        });
      }
    };
  });
}

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);
  const health = await request("/health");
  assert(health.status === 200, `health: ${health.status}`);

  const origin = randomOrigin();
  const ana = await createMember("ana", { name: "Ana", gender: "woman", interestedIn: ["man"], origin });
  const ben = await createMember("ben", { name: "Ben", gender: "man", interestedIn: ["woman"], origin });
  console.log("[e2e-smoke] members created");

  const feed = await request("/discover", { token: ana.token });
  assert(feed.status === 200 && feed.payload.profiles.some((p) => p.id === ben.id), "ana should discover ben");
  assert(!JSON.stringify(feed.payload).includes("latitude"), "discovery must not leak coordinates");

  const benSocket = await openSocket(ben.token);

  let res = await request("/swipes", { token: ana.token, json: { targetId: ben.id, action: "like" } });
  assert(res.status === 200 && res.payload.matched === false, "first like must not match");
  res = await request("/swipes", { token: ben.token, json: { targetId: ana.id, action: "like" } });
  assert(res.status === 200 && res.payload.matched === true && res.payload.matchId, "second like must match");
  const matchId = res.payload.matchId;
  await benSocket.next("match.new");
  console.log("[e2e-smoke] match + realtime match event");

  res = await request(`/conversations/${matchId}/messages`, { token: ana.token, json: { body: "Hello Ben" } });
  assert(res.status === 201, `send: ${res.status}`);
  const pushed = await benSocket.next("message.new");
  assert(pushed.data.body === "Hello Ben", "ben should receive the message in real time");

  res = await request("/conversations", { token: ben.token });
  assert(res.payload.conversations[0].unreadCount === 1, "ben should have 1 unread");
  res = await request(`/conversations/${matchId}/read`, { token: ben.token, method: "POST" });
  assert(res.status === 204, "mark read");
  res = await request(`/conversations/${matchId}/messages`, { token: ana.token });
  assert(res.payload.messages[0].readAt, "ana should see the read receipt");
  console.log("[e2e-smoke] chat OK");

  const mallory = await createMember("mallory", { name: "Mallory", gender: "man", interestedIn: ["woman"], origin });
  res = await request(`/conversations/${matchId}/messages`, { token: mallory.token });
  assert(res.status === 404, "strangers cannot read a conversation");

  res = await request("/blocks", { token: ana.token, json: { userId: ben.id } });
  assert(res.status === 204, "block");
  res = await request(`/conversations/${matchId}/messages`, { token: ben.token });
  assert(res.status === 404, "blocked conversation is gone");
  console.log("[e2e-smoke] permissions + block OK");

  benSocket.close();
  for (const member of [ana, ben, mallory]) {
    res = await request("/me", { token: member.token, method: "DELETE", json: { password: PASSWORD } });
    assert(res.status === 204, `delete account: ${res.status}`);
  }
  res = await request("/auth/login", { json: { email: ana.email, password: PASSWORD } });
  assert(res.status === 401, "deleted account cannot log in");
  console.log("[e2e-smoke] PASS");
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
