/* eslint-disable no-console */
// End-to-end smoke test against a running API (docker compose up):
// register -> profile -> photo -> discover -> like -> match -> chat -> notifications.
// Usage: MOBILE_E2E_API_URL=http://<LAN_IP>:18080 npm run e2e:smoke
const API = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = "Password123";

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

async function call(path, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${path}`, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  return { status: res.status, json: text ? JSON.parse(text) : null };
}

// A tiny valid 256x256 PNG (generated in memory, no asset needed).
function png() {
  const zlib = require("zlib");
  const size = 256;
  const raw = Buffer.alloc((size * 3 + 1) * size);
  for (let y = 0; y < size; y++) {
    raw[y * (size * 3 + 1)] = 0;
    for (let x = 0; x < size; x++) {
      const o = y * (size * 3 + 1) + 1 + x * 3;
      raw[o] = x;
      raw[o + 1] = y;
      raw[o + 2] = 128;
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
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function onboard(name, gender, interestedIn, lat, lng) {
  const email = `smoke_${name.toLowerCase()}_${Date.now()}@smoke.tandem.local`;
  const reg = await call("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  assert(reg.status === 201, `register ${name}: ${reg.status}`);
  const token = reg.json.accessToken;
  const birth = new Date(Date.now() - 28 * 365.25 * 86400000).toISOString().slice(0, 10);
  let r = await call("/me/profile", {
    method: "PUT",
    token,
    body: { firstName: name, birthDate: birth, gender, bio: "smoke", city: "Smoke", latitude: lat, longitude: lng, interestIds: [1, 2] }
  });
  assert(r.status === 200, `profile ${name}: ${r.status}`);
  r = await call("/me/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 60, maxDistanceKm: 25 } });
  assert(r.status === 200, `preferences ${name}: ${r.status}`);
  const form = new FormData();
  form.append("file", new Blob([png()], { type: "image/png" }), "p.png");
  r = await call("/me/photos", { method: "POST", token, form });
  assert(r.status === 201, `photo ${name}: ${r.status}`);
  r = await call("/me/profile", { token });
  assert(r.json.complete === true, `profile of ${name} should be complete`);
  return { token, id: reg.json.user.id };
}

async function main() {
  console.log(`[smoke] API ${API}`);
  const health = await call("/health");
  assert(health.status === 200, `health: ${health.status}`);

  // Isolated spot so the run never sees other users of the database.
  const lat = -40 + Math.random() * 60;
  const lng = -120 + Math.random() * 200;
  const a = await onboard("Alice", "woman", ["man"], lat, lng);
  const b = await onboard("Bob", "man", ["woman"], lat + 0.02, lng);
  console.log("[smoke] register + profile + photo: ok");

  const deck = await call("/discover", { token: a.token });
  assert(
    deck.json.items.some((c) => c.userId === b.id),
    "Bob must be in Alice's deck"
  );
  console.log("[smoke] discovery: ok");

  let r = await call(`/discover/${b.id}/like`, { method: "POST", token: a.token });
  assert(r.json.matched === false, "first like must not match");
  r = await call(`/discover/${a.id}/like`, { method: "POST", token: b.token });
  assert(r.json.matched === true && r.json.matchId, "reciprocal like must match");
  const matchId = r.json.matchId;
  console.log("[smoke] like + match: ok");

  r = await call(`/matches/${matchId}/messages`, { method: "POST", token: a.token, body: { body: "Salut Bob !" } });
  assert(r.status === 201, `send: ${r.status}`);
  r = await call("/matches", { token: b.token });
  assert(r.json.items[0].unreadCount === 1 && r.json.items[0].lastMessage.body === "Salut Bob !", "Bob must see 1 unread message");
  r = await call(`/matches/${matchId}/read`, { method: "POST", token: b.token });
  assert(r.status === 204, `read: ${r.status}`);
  r = await call(`/matches/${matchId}/messages`, { token: a.token });
  assert(r.json.items[0].readAt, "Alice must see the read receipt");
  console.log("[smoke] chat send/read: ok");

  r = await call("/notifications", { token: b.token });
  assert(
    r.json.items.some((n) => n.type === "match"),
    "Bob must have a match notification"
  );
  r = await call("/notifications/read-all", { method: "POST", token: b.token });
  assert(r.status === 204, "mark all read");
  console.log("[smoke] notifications: ok");

  for (const u of [a, b]) {
    const del = await call("/me", { method: "DELETE", token: u.token, body: { password: PASSWORD } });
    assert(del.status === 204, `cleanup delete: ${del.status}`);
  }
  console.log("[smoke] account deletion cleanup: ok\n[smoke] ALL GOOD");
}

main().catch((error) => {
  console.error(`[smoke] FAILED: ${error.message}`);
  process.exit(1);
});
