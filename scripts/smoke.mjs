#!/usr/bin/env node
// End-to-end smoke test of the critical user journey against a running API:
//   register -> profile -> photo -> discover -> like -> match -> chat -> notifications
//   -> realtime -> block/report -> account deletion
//
// Usage: API_URL=http://<LAN_IP>:18080 node scripts/smoke.mjs   (Node >= 22)
import { deflateSync } from "node:zlib";

const API = (process.env.API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "smoke-test-password-1";
const run = Date.now().toString(36);

let step = 0;
function log(message) {
  step += 1;
  console.log(`  ${String(step).padStart(2, "0")}. ${message}`);
}
function assert(condition, message) {
  if (!condition) {
    throw new Error(`ASSERTION FAILED: ${message}`);
  }
}

async function call(method, path, { token, body, form, expect } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(API + path, {
    method,
    headers,
    body: form ?? (body === undefined ? undefined : JSON.stringify(body))
  });
  const text = await response.text();
  const data = text ? JSON.parse(text) : undefined;
  if (expect !== undefined && response.status !== expect) {
    throw new Error(`${method} ${path} -> ${response.status} (expected ${expect}): ${text}`);
  }
  return { status: response.status, data };
}

function crc32(buf) {
  let c;
  let crc = 0xffffffff;
  for (let n = 0; n < buf.length; n++) {
    c = (crc ^ buf[n]) & 0xff;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    crc = (crc >>> 8) ^ c;
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function pngChunk(type, data) {
  const out = Buffer.alloc(12 + data.length);
  out.writeUInt32BE(data.length, 0);
  out.write(type, 4, "ascii");
  data.copy(out, 8);
  out.writeUInt32BE(crc32(out.subarray(4, 8 + data.length)), 8 + data.length);
  return out;
}

function makePng(width, height, seed) {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const rows = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const offset = y * (width * 3 + 1);
    for (let x = 0; x < width; x++) {
      rows[offset + 1 + x * 3] = (x + seed) % 256;
      rows[offset + 2 + x * 3] = (y * 2) % 256;
      rows[offset + 3 + x * 3] = (x + y + seed) % 256;
    }
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    pngChunk("IHDR", header),
    pngChunk("IDAT", deflateSync(rows)),
    pngChunk("IEND", Buffer.alloc(0))
  ]);
}

async function createUser(name, gender, interestedIn) {
  const email = `smoke-${name.toLowerCase()}-${run}@smoke.invalid`;
  const { data: session } = await call("POST", "/auth/register", { body: { email, password: PASSWORD }, expect: 201 });
  const token = session.accessToken;
  await call("PUT", "/profile", {
    token,
    expect: 200,
    body: {
      firstName: name,
      birthDate: "1994-06-12",
      gender,
      bio: `Profil de test ${name}`,
      city: "Paris",
      latitude: 48.8566,
      longitude: 2.3522,
      interestIds: [1, 3, 5]
    }
  });
  await call("PUT", "/profile/preferences", {
    token,
    expect: 200,
    body: { interestedIn, minAge: 18, maxAge: 60, maxDistanceKm: 50 }
  });
  const form = new FormData();
  form.append("photo", new Blob([makePng(480, 600, name.length)], { type: "image/png" }), "photo.png");
  await call("POST", "/photos", { token, form, expect: 201 });
  const { data: me } = await call("GET", "/me", { token, expect: 200 });
  assert(me.onboardingComplete, `${name} should have completed onboarding`);
  return { id: me.id, email, token, name };
}

function waitForEvent(socket, type, timeoutMs = 5000) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`timeout waiting for "${type}" event`)), timeoutMs);
    const onMessage = (event) => {
      const parsed = JSON.parse(event.data);
      if (parsed.type === type) {
        clearTimeout(timer);
        socket.removeEventListener("message", onMessage);
        resolve(parsed);
      }
    };
    socket.addEventListener("message", onMessage);
  });
}

async function openSocket(token) {
  const socket = new WebSocket(API.replace(/^http/, "ws") + "/ws");
  await new Promise((resolve, reject) => {
    socket.addEventListener("open", resolve, { once: true });
    socket.addEventListener("error", () => reject(new Error("websocket connection failed")), { once: true });
  });
  const ready = waitForEvent(socket, "ready");
  socket.send(JSON.stringify({ type: "auth", token }));
  await ready;
  return socket;
}

async function main() {
  console.log(`Smoke test against ${API}`);

  const health = await call("GET", "/health", { expect: 200 });
  assert(health.data.status === "ok", "health status");
  log("API is healthy");

  const alice = await createUser("Alice", "woman", ["man"]);
  const bob = await createUser("Bob", "man", ["woman"]);
  log("registered two users and completed onboarding (profile, preferences, photo)");

  const login = await call("POST", "/auth/login", { body: { email: alice.email, password: PASSWORD }, expect: 200 });
  assert(login.data.user.id === alice.id, "login returns the same user");
  log("login works");

  await call("POST", "/auth/login", { body: { email: alice.email, password: "wrong-password" }, expect: 401 });
  await call("GET", "/me", { expect: 401 });
  log("bad credentials and anonymous access are rejected");

  const aliceSocket = await openSocket(alice.token);
  const bobSocket = await openSocket(bob.token);
  log("both users connected over WebSocket");

  const { data: discover } = await call("GET", "/discover?limit=30", { token: alice.token, expect: 200 });
  const bobCard = discover.profiles.find((p) => p.id === bob.id);
  assert(bobCard, "Alice should discover Bob");
  assert(bobCard.distanceKm !== null && !("latitude" in bobCard) && !("birthDate" in bobCard), "only approximate data is exposed");
  log("Alice discovers Bob (approximate distance only)");

  const photoUrl = bobCard.photos[0].url;
  const photo = await fetch(API + photoUrl, { headers: { Authorization: `Bearer ${alice.token}` } });
  assert(photo.status === 200 && photo.headers.get("content-type") === "image/jpeg", "photo is served as JPEG to authenticated users");
  const anonymousPhoto = await fetch(API + photoUrl);
  assert(anonymousPhoto.status === 401, "photos are never public");
  log("photos are protected");

  const first = await call("POST", "/swipes", { token: alice.token, body: { userId: bob.id, action: "like" }, expect: 200 });
  assert(!first.data.matched, "no match on a one-sided like");
  const repeat = await call("POST", "/swipes", { token: alice.token, body: { userId: bob.id, action: "like" }, expect: 200 });
  assert(repeat.data.alreadyProcessed, "repeated like is idempotent");
  log("one-sided like does not match; repeating it changes nothing");

  const matchForAlice = waitForEvent(aliceSocket, "match");
  const matchForBob = waitForEvent(bobSocket, "match");
  const second = await call("POST", "/swipes", { token: bob.token, body: { userId: alice.id, action: "like" }, expect: 200 });
  assert(second.data.matched && second.data.match.user.id === alice.id, "mutual like creates a match");
  await Promise.all([matchForAlice, matchForBob]);
  const conversationId = second.data.match.conversationId;
  log("mutual like creates a match, announced in realtime to both");

  const incoming = waitForEvent(bobSocket, "message");
  const sent = await call("POST", `/conversations/${conversationId}/messages`, {
    token: alice.token,
    body: { body: "Salut Bob !" },
    expect: 201
  });
  const pushed = await incoming;
  assert(pushed.data.body === "Salut Bob !" && pushed.data.id === sent.data.id, "message delivered in realtime");
  log("message sent and received in realtime");

  const { data: bobChats } = await call("GET", "/conversations", { token: bob.token, expect: 200 });
  assert(bobChats.conversations[0].unreadCount === 1, "unread count for the recipient");
  const readReceipt = waitForEvent(aliceSocket, "read");
  await call("POST", `/conversations/${conversationId}/read`, { token: bob.token, expect: 200 });
  await readReceipt;
  const { data: history } = await call("GET", `/conversations/${conversationId}/messages`, { token: alice.token, expect: 200 });
  assert(history.messages[0].readAt, "read receipt visible to the sender");
  log("read/unread state works");

  const { data: notifications } = await call("GET", "/notifications", { token: bob.token, expect: 200 });
  assert(notifications.notifications.some((n) => n.type === "match"), "match notification");
  await call("POST", "/notifications/read-all", { token: bob.token, expect: 204 });
  log("notifications are created and can be read");

  const mallory = await createUser("Mallory", "man", ["woman"]);
  await call("GET", `/conversations/${conversationId}/messages`, { token: mallory.token, expect: 404 });
  await call("POST", `/conversations/${conversationId}/messages`, { token: mallory.token, body: { body: "hi" }, expect: 404 });
  log("an outsider can neither read nor write the conversation");

  await call("POST", "/reports", { token: alice.token, body: { userId: mallory.id, reason: "spam", details: "smoke" }, expect: 201 });
  await call("POST", "/blocks", { token: alice.token, body: { userId: bob.id }, expect: 204 });
  await call("GET", `/conversations/${conversationId}/messages`, { token: bob.token, expect: 404 });
  await call("GET", `/profiles/${alice.id}`, { token: bob.token, expect: 404 });
  log("report and block work; blocking removes the conversation for both");

  for (const user of [alice, bob, mallory]) {
    await call("DELETE", "/me", { token: user.token, body: { password: PASSWORD }, expect: 204 });
    await call("GET", "/me", { token: user.token, expect: 401 });
  }
  log("accounts deleted and their tokens revoked");

  aliceSocket.close();
  bobSocket.close();
  console.log(`\nSmoke test passed (${step} checks).`);
}

main().catch((error) => {
  console.error(`\nSmoke test FAILED: ${error.message}`);
  process.exit(1);
});
