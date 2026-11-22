/**
 * API smoke test of the critical dating path against a running stack:
 * register -> profile -> photo -> location -> discovery -> like -> match -> chat -> read -> notifications.
 *
 *   MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
 */
const API = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Password123";

// 400x500 solid PNG generated in memory (valid image, no external asset).
function makePng() {
  const zlib = require("zlib");
  const w = 400;
  const h = 500;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) {
    raw[y * (w * 3 + 1)] = 0;
    for (let x = 0; x < w; x++) {
      const o = y * (w * 3 + 1) + 1 + x * 3;
      raw[o] = 240;
      raw[o + 1] = 169;
      raw[o + 2] = 121;
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
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function call(path, { method = "GET", body, token, form } = {}) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(`${API}${path}`, { method, headers, body: form ?? (body !== undefined ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  return { status: res.status, data };
}

function expect(res, status, what) {
  if (res.status !== status) {
    throw new Error(`${what}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`);
  }
  return res.data;
}

async function member(name, gender, interestedIn) {
  const email = `smoke_${name.toLowerCase()}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@lueur.test`;
  const auth = expect(await call("/auth/register", { method: "POST", body: { email, password: PASSWORD } }), 201, `register ${name}`);
  const token = auth.accessToken;
  expect(
    await call("/me/profile", {
      method: "PATCH",
      token,
      body: { firstName: name, birthdate: "1993-06-15", gender, bio: "Smoke test", interests: ["coffee"] }
    }),
    200,
    "profile"
  );
  expect(
    await call("/me/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 50 } }),
    200,
    "preferences"
  );
  expect(
    await call("/me/location", { method: "PUT", token, body: { latitude: 48.8566, longitude: 2.3522, city: "Paris" } }),
    200,
    "location"
  );
  const form = new FormData();
  form.append("photo", new Blob([makePng()], { type: "image/png" }), "photo.png");
  expect(await call("/me/photos", { method: "POST", token, form }), 201, "photo");
  return { id: auth.user.id, token, email };
}

async function main() {
  console.log(`[smoke] API ${API}`);
  expect(await call("/health"), 200, "health");
  const a = await member("Alix", "woman", ["man"]);
  const b = await member("Basile", "man", ["woman"]);

  const feed = expect(await call("/discovery?limit=20", { token: b.token }), 200, "discovery");
  if (!feed.profiles.some((p) => p.userId === a.id)) throw new Error("discovery did not return the compatible member");
  if (JSON.stringify(feed).includes("@")) throw new Error("discovery leaked an e-mail address");

  expect(await call("/swipes", { method: "POST", token: b.token, body: { targetUserId: a.id, action: "like" } }), 200, "like");
  const match = expect(
    await call("/swipes", { method: "POST", token: a.token, body: { targetUserId: b.id, action: "like" } }),
    200,
    "like back"
  );
  if (!match.matched) throw new Error("mutual like did not create a match");

  expect(await call(`/chats/${a.id}/messages`, { method: "POST", token: b.token, body: { content: "Salut !" } }), 201, "send");
  const chats = expect(await call("/chats", { token: a.token }), 200, "chats");
  if (chats.totalUnread < 1) throw new Error("recipient should have an unread message");
  const thread = expect(await call(`/chats/${b.id}/messages`, { token: a.token }), 200, "messages");
  if (thread.messages.length !== 1) throw new Error("thread should contain the message");
  expect(await call(`/chats/${b.id}/read`, { method: "POST", token: a.token }), 200, "read");

  const notifications = expect(await call("/notifications", { token: b.token }), 200, "notifications");
  if (!notifications.notifications.some((n) => n.type === "match.created")) throw new Error("match notification missing");
  expect(await call("/notifications/read-all", { method: "POST", token: b.token }), 200, "read all");

  for (const m of [a, b]) {
    expect(await call("/me", { method: "DELETE", token: m.token, body: { password: PASSWORD } }), 204, "delete account");
  }
  console.log("[smoke] OK — register, profile, photo, discovery, like, match, chat, read, notifications, deletion");
}

main().catch((err) => {
  console.error(`[smoke] FAILED: ${err.message}`);
  process.exit(1);
});
