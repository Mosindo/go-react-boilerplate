/* eslint-disable no-console */
// End-to-end smoke test against a running API (no device needed):
//   register -> profile -> photo -> discover -> like -> match -> realtime chat -> read receipts
//   -> block -> account deletion.
// Usage: MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
const zlib = require("node:zlib");

const API = (
  process.env.MOBILE_E2E_API_URL ||
  process.env.EXPO_PUBLIC_API_URL ||
  "http://localhost:18080"
).replace(/\/+$/, "");
const PASSWORD = "Password123";

function assert(condition, message) {
  if (!condition) throw new Error(`ASSERTION FAILED: ${message}`);
}

function step(name) {
  console.log(`✓ ${name}`);
}

/** Builds a valid 320x320 RGB PNG (the API re-encodes it to JPEG). */
function makePng() {
  const w = 320;
  const h = 320;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) {
    raw[y * (w * 3 + 1)] = 0;
    for (let x = 0; x < w; x++) {
      const o = y * (w * 3 + 1) + 1 + x * 3;
      raw[o] = x % 256;
      raw[o + 1] = y % 256;
      raw[o + 2] = 128;
    }
  }
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type), data]);
    const crc = Buffer.alloc(4);
    crc.writeUInt32BE(zlib.crc32(body));
    return Buffer.concat([len, body, crc]);
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

async function call(token, method, path, body) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined && !(body instanceof FormData)) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body)
  });
  const text = await res.text();
  return { status: res.status, data: text ? JSON.parse(text) : undefined };
}

async function onboard(name, gender, interestedIn) {
  const email = `${name.toLowerCase()}_${Date.now()}_${Math.floor(Math.random() * 1e5)}@e2e.invalid`;
  const reg = await call(null, "POST", "/auth/register", { email, password: PASSWORD });
  assert(reg.status === 201, `register ${name}: ${reg.status}`);
  const token = reg.data.accessToken;
  const profile = await call(token, "PUT", "/profile", {
    firstName: name,
    birthDate: "1994-04-12",
    gender,
    bio: `Hello from ${name}`,
    city: "Paris",
    interests: [1, 3]
  });
  assert(profile.status === 200, `profile ${name}: ${profile.status}`);
  const prefs = await call(token, "PUT", "/preferences", {
    interestedIn,
    minAge: 18,
    maxAge: 60,
    maxDistanceKm: null,
    showDistance: true
  });
  assert(prefs.status === 200, `preferences ${name}: ${prefs.status}`);
  const form = new FormData();
  form.append("photo", new Blob([makePng()], { type: "image/png" }), "photo.png");
  const photo = await call(token, "POST", "/profile/photos", form);
  assert(photo.status === 201, `photo ${name}: ${photo.status}`);
  return { name, email, token, id: reg.data.user.id };
}

function openSocket(token) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`${API.replace(/^http/, "ws")}/ws`);
    const events = [];
    const waiters = [];
    ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
    ws.onerror = () => reject(new Error("websocket error"));
    ws.onmessage = (m) => {
      const event = JSON.parse(String(m.data));
      if (event.type === "ready") return resolve({ ws, next });
      events.push(event);
      waiters.splice(0).forEach((w) => w());
    };
    async function next(type, timeoutMs = 4000) {
      const deadline = Date.now() + timeoutMs;
      for (;;) {
        const idx = events.findIndex((e) => e.type === type);
        if (idx >= 0) return events.splice(idx, 1)[0];
        const left = deadline - Date.now();
        if (left <= 0) throw new Error(`timed out waiting for ${type}`);
        await new Promise((r) => {
          waiters.push(r);
          setTimeout(r, left);
        });
      }
    }
  });
}

async function main() {
  console.log(`Smoke testing ${API}`);
  const health = await call(null, "GET", "/health");
  assert(health.status === 200, `health: ${health.status}`);
  step("health");

  const ana = await onboard("Ana", "woman", ["man"]);
  const ben = await onboard("Ben", "man", ["woman"]);
  step("register + profile + preferences + photo for two users");

  const unauth = await call(null, "GET", "/discover");
  assert(unauth.status === 401, "discover must require auth");
  step("protected endpoints reject anonymous requests");

  const feed = await call(ben.token, "GET", "/discover?limit=20");
  assert(feed.status === 200, `discover: ${feed.status}`);
  assert(
    feed.data.profiles.some((p) => p.userId === ana.id),
    "Ben should discover Ana"
  );
  assert(!feed.data.profiles.some((p) => p.userId === ben.id), "no self in discover");
  step("discovery returns compatible profiles");

  const sockAna = await openSocket(ana.token);
  const sockBen = await openSocket(ben.token);

  const first = await call(ben.token, "POST", "/swipes", { userId: ana.id, action: "like" });
  assert(first.status === 200 && first.data.matched === false, "one-sided like must not match");
  const second = await call(ana.token, "POST", "/swipes", { userId: ben.id, action: "like" });
  assert(second.data.matched === true && second.data.match, "mutual like must match");
  const dup = await call(ana.token, "POST", "/swipes", { userId: ben.id, action: "like" });
  assert(dup.data.duplicate === true, "repeated like is a no-op");
  step("like -> match (no duplicates)");

  const matchEvent = await sockBen.next("match.new");
  assert(matchEvent.data.profile.firstName === "Ana", "Ben gets a live match event");
  step("realtime match notification");

  const conversationId = second.data.match.conversationId;
  const sent = await call(ben.token, "POST", `/conversations/${conversationId}/messages`, {
    body: "Bonjour Ana 👋"
  });
  assert(sent.status === 201, `send: ${sent.status}`);
  const live = await sockAna.next("message.new");
  assert(live.data.body === "Bonjour Ana 👋", "Ana receives the message live");
  step("realtime chat delivery");

  const conv = await call(ana.token, "GET", "/conversations");
  assert(conv.data.conversations[0].unreadCount === 1, "unread count = 1");
  await call(ana.token, "POST", `/conversations/${conversationId}/read`);
  const read = await sockBen.next("message.read");
  assert(read.data.conversationId === conversationId, "Ben sees the read receipt live");
  const after = await call(ana.token, "GET", "/conversations");
  assert(after.data.conversations[0].unreadCount === 0, "unread cleared");
  step("read receipts + unread counters");

  const photoId = (await call(ana.token, "GET", "/profile")).data.photos[0].id;
  const anonPhoto = await call(null, "GET", `/photos/${photoId}`);
  assert(anonPhoto.status === 401, "photos are private");
  step("photos require authentication");

  const intruder = await onboard("Eve", "woman", ["man"]);
  const peek = await call(intruder.token, "GET", `/conversations/${conversationId}/messages`);
  assert(peek.status === 404, "non-members cannot read a conversation");
  step("conversation permissions");

  const notes = await call(ana.token, "GET", "/notifications");
  assert(
    notes.data.notifications.some((n) => n.type === "new_match"),
    "match notification stored"
  );
  step("notifications");

  const block = await call(ana.token, "POST", "/blocks", { userId: ben.id });
  assert(block.status === 204, "block");
  const gone = await call(ben.token, "GET", `/conversations/${conversationId}/messages`);
  assert(gone.status === 404, "block dissolves the conversation");
  step("block");

  for (const user of [ana, ben, intruder]) {
    const del = await call(user.token, "POST", "/me/delete", { password: PASSWORD });
    assert(del.status === 204, `delete ${user.name}: ${del.status}`);
    const login = await call(null, "POST", "/auth/login", { email: user.email, password: PASSWORD });
    assert(login.status === 401, "deleted account cannot log in");
  }
  step("account deletion");

  sockAna.ws.close();
  sockBen.ws.close();
  console.log("\nAll smoke checks passed.");
}

main().catch((error) => {
  console.error(`\nSmoke test failed: ${error.message}`);
  process.exit(1);
});
