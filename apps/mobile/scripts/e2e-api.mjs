// End-to-end smoke test of the critical user journey against a running API:
// register -> profile -> photo -> discover -> like -> match -> chat (REST + WebSocket) -> read receipts.
//
//   API_URL=http://localhost:18080 npm run e2e:api
import { deflateSync } from "node:zlib";

const API = (process.env.API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
const suffix = Date.now().toString(36);

function crc32(buf) {
  let c;
  let crc = ~0;
  for (const byte of buf) {
    c = (crc ^ byte) & 0xff;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    crc = (crc >>> 8) ^ c;
  }
  return ~crc >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}

function png(size = 128) {
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8;
  ihdr[9] = 2; // RGB
  const row = Buffer.alloc(1 + size * 3);
  for (let x = 0; x < size; x++) {
    row[1 + x * 3] = x * 2;
    row[2 + x * 3] = 90;
    row[3 + x * 3] = 160;
  }
  const raw = Buffer.concat(Array.from({ length: size }, () => row));
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function call(method, path, token, body, expected) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (body instanceof FormData) payload = body;
  else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(`${API}${path}`, { method, headers, body: payload });
  const text = await res.text();
  if (res.status !== expected) throw new Error(`${method} ${path}: expected ${expected}, got ${res.status} ${text}`);
  return text ? JSON.parse(text) : undefined;
}

function step(name) {
  console.log(`✓ ${name}`);
}

async function onboard(label, gender, seeking) {
  const auth = await call("POST", "/auth/register", undefined, { email: `${label}-${suffix}@e2e.test`, password: "e2e-password-1" }, 201);
  const token = auth.accessToken;
  await call(
    "PUT",
    "/me/profile",
    token,
    {
      firstName: label,
      birthDate: "1994-06-15",
      gender,
      bio: `Hello from ${label}`,
      city: "Lyon",
      latitude: 45.76,
      longitude: 4.84,
      interestIds: [1, 2]
    },
    200
  );
  const form = new FormData();
  form.append("photo", new Blob([png()], { type: "image/png" }), "photo.png");
  await call("POST", "/me/photos", token, form, 201);
  await call("PUT", "/me/preferences", token, { interestedIn: seeking, minAge: 18, maxAge: 60, maxDistanceKm: 100 }, 200);
  return { id: auth.user.id, token, label };
}

function nextWsEvent(ws, type) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`timeout waiting for ws event ${type}`)), 5000);
    const handler = (e) => {
      const event = JSON.parse(String(e.data));
      if (event.type === type) {
        clearTimeout(timer);
        ws.removeEventListener("message", handler);
        resolve(event);
      }
    };
    ws.addEventListener("message", handler);
  });
}

const health = await fetch(`${API}/health`);
if (!health.ok) throw new Error(`API not healthy at ${API}`);
step("health");

const alice = await onboard("alice", "woman", ["man"]);
const bob = await onboard("bob", "man", ["woman"]);
step("registered + onboarded two users");

const aliceFeed = await call("GET", "/discover?limit=20", alice.token, undefined, 200);
if (!aliceFeed.profiles.some((p) => p.userId === bob.id)) throw new Error("bob missing from alice's discovery");
if (aliceFeed.profiles.some((p) => p.userId === alice.id)) throw new Error("alice sees herself");
step("discovery lists the other user");

const first = await call("POST", "/swipes", alice.token, { targetUserId: bob.id, action: "like" }, 200);
if (first.matched) throw new Error("one-sided like produced a match");
step("one-sided like does not match");

const bobWs = new WebSocket(API.replace(/^http/, "ws") + "/ws");
await new Promise((resolve, reject) => {
  bobWs.addEventListener("open", resolve, { once: true });
  bobWs.addEventListener("error", reject, { once: true });
});
bobWs.send(JSON.stringify({ type: "auth", token: bob.token }));
await nextWsEvent(bobWs, "ready");
const matchEvent = nextWsEvent(bobWs, "match");

const second = await call("POST", "/swipes", bob.token, { targetUserId: alice.id, action: "like" }, 200);
if (!second.matched || !second.conversationId) throw new Error("mutual like did not match");
await matchEvent;
step("mutual like matches (+ realtime match event)");

const messageEvent = nextWsEvent(bobWs, "message");
await call("POST", `/conversations/${second.conversationId}/messages`, alice.token, { body: "Salut Bob 👋" }, 201);
const received = await messageEvent;
if (received.data.body !== "Salut Bob 👋") throw new Error("wrong realtime message body");
step("message delivered in real time");

const convs = await call("GET", "/conversations", bob.token, undefined, 200);
if (convs.conversations[0]?.unreadCount !== 1) throw new Error("expected 1 unread message");
await call("POST", `/conversations/${second.conversationId}/read`, bob.token, undefined, 204);
const msgs = await call("GET", `/conversations/${second.conversationId}/messages`, alice.token, undefined, 200);
if (!msgs.messages[0].readAt) throw new Error("read receipt missing");
step("unread count + read receipt");

const intruder = await onboard("mallory", "man", ["woman"]);
await call("GET", `/conversations/${second.conversationId}/messages`, intruder.token, undefined, 404);
step("outsider cannot read the conversation");

await call("PUT", `/users/${bob.id}/block`, alice.token, undefined, 204);
await call("POST", `/conversations/${second.conversationId}/messages`, bob.token, { body: "?" }, 404);
step("block ends the conversation");

for (const user of [alice, bob, intruder]) {
  await call("DELETE", "/me", user.token, { password: "e2e-password-1" }, 204);
}
step("accounts deleted");
bobWs.close();
console.log("\nE2E OK");
