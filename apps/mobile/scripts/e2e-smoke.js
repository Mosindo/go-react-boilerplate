/* API smoke test of the critical journey, no browser needed.
 * register x2 → profile → photo → location → discovery → like/like → match
 * → message → read → notifications → block → account deletion.
 *
 * Usage: E2E_API_URL=http://localhost:18080 node scripts/e2e-smoke.js
 */
const zlib = require("zlib");
const { Buffer } = require("buffer");

const API_URL = (process.env.E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Smoke2026pass";
const run = `${Date.now()}_${Math.floor(Math.random() * 1e5)}`;
const spot = { latitude: -20 - Math.random(), longitude: 55 + Math.random() }; // far from demo data

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function api(path, { method = "GET", token, body, form, expect } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_URL}${path}`, { method, headers, body: form || (body !== undefined ? JSON.stringify(body) : undefined) });
  const text = await response.text();
  const payload = text ? JSON.parse(text) : null;
  const wanted = expect || [200, 201, 202, 204];
  assert(wanted.includes(response.status), `${method} ${path}: expected ${wanted.join("/")} got ${response.status} ${text}`);
  return payload;
}

// Minimal RGB PNG encoder (solid colour), enough for the upload pipeline.
function png(width, height, [r, g, b]) {
  const crcTable = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k += 1) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (buf) => {
    let c = 0xffffffff;
    for (const byte of buf) c = crcTable[(c ^ byte) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type), data]);
    const sum = Buffer.alloc(4);
    sum.writeUInt32BE(crc(body));
    return Buffer.concat([len, body, sum]);
  };
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8;
  header[9] = 2;
  const row = Buffer.concat([Buffer.from([0]), Buffer.from(Array.from({ length: width }, () => [r, g, b]).flat())]);
  const raw = Buffer.concat(Array.from({ length: height }, () => row));
  return Buffer.concat([Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]), chunk("IHDR", header), chunk("IDAT", zlib.deflateSync(raw)), chunk("IEND", Buffer.alloc(0))]);
}

async function member(name, gender, interestedIn, color) {
  const email = `smoke_${name.toLowerCase()}_${run}@lueur.test`;
  const auth = await api("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  const token = auth.accessToken;
  await api("/me/profile", { method: "PUT", token, body: { firstName: name, birthdate: "1992-09-14", gender, bio: "Smoke test", relationshipGoal: "unsure", city: "" } });
  await api("/me/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 30 } });
  await api("/me/location", { method: "PUT", token, body: spot });
  const form = new FormData();
  form.append("photo", new Blob([png(320, 400, color)], { type: "image/png" }), "photo.png");
  await api("/me/photos", { method: "POST", token, form });
  const mine = await api("/me/profile", { token });
  assert(mine.isComplete, `${name} profile should be complete: ${JSON.stringify(mine.missing)}`);
  return { name, email, token, id: auth.user.id };
}

async function main() {
  console.log(`[smoke] API ${API_URL}`);
  const health = await api("/health");
  assert(health.status === "ok", "health must be ok");
  await api("/discovery", { expect: [401] });

  const ana = await member("Ana", "woman", ["man"], [230, 120, 90]);
  const ben = await member("Ben", "man", ["woman"], [90, 120, 230]);
  console.log("[smoke] members onboarded");

  const anaDeck = await api("/discovery?limit=30", { token: ana.token });
  assert(anaDeck.profiles.some((p) => p.userId === ben.id), "Ana should discover Ben");
  const benCard = anaDeck.profiles.find((p) => p.userId === ben.id);
  assert(!("email" in benCard) && !("latitude" in benCard), "cards must not leak private data");

  const first = await api("/swipes", { method: "POST", token: ana.token, body: { targetUserId: ben.id, decision: "like" } });
  assert(!first.matched, "one-sided like must not match");
  await api("/swipes", { method: "POST", token: ana.token, body: { targetUserId: ben.id, decision: "like" }, expect: [409] });
  const second = await api("/swipes", { method: "POST", token: ben.token, body: { targetUserId: ana.id, decision: "like" } });
  assert(second.matched && second.match.conversationId, "mutual like must create a match");
  const conversationId = second.match.conversationId;
  console.log("[smoke] match created");

  await api(`/conversations/${conversationId}/messages`, { method: "POST", token: ana.token, body: { body: "Bonjour Ben !" } });
  const benConversations = await api("/conversations", { token: ben.token });
  assert(benConversations.conversations[0].unreadCount === 1, "Ben should have one unread message");
  await api(`/conversations/${conversationId}/read`, { method: "POST", token: ben.token });
  const anaMessages = await api(`/conversations/${conversationId}/messages`, { token: ana.token });
  assert(anaMessages.messages[0].isRead, "Ana's message should be read");
  const notifications = await api("/notifications", { token: ben.token });
  assert(notifications.notifications.some((n) => n.type === "match"), "Ben should have a match notification");
  console.log("[smoke] chat and notifications ok");

  await api("/blocks", { method: "POST", token: ben.token, body: { userId: ana.id } });
  await api(`/conversations/${conversationId}/messages`, { token: ana.token, expect: [404] });
  console.log("[smoke] block ok");

  for (const m of [ana, ben]) {
    await api("/me", { method: "DELETE", token: m.token, body: { password: PASSWORD } });
    await api("/auth/login", { method: "POST", body: { email: m.email, password: PASSWORD }, expect: [401] });
  }
  console.log("[smoke] account deletion ok");
  console.log("[smoke] PASS");
}

main().catch((error) => {
  console.error(`[smoke] FAIL: ${error.message}`);
  process.exit(1);
});
