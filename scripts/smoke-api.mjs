#!/usr/bin/env node
// End-to-end smoke test of a running API (Node >= 22, no dependencies).
//   API_URL=http://localhost:18080 node scripts/smoke-api.mjs
// Flow: register x2 -> onboarding (profile, preferences, photo) -> discover -> like x2
//       -> match -> real-time chat -> read receipt -> notifications -> cleanup (delete accounts).
import { deflateSync } from "node:zlib";

const API = (process.env.API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "smoke-test-password";
let failures = 0;

function step(name) {
  console.log(`- ${name}`);
}
function check(condition, message) {
  if (!condition) {
    failures += 1;
    console.error(`  FAIL: ${message}`);
    throw new Error(message);
  }
}

async function call(method, path, { token, body, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(API + path, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  return { status: res.status, data };
}

function crc32(buf) {
  let c = ~0;
  for (const byte of buf) {
    c ^= byte;
    for (let k = 0; k < 8; k++) c = (c >>> 1) ^ (0xedb88320 & -(c & 1));
  }
  return ~c >>> 0;
}
function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}
/** Generates a small valid gradient PNG (the API re-encodes it as JPEG). */
function makePng(size = 320) {
  const raw = Buffer.alloc((size * 3 + 1) * size);
  for (let y = 0; y < size; y++) {
    const row = y * (size * 3 + 1);
    for (let x = 0; x < size; x++) {
      raw[row + 1 + x * 3] = (x * 255) / size;
      raw[row + 2 + x * 3] = (y * 255) / size;
      raw[row + 3 + x * 3] = 128;
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr.set([8, 2, 0, 0, 0], 8);
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

function birthDate(age) {
  const d = new Date();
  d.setUTCFullYear(d.getUTCFullYear() - age);
  d.setUTCDate(d.getUTCDate() - 1);
  return d.toISOString().slice(0, 10);
}

async function onboard(token, { name, gender, into, lat }) {
  let r = await call("PUT", "/me/profile", {
    token,
    body: {
      firstName: name, birthDate: birthDate(29), gender, bio: `Smoke profile ${name}`, city: "Smoketown",
      latitude: lat, longitude: 10.0, interests: ["music", "travel"],
    },
  });
  check(r.status === 200, `profile ${name}: ${r.status} ${JSON.stringify(r.data)}`);
  r = await call("PUT", "/me/preferences", { token, body: { interestedIn: into, minAge: 18, maxAge: 60, maxDistanceKm: 20 } });
  check(r.status === 200, `preferences ${name}: ${r.status}`);
  const form = new FormData();
  form.append("file", new Blob([makePng()], { type: "image/png" }), "smoke.png");
  r = await call("POST", "/me/photos", { token, form });
  check(r.status === 201, `photo ${name}: ${r.status} ${JSON.stringify(r.data)}`);
}

function nextEvent(ws, type, timeoutMs = 5000) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`timeout waiting for ${type}`)), timeoutMs);
    const onMessage = (event) => {
      const frame = JSON.parse(event.data);
      if (frame.type === type) {
        clearTimeout(timer);
        ws.removeEventListener("message", onMessage);
        resolve(frame.data);
      }
    };
    ws.addEventListener("message", onMessage);
  });
}

const accounts = [];
try {
  step(`health (${API})`);
  const health = await call("GET", "/health");
  check(health.status === 200, `health returned ${health.status}`);

  // A random latitude cell keeps repeated smoke runs from seeing each other.
  const lat = -50 + Math.random() * 100;
  const suffix = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
  const people = [
    { name: "Smokealice", gender: "woman", into: ["man"], email: `alice.${suffix}@smoke.example.org` },
    { name: "Smokebob", gender: "man", into: ["woman"], email: `bob.${suffix}@smoke.example.org` },
  ];

  step("register + login");
  for (const p of people) {
    let r = await call("POST", "/auth/register", { body: { email: p.email, password: PASSWORD } });
    check(r.status === 201, `register ${p.name}: ${r.status} ${JSON.stringify(r.data)}`);
    r = await call("POST", "/auth/login", { body: { email: p.email, password: PASSWORD } });
    check(r.status === 200, `login ${p.name}: ${r.status}`);
    p.token = r.data.accessToken;
    p.id = r.data.user.id;
    accounts.push(p);
    const me = await call("GET", "/me", { token: p.token });
    check(me.status === 200 && me.data.onboarding.complete === false, `me ${p.name}: ${JSON.stringify(me.data)}`);
  }
  const [alice, bob] = people;

  step("onboarding (profile, preferences, photo)");
  for (const p of people) await onboard(p.token, { ...p, lat });

  step("users load: discovery shows the compatible profile");
  let r = await call("GET", "/discover?limit=10", { token: alice.token });
  check(r.status === 200 && r.data.profiles.some((x) => x.id === bob.id), `alice does not discover bob: ${JSON.stringify(r.data)}`);
  const photoUrl = r.data.profiles.find((x) => x.id === bob.id).photos[0].url;
  const img = await fetch(API + photoUrl, { headers: { Authorization: `Bearer ${alice.token}` } });
  check(img.status === 200 && img.headers.get("content-type") === "image/jpeg", "photo is not served as JPEG");
  const anon = await fetch(API + photoUrl);
  check(anon.status === 401, "photos must require authentication");

  step("like -> match");
  r = await call("POST", "/swipes", { token: alice.token, body: { userId: bob.id, action: "like" } });
  check(r.status === 200 && r.data.matched === false, `first like: ${JSON.stringify(r.data)}`);

  const ticket = await call("POST", "/ws/ticket", { token: alice.token });
  check(ticket.status === 200, "ws ticket");
  const ws = new WebSocket(API.replace(/^http/, "ws") + `/ws?ticket=${ticket.data.ticket}`);
  await new Promise((resolve, reject) => {
    ws.addEventListener("open", resolve, { once: true });
    ws.addEventListener("error", () => reject(new Error("websocket failed to open")), { once: true });
  });
  const matchEvent = nextEvent(ws, "match.new");
  r = await call("POST", "/swipes", { token: bob.token, body: { userId: alice.id, action: "like" } });
  check(r.status === 200 && r.data.matched === true, `second like should match: ${JSON.stringify(r.data)}`);
  const conversationId = r.data.match.conversationId;
  check((await matchEvent).conversationId === conversationId, "match.new real-time event");

  step("chat send / real-time read");
  const messageEvent = nextEvent(ws, "message.new");
  r = await call("POST", `/conversations/${conversationId}/messages`, { token: bob.token, body: { body: "Salut Alice !" } });
  check(r.status === 201, `send: ${r.status}`);
  check((await messageEvent).body === "Salut Alice !", "message.new real-time event");
  r = await call("GET", "/conversations", { token: alice.token });
  check(r.data.conversations[0].unreadCount === 1, `unread count: ${JSON.stringify(r.data)}`);
  r = await call("POST", `/conversations/${conversationId}/read`, { token: alice.token });
  check(r.status === 204, `mark read: ${r.status}`);
  r = await call("GET", `/conversations/${conversationId}/messages`, { token: bob.token });
  check(r.data.messages[0].readAt !== null, "read receipt missing");
  ws.close();

  step("notifications read flow");
  r = await call("GET", "/notifications", { token: alice.token });
  check(r.status === 200 && r.data.unreadCount >= 1, `notifications: ${JSON.stringify(r.data)}`);
  r = await call("POST", "/notifications/read-all", { token: alice.token });
  check(r.status === 204, "read-all");

  step("block + report");
  r = await call("POST", "/reports", { token: alice.token, body: { userId: bob.id, reason: "other", details: "smoke test" } });
  check(r.status === 201, `report: ${r.status}`);
  r = await call("POST", "/blocks", { token: alice.token, body: { userId: bob.id } });
  check(r.status === 204, `block: ${r.status}`);
  r = await call("GET", "/matches", { token: bob.token });
  check(r.data.matches.length === 0, "block must dissolve the match");
} catch (error) {
  failures = failures || 1;
  console.error(`smoke test failed: ${error.message}`);
} finally {
  for (const p of accounts) {
    if (p.token) await call("DELETE", "/me", { token: p.token, body: { password: PASSWORD } });
  }
}

if (failures) {
  process.exit(1);
}
console.log("smoke test passed");
