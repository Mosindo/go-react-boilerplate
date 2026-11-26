// API smoke test of the critical journey, run against a live API:
// register -> profile -> photo -> discover -> like -> match -> chat -> notifications.
// Usage: MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
const zlib = require("zlib");

const API = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Password123";

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function call(path, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${path}`, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  return { status: res.status, data };
}

async function expect(path, status, options) {
  const res = await call(path, options);
  assert(res.status === status, `${options?.method ?? "GET"} ${path}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`);
  return res.data;
}

/** Minimal valid PNG (solid colour) generated without dependencies. */
function png(width, height) {
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
    const body = Buffer.concat([Buffer.from(type), data]);
    const sum = Buffer.alloc(4);
    sum.writeUInt32BE(crc(body));
    return Buffer.concat([len, body, sum]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  const row = Buffer.concat([Buffer.from([0]), Buffer.alloc(width * 3, 0xc8)]);
  const raw = Buffer.concat(Array.from({ length: height }, () => row));
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

function birthdate(age) {
  const d = new Date();
  d.setFullYear(d.getFullYear() - age);
  d.setDate(d.getDate() - 10);
  return d.toISOString().slice(0, 10);
}

async function member(name, gender, interestedIn) {
  const email = `smoke_${name.toLowerCase()}_${Date.now()}_${Math.floor(Math.random() * 1e5)}@smoke.test`;
  const auth = await expect("/auth/register", 201, { method: "POST", body: { email, password: PASSWORD, acceptedTerms: true } });
  const token = auth.accessToken;
  await expect("/auth/login", 200, { method: "POST", body: { email, password: PASSWORD } });
  await expect("/profile", 200, {
    method: "PUT",
    token,
    body: { firstName: name, birthdate: birthdate(30), gender, bio: "Smoke test", jobTitle: "", city: "", interests: ["hiking"] }
  });
  await expect("/profile/preferences", 200, { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 50 } });
  await expect("/profile/location", 204, { method: "PUT", token, body: { latitude: 48.8566, longitude: 2.3522 } });
  const form = new FormData();
  form.append("photo", new Blob([png(400, 500)], { type: "image/png" }), "photo.png");
  await expect("/photos", 201, { method: "POST", token, form });
  const profile = await expect("/profile", 200, { token });
  assert(profile.completion.complete, `${name}'s profile should be complete`);
  return { token, id: auth.user.id, email };
}

async function main() {
  console.log(`[e2e-smoke] API: ${API}`);
  await expect("/health", 200);
  const a = await member("Alice", "woman", ["man"]);
  const b = await member("Bruno", "man", ["woman"]);
  const cleanup = async () => {
    for (const m of [a, b]) await call("/me", { method: "DELETE", token: m.token, body: { password: PASSWORD } });
  };

  try {
    const deck = await expect("/discover?limit=20", 200, { token: a.token });
    assert(Array.isArray(deck.profiles), "deck should list profiles");

    const first = await expect("/swipes", 201, { method: "POST", token: a.token, body: { targetUserId: b.id, action: "like" } });
    assert(!first.matched, "one-sided like must not match");
    await expect("/swipes", 409, { method: "POST", token: a.token, body: { targetUserId: b.id, action: "like" } });

    const likes = await expect("/likes/received", 200, { token: b.token });
    assert(
      likes.profiles.some((p) => p.userId === a.id),
      "B should see A's like"
    );

    const second = await expect("/swipes", 201, { method: "POST", token: b.token, body: { targetUserId: a.id, action: "like" } });
    assert(second.matched && second.match.conversationId, "mutual like must match");
    const conv = second.match.conversationId;

    await expect(`/conversations/${conv}/messages`, 201, { method: "POST", token: a.token, body: { body: "Salut !" } });
    const page = await expect(`/conversations/${conv}/messages`, 200, { token: b.token });
    assert(page.messages[0].body === "Salut !", "B should read A's message");
    await expect(`/conversations/${conv}/read`, 200, { method: "POST", token: b.token });

    const notifications = await expect("/notifications", 200, { token: a.token });
    assert(
      notifications.notifications.some((n) => n.type === "match"),
      "A should have a match notification"
    );
    console.log("[e2e-smoke] PASS");
  } finally {
    await cleanup();
  }
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL: ${err.message}`);
  process.exit(1);
});
