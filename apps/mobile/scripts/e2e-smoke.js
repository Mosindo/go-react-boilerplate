/**
 * API smoke test of the critical dating journey, runnable against any running
 * API (local, Docker or staging):
 *   register -> profile -> photo -> discovery -> like -> match -> chat -> read
 *   -> notifications -> block -> delete account
 *
 * Usage: MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
 */
const zlib = require("zlib");

const API = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Password123";

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function request(path, { method = "GET", body, token, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) {
    payload = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(`${API}${path}`, { method, headers, body: payload });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = null;
  }
  return { status: res.status, json };
}

// Minimal valid 320x400 PNG generated in memory (no binary fixture needed).
function makePng(seed) {
  const width = 320;
  const height = 400;
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    raw[row] = 0;
    for (let x = 0; x < width; x++) {
      raw[row + 1 + x * 3] = (x + seed) & 0xff;
      raw[row + 2 + x * 3] = y & 0xff;
      raw[row + 3 + x * 3] = 140;
    }
  }
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
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function member(prefix, profile) {
  const email = `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@smoke.lueur.invalid`;
  const reg = await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  assert(reg.status === 201, `register ${prefix}: ${reg.status} ${JSON.stringify(reg.json)}`);
  const token = reg.json.accessToken;
  const me = await request("/me", { token });
  assert(me.status === 200, `/me ${prefix}: ${me.status}`);
  const up = await request("/me/profile", { method: "PUT", token, body: profile });
  assert(up.status === 200, `profile ${prefix}: ${up.status} ${JSON.stringify(up.json)}`);
  const prefs = await request("/me/preferences", {
    method: "PUT",
    token,
    body: { interestedIn: profile.interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 100 }
  });
  assert(prefs.status === 200, `preferences ${prefix}: ${prefs.status}`);
  const loc = await request("/me/location", { method: "PUT", token, body: { latitude: 48.85 + Math.random() / 100, longitude: 2.35 } });
  assert(loc.status === 204, `location ${prefix}: ${loc.status}`);
  const form = new FormData();
  form.append("photo", new Blob([makePng(prefix.length * 20)], { type: "image/png" }), "photo.png");
  const photo = await request("/me/photos", { method: "POST", token, form });
  assert(photo.status === 201, `photo ${prefix}: ${photo.status} ${JSON.stringify(photo.json)}`);
  return { email, token, id: reg.json.user.id };
}

async function main() {
  const health = await request("/health");
  assert(health.status === 200, `health: ${health.status}`);

  const a = await member("smoke_a", {
    firstName: "Alice",
    birthdate: "1994-02-03",
    gender: "woman",
    bio: "Smoke test",
    city: "Paris",
    jobTitle: "",
    interestedIn: ["man"]
  });
  const b = await member("smoke_b", {
    firstName: "Bruno",
    birthdate: "1991-07-08",
    gender: "man",
    bio: "Smoke test",
    city: "Paris",
    jobTitle: "",
    interestedIn: ["woman"]
  });

  const feed = await request(`/discovery?limit=20`, { token: a.token });
  assert(feed.status === 200, `discovery: ${feed.status}`);
  assert(
    feed.json.profiles.some((p) => p.userId === b.id),
    "alice should discover bruno"
  );

  const like1 = await request("/swipes", { method: "POST", token: a.token, body: { targetUserId: b.id, action: "like" } });
  assert(like1.status === 201 && like1.json.matched === false, `first like: ${like1.status}`);
  const like2 = await request("/swipes", { method: "POST", token: b.token, body: { targetUserId: a.id, action: "like" } });
  assert(like2.status === 201 && like2.json.matched === true, `mutual like should match: ${JSON.stringify(like2.json)}`);

  const send = await request(`/chats/${b.id}/messages`, { method: "POST", token: a.token, body: { content: "Bonjour !" } });
  assert(send.status === 201, `send: ${send.status}`);
  const chats = await request("/chats", { token: b.token });
  assert(chats.status === 200 && chats.json.chats[0]?.unreadCount === 1, "bruno should have one unread message");
  const read = await request(`/chats/${a.id}/read`, { method: "POST", token: b.token });
  assert(read.status === 200, `read: ${read.status}`);
  const msgs = await request(`/chats/${b.id}/messages`, { token: a.token });
  assert(msgs.json.messages[0]?.isRead === true, "message should be read");

  const notifs = await request("/notifications", { token: b.token });
  assert(notifs.status === 200 && notifs.json.notifications.some((n) => n.type === "match.new"), "match notification expected");

  const block = await request("/blocks", { method: "POST", token: b.token, body: { userId: a.id } });
  assert(block.status === 204, `block: ${block.status}`);
  const afterBlock = await request(`/chats/${b.id}/messages`, { token: a.token });
  assert(afterBlock.status === 404, "conversation must be gone after block");

  for (const m of [a, b]) {
    const del = await request("/me", { method: "DELETE", token: m.token, body: { password: PASSWORD } });
    assert(del.status === 204, `delete: ${del.status}`);
  }
  console.log("[e2e-smoke] PASS register/profile/photo/discovery/like/match/chat/read/notifications/block/delete");
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err.message}`);
  process.exit(1);
});
