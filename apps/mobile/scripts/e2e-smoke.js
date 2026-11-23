/**
 * End-to-end API smoke test of the critical flow against a running API:
 * register → onboarding (profile, preferences, location, photo) → discovery →
 * like → match → realtime message → read receipt → notifications → cleanup.
 *
 *   MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
 */
const { png } = require("./lib/png");

const API = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "SmokeTest123";

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function call(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${path}`, { method, headers, body: form ?? (body !== undefined ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = text;
  }
  return { status: res.status, json };
}

async function expect(status, promise, label) {
  const res = await promise;
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.json)}`);
  return res.json;
}

async function createMember(label, persona, lat, lon) {
  const email = `smoke_${label}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@hearth.test`;
  const auth = await expect(201, call("POST", "/auth/register", { body: { email, password: PASSWORD } }), `register ${label}`);
  const token = auth.accessToken;
  await expect(200, call("PATCH", "/profile", { token, body: persona.profile }), `profile ${label}`);
  await expect(200, call("PUT", "/profile/preferences", { token, body: persona.preferences }), `preferences ${label}`);
  await expect(
    200,
    call("PUT", "/profile/location", { token, body: { latitude: lat, longitude: lon, city: "Smokeville" } }),
    `location ${label}`
  );
  const form = new FormData();
  form.append("photo", new Blob([png(320, 400, persona.color)], { type: "image/png" }), "photo.png");
  await expect(201, call("POST", "/profile/photos", { token, form }), `photo ${label}`);
  const me = await expect(200, call("GET", "/profile", { token }), `get profile ${label}`);
  assert(me.isComplete, `${label} profile should be complete: missing ${me.missing}`);
  return { email, token, id: auth.user.id };
}

function openRealtime(token) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`${API.replace(/^http/, "ws")}/realtime`);
    const events = [];
    const waiters = [];
    ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
    ws.onmessage = (msg) => {
      const event = JSON.parse(msg.data);
      events.push(event);
      waiters.splice(0).forEach((w) => w());
      if (event.type === "ready") resolve(api);
    };
    ws.onerror = () => reject(new Error("websocket error"));
    const api = {
      close: () => ws.close(),
      async waitFor(type, predicate = () => true, timeoutMs = 5000) {
        const deadline = Date.now() + timeoutMs;
        while (Date.now() < deadline) {
          const found = events.find((e) => e.type === type && predicate(e.data));
          if (found) return found;
          await new Promise((r) => {
            waiters.push(r);
            setTimeout(r, 200);
          });
        }
        throw new Error(`timed out waiting for realtime ${type}`);
      }
    };
  });
}

async function main() {
  console.log(`[smoke] API ${API}`);
  await expect(200, call("GET", "/health"), "health");

  const lat = -40 + Math.random() * 20;
  const lon = -150 + Math.random() * 20;
  const alice = await createMember(
    "alice",
    {
      profile: { firstName: "Alice", birthdate: "1994-05-12", gender: "woman", bio: "Smoke test", interests: ["hiking", "coffee"] },
      preferences: { interestedIn: ["man"], minAge: 18, maxAge: 99, maxDistanceKm: 50 },
      color: [220, 120, 90]
    },
    lat,
    lon
  );
  const bruno = await createMember(
    "bruno",
    {
      profile: { firstName: "Bruno", birthdate: "1992-03-02", gender: "man", bio: "Smoke test", interests: ["hiking"] },
      preferences: { interestedIn: ["woman"], minAge: 18, maxAge: 99, maxDistanceKm: 50 },
      color: [90, 120, 220]
    },
    lat + 0.02,
    lon
  );
  console.log("[smoke] onboarding OK");

  try {
    const deck = await expect(200, call("GET", "/discover?limit=30", { token: alice.token }), "discover");
    assert(
      deck.profiles.some((p) => p.userId === bruno.id),
      "Bruno should appear in Alice's deck"
    );
    const card = deck.profiles.find((p) => p.userId === bruno.id);
    assert(card.distanceKm !== null && card.photos.length === 1, "card should show approximate distance and photo");
    const media = await fetch(`${API}${card.photos[0].url}`);
    assert(media.status === 200 && media.headers.get("content-type") === "image/jpeg", "signed photo URL should serve a JPEG");
    console.log("[smoke] discovery OK");

    const first = await expect(
      200,
      call("POST", "/swipes", { token: alice.token, body: { targetUserId: bruno.id, action: "like" } }),
      "like 1"
    );
    assert(!first.matched, "one-sided like must not match");
    const dup = await call("POST", "/swipes", { token: alice.token, body: { targetUserId: bruno.id, action: "like" } });
    assert(dup.status === 409, "duplicate like must be rejected");
    const second = await expect(
      200,
      call("POST", "/swipes", { token: bruno.token, body: { targetUserId: alice.id, action: "like" } }),
      "like 2"
    );
    assert(second.matched && second.match.conversationId, "mutual like must match");
    const conversationId = second.match.conversationId;
    console.log("[smoke] match OK");

    const realtime = await openRealtime(bruno.token);
    await expect(
      201,
      call("POST", `/conversations/${conversationId}/messages`, { token: alice.token, body: { content: "Salut Bruno !" } }),
      "send"
    );
    await realtime.waitFor("message.created", (m) => m.content === "Salut Bruno !");
    realtime.close();
    const page = await expect(200, call("GET", `/conversations/${conversationId}/messages`, { token: bruno.token }), "messages");
    assert(page.messages[0].content === "Salut Bruno !", "message must be listed");
    await expect(200, call("POST", `/conversations/${conversationId}/read`, { token: bruno.token }), "read");
    const forAlice = await expect(200, call("GET", `/conversations/${conversationId}/messages`, { token: alice.token }), "messages alice");
    assert(forAlice.otherLastReadAt, "read receipt must be visible to the sender");
    console.log("[smoke] chat + realtime OK");

    const notifications = await expect(200, call("GET", "/notifications", { token: bruno.token }), "notifications");
    assert(
      notifications.notifications.some((n) => n.type === "match"),
      "match notification expected"
    );
    console.log("[smoke] notifications OK");
  } finally {
    for (const m of [alice, bruno]) {
      await call("DELETE", "/me", { token: m.token, body: { password: PASSWORD } });
    }
  }
  console.log("[smoke] cleanup OK — all checks passed");
}

main().catch((error) => {
  console.error(`[smoke] FAILED: ${error.message}`);
  process.exit(1);
});
