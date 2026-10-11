#!/usr/bin/env node
// End-to-end smoke test against a running API (no dependencies, Node 20+).
//   API_URL=http://localhost:18080 node scripts/smoke-api.mjs
import { deflateSync } from "node:zlib";

const API = (process.env.API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
const run = Date.now().toString(36);

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
  const body = Buffer.concat([Buffer.from(type), data]);
  const out = Buffer.alloc(12 + data.length);
  out.writeUInt32BE(data.length, 0);
  body.copy(out, 4);
  out.writeUInt32BE(crc32(body), 8 + data.length);
  return out;
}

function png(width, height) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    raw[y * (width * 3 + 1)] = 0;
    for (let x = 0; x < width; x++) {
      const o = y * (width * 3 + 1) + 1 + x * 3;
      raw[o] = x % 256;
      raw[o + 1] = y % 256;
      raw[o + 2] = 128;
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

async function call(method, path, { token, body, form, expect } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(API + path, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (expect !== undefined && res.status !== expect) {
    throw new Error(`${method} ${path}: expected ${expect}, got ${res.status} ${text}`);
  }
  return data;
}

let step = 0;
function ok(label) {
  console.log(`  ✓ ${String(++step).padStart(2, "0")} ${label}`);
}

async function person(name, gender, into) {
  const auth = await call("POST", "/auth/register", {
    body: { email: `${name.toLowerCase()}-${run}@smoke.invalid`, password: "smoke-pass-123" },
    expect: 201,
  });
  const token = auth.accessToken;
  await call("PUT", "/me/profile", {
    token,
    expect: 200,
    body: { firstName: name, birthDate: "1993-05-05", gender, bio: "smoke", city: "Paris", interestIds: [1, 2] },
  });
  await call("PUT", "/me/preferences", {
    token,
    expect: 200,
    body: { interestedIn: into, minAge: 18, maxAge: 99, maxDistanceKm: null },
  });
  const form = new FormData();
  form.append("file", new Blob([png(320, 400)], { type: "image/png" }), "photo.png");
  await call("POST", "/me/photos", { token, form, expect: 201 });
  return { id: auth.user.id, token };
}

console.log(`Smoke test against ${API}`);
await call("GET", "/health", { expect: 200 });
ok("health");

const a = await person("Alice", "woman", ["man"]);
const b = await person("Bruno", "man", ["woman"]);
ok("register + profile + preferences + photo for two users");

const me = await call("GET", "/me", { token: a.token, expect: 200 });
if (!me.profileComplete) throw new Error("profile should be complete");
ok("/me reports a complete profile");

const deck = await call("GET", "/discover?limit=20", { token: a.token, expect: 200 });
if (!deck.profiles.some((p) => p.userId === b.id)) throw new Error("Bruno missing from Alice's deck");
ok("discovery shows the compatible profile");

const photo = deck.profiles.find((p) => p.userId === b.id).photos[0];
const img = await fetch(API + photo.url);
if (img.status !== 200 || img.headers.get("content-type") !== "image/jpeg") throw new Error("photo not served");
ok("signed photo URL serves a JPEG");

const first = await call("POST", "/swipes", { token: a.token, body: { userId: b.id, action: "like" }, expect: 200 });
if (first.matched) throw new Error("one-sided like must not match");
ok("one-sided like");

const second = await call("POST", "/swipes", { token: b.token, body: { userId: a.id, action: "like" }, expect: 200 });
if (!second.matched || !second.conversation) throw new Error("expected a match");
ok("reciprocal like creates a match");

const conv = second.conversation.id;
await call("POST", `/conversations/${conv}/messages`, { token: a.token, body: { body: "Salut !" }, expect: 201 });
const list = await call("GET", "/conversations", { token: b.token, expect: 200 });
if (list.conversations[0].unreadCount !== 1) throw new Error("unread count wrong");
ok("message sent, unread badge on the recipient");

await call("POST", `/conversations/${conv}/read`, { token: b.token, expect: 204 });
const msgs = await call("GET", `/conversations/${conv}/messages`, { token: a.token, expect: 200 });
if (!msgs.otherLastReadAt) throw new Error("read receipt missing");
ok("read receipt");

const notifs = await call("GET", "/notifications", { token: b.token, expect: 200 });
if (!notifs.notifications.some((n) => n.type === "new_match")) throw new Error("match notification missing");
ok("match notification");

await call("POST", "/blocks", { token: a.token, body: { userId: b.id }, expect: 204 });
await call("GET", `/conversations/${conv}/messages`, { token: a.token, expect: 404 });
ok("block dissolves the conversation");

await call("DELETE", "/me", { token: a.token, body: { password: "smoke-pass-123" }, expect: 204 });
await call("DELETE", "/me", { token: b.token, body: { password: "smoke-pass-123" }, expect: 204 });
ok("accounts deleted");

console.log("\nSMOKE OK");
