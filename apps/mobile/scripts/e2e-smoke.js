/* eslint-disable no-console */
// End-to-end smoke test against a running API:
// register x2 -> profiles -> preferences -> photos -> discover -> mutual like -> match -> chat -> read -> cleanup.
const zlib = require("node:zlib");

const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = "Password123!";

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function crc32(buf) {
  let crc = ~0;
  for (const byte of buf) {
    crc ^= byte;
    for (let i = 0; i < 8; i += 1) {
      crc = (crc >>> 1) ^ (0xedb88320 & -(crc & 1));
    }
  }
  return ~crc >>> 0;
}

function chunk(type, data) {
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([length, body, crc]);
}

/** Builds a small valid solid-colour PNG without any dependency. */
function makePng(size, [r, g, b]) {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const row = Buffer.concat([Buffer.from([0]), Buffer.from(Array.from({ length: size }, () => [r, g, b]).flat())]);
  const raw = Buffer.concat(Array.from({ length: size }, () => row));
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function request(path, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    body: form ?? (body !== undefined ? JSON.stringify(body) : undefined)
  });
  const text = await response.text();
  return { status: response.status, payload: text ? JSON.parse(text) : null };
}

async function expectStatus(label, promise, status) {
  const res = await promise;
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.payload)}`);
  console.log(`ok  ${label}`);
  return res.payload;
}

async function createUser(label, { firstName, gender, interestedIn, color }) {
  const email = `${label}_${Date.now()}_${Math.floor(Math.random() * 100000)}@lumen.test`;
  const session = await expectStatus(`register ${label}`, request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }), 201);
  const token = session.accessToken;
  const interests = await expectStatus("interests", request("/interests", { token }), 200);
  await expectStatus(
    `profile ${label}`,
    request("/profile", {
      method: "PUT",
      token,
      body: {
        firstName,
        birthDate: "1994-04-12",
        gender,
        bio: "Smoke test profile",
        city: "Paris",
        interestIds: interests.interests.slice(0, 2).map((i) => i.id),
        showDistance: true,
        discoverable: true
      }
    }),
    200
  );
  await expectStatus(`location ${label}`, request("/location", { method: "PUT", token, body: { latitude: 48.8566, longitude: 2.3522, city: "Paris" } }), 200);
  await expectStatus(
    `preferences ${label}`,
    request("/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 60, maxDistanceKm: 100 } }),
    200
  );
  const form = new FormData();
  form.append("photo", new Blob([makePng(256, color)], { type: "image/png" }), "photo.png");
  await expectStatus(`photo ${label}`, request("/photos", { method: "POST", token, form }), 201);
  const me = await expectStatus(`me ${label}`, request("/me", { token }), 200);
  assert(me.profileComplete === true, `${label} profile should be complete`);
  return { email, token, id: me.id, refreshToken: session.refreshToken };
}

async function main() {
  const health = await request("/health");
  assert(health.status === 200, `API not healthy at ${API_BASE_URL}`);

  const alice = await createUser("alice", { firstName: "Alice", gender: "woman", interestedIn: ["man"], color: [200, 80, 60] });
  const bob = await createUser("bob", { firstName: "Bob", gender: "man", interestedIn: ["woman"], color: [60, 80, 200] });

  try {
    const queue = await expectStatus("discover alice", request("/discover?limit=50", { token: alice.token }), 200);
    assert(queue.profiles.some((p) => p.userId === bob.id), "Bob should appear in Alice's discovery");
    assert(!queue.profiles.some((p) => p.userId === alice.id), "Alice must never see herself");

    const first = await expectStatus("alice likes bob", request("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.id, action: "like" } }), 200);
    assert(first.matched === false, "One-sided like must not match");
    const repeat = await expectStatus("alice likes bob again (idempotent)", request("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.id, action: "like" } }), 200);
    assert(repeat.matched === false, "Repeated like must stay unmatched");

    const second = await expectStatus("bob likes alice", request("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.id, action: "like" } }), 200);
    assert(second.matched === true && second.conversation, "Mutual like must match");
    const conversationId = second.conversation.id;

    await expectStatus("bob sends message", request(`/conversations/${conversationId}/messages`, { method: "POST", token: bob.token, body: { body: "Hello Alice" } }), 201);
    const convs = await expectStatus("alice conversations", request("/conversations", { token: alice.token }), 200);
    const conv = convs.conversations.find((c) => c.id === conversationId);
    assert(conv && conv.unreadCount === 1, "Alice should see 1 unread message");

    await expectStatus("alice reads", request(`/conversations/${conversationId}/read`, { method: "POST", token: alice.token }), 204);
    const messages = await expectStatus("bob messages", request(`/conversations/${conversationId}/messages`, { token: bob.token }), 200);
    assert(messages.messages[0] && messages.messages[0].readAt, "Message should be marked read");

    const notifications = await expectStatus("alice notifications", request("/notifications", { token: alice.token }), 200);
    assert(notifications.unreadCount >= 1, "Alice should have a match notification");

    await expectStatus("outsider cannot read", request(`/conversations/${conversationId}/messages`, { token: (await createUserMinimal()).token }), 404);
    console.log("\nSmoke test passed.");
  } finally {
    await request("/me", { method: "DELETE", token: alice.token, body: { password: PASSWORD } });
    await request("/me", { method: "DELETE", token: bob.token, body: { password: PASSWORD } });
  }
}

async function createUserMinimal() {
  const email = `outsider_${Date.now()}@lumen.test`;
  const session = await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  return { token: session.payload.accessToken };
}

main().catch((error) => {
  console.error(`\nSmoke test failed: ${error.message}`);
  process.exit(1);
});
