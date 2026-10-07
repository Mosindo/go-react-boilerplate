/* eslint-disable no-console */
// Black-box journey against a running API:
// register -> profile -> photo -> discover -> like -> match -> chat -> read -> notifications.
// Usage: MOBILE_E2E_API_URL=http://localhost:18080 node scripts/e2e-smoke.js
const API = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Sup3rSecret!";
const crypto = require("node:crypto");
const zlib = require("node:zlib");

function assert(cond, message) {
  if (!cond) throw new Error(message);
}

async function call(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${path}`, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = null;
  }
  return { status: res.status, json };
}

function expect(res, status, label) {
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.json)}`);
  return res.json;
}

// Minimal valid PNG (solid colour) so the upload exercises the real image pipeline.
function png(w, h, rgb) {
  const crcTable = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (buf) => {
    let c = 0xffffffff;
    for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const td = Buffer.concat([Buffer.from(type), data]);
    const c = Buffer.alloc(4);
    c.writeUInt32BE(crc(td));
    return Buffer.concat([len, td, c]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  const row = Buffer.concat([Buffer.from([0]), Buffer.from(Array.from({ length: w }, () => rgb).flat())]);
  const raw = Buffer.concat(Array.from({ length: h }, () => row));
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function onboard(name, gender, lat, lng) {
  const email = `smoke_${name}_${Date.now()}_${crypto.randomInt(1e6)}@smoke.invalid`;
  const auth = expect(await call("POST", "/auth/register", { body: { email, password: PASSWORD } }), 201, "register");
  const token = auth.accessToken;
  expect(
    await call("PUT", "/profile/me", {
      token,
      body: { firstName: name, birthDate: "1993-04-12", gender, bio: "Smoke test", city: "Testville", interests: ["hiking"] }
    }),
    200,
    "profile"
  );
  expect(await call("PUT", "/profile/me/location", { token, body: { latitude: lat, longitude: lng } }), 204, "location");
  const form = new FormData();
  form.append("file", new Blob([png(300, 300, [200, 90, 60])], { type: "image/png" }), "me.png");
  expect(await call("POST", "/photos", { token, form }), 201, "photo");
  return { id: auth.user.id, token };
}

async function main() {
  console.log(`[e2e-smoke] API=${API}`);
  expect(await call("GET", "/health"), 200, "health");

  // Isolated spot so leftover accounts never interfere.
  const lat = -40 + Math.random() * 80;
  const lng = -170 + Math.random() * 340;
  const ana = await onboard("Ana", "woman", lat, lng);
  const ben = await onboard("Ben", "man", lat + 0.01, lng + 0.01);

  const me = expect(await call("GET", "/me", { token: ana.token }), 200, "me");
  assert(me.profileComplete === true, "profile should be complete");

  const feed = expect(await call("GET", "/discover?limit=10", { token: ana.token }), 200, "discover");
  assert(feed.items.some((p) => p.userId === ben.id), "Ben should appear in Ana's feed");

  const first = expect(await call("POST", "/swipes", { token: ana.token, body: { targetUserId: ben.id, action: "like" } }), 200, "like");
  assert(first.matched === false, "one-sided like must not match");
  expect(await call("POST", "/swipes", { token: ana.token, body: { targetUserId: ben.id, action: "like" } }), 409, "duplicate like");

  const second = expect(await call("POST", "/swipes", { token: ben.token, body: { targetUserId: ana.id, action: "like" } }), 200, "like back");
  assert(second.matched === true && second.match.conversationId, "reciprocal like must match");
  const conversationId = second.match.conversationId;

  expect(await call("POST", `/conversations/${conversationId}/messages`, { token: ana.token, body: { content: "Hello Ben" } }), 201, "send");
  const convs = expect(await call("GET", "/conversations", { token: ben.token }), 200, "conversations");
  assert(convs.items[0].unreadCount === 1, "Ben should have one unread message");
  expect(await call("POST", `/conversations/${conversationId}/read`, { token: ben.token }), 204, "read");
  const history = expect(await call("GET", `/conversations/${conversationId}/messages`, { token: ana.token }), 200, "history");
  assert(history.items[0].content === "Hello Ben", "history should contain the message");

  const notes = expect(await call("GET", "/notifications", { token: ben.token }), 200, "notifications");
  assert(notes.unreadCount >= 1, "Ben should have notifications");

  for (const who of [ana, ben]) {
    expect(await call("DELETE", "/me", { token: who.token, body: { password: PASSWORD } }), 204, "cleanup");
  }
  console.log("[e2e-smoke] PASS");
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err.message}`);
  process.exit(1);
});
