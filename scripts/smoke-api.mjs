#!/usr/bin/env node
/* eslint-disable no-console */
// API smoke test of the critical dating flow against a running API.
// Cross-platform (Node 20+, no dependency):
//   node scripts/smoke-api.mjs [--api http://localhost:18080]
import { deflateSync } from "node:zlib";

const argIndex = process.argv.indexOf("--api");
const API = (argIndex > 0 ? process.argv[argIndex + 1] : process.env.API_URL) || "http://localhost:18080";
const PASSWORD = "Password123";
const run = `${Date.now()}${Math.floor(Math.random() * 1e4)}`;

let failures = 0;
function check(condition, label) {
  if (condition) {
    console.log(`  ✔ ${label}`);
  } else {
    failures++;
    console.error(`  ✖ ${label}`);
  }
}

async function call(method, path, { body, token, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) payload = form;
  else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(`${API}${path}`, { method, headers, body: payload });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  return { status: res.status, data };
}

// Minimal PNG encoder (solid colour) so the smoke test needs no fixture.
const crcTable = Array.from({ length: 256 }, (_, n) => {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});
function crc32(buf) {
  let c = 0xffffffff;
  for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}
function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const td = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(td));
  return Buffer.concat([len, td, crc]);
}
function png(w, h, [r, g, b]) {
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  const row = Buffer.concat([Buffer.from([0]), Buffer.from(Array.from({ length: w }, () => [r, g, b]).flat())]);
  const raw = Buffer.concat(Array.from({ length: h }, () => row));
  return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk("IHDR", ihdr), chunk("IDAT", deflateSync(raw)), chunk("IEND", Buffer.alloc(0))]);
}

async function member(prefix, name, gender, interestedIn, color) {
  const email = `smoke_${prefix}_${run}@lueur.test`;
  const reg = await call("POST", "/auth/register", { body: { email, password: PASSWORD } });
  check(reg.status === 201, `register ${prefix}`);
  const token = reg.data?.accessToken;
  const prof = await call("PUT", "/me/profile", { token, body: { firstName: name, birthdate: "1994-03-10", gender, bio: "Smoke test", city: "Paris" } });
  check(prof.status === 200, `profile ${prefix}`);
  await call("PUT", "/me/preferences", { token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 50 } });
  await call("PUT", "/me/location", { token, body: { latitude: 48.8566, longitude: 2.3522 } });
  const form = new FormData();
  form.append("photo", new Blob([png(320, 400, color)], { type: "image/png" }), "photo.png");
  const up = await call("POST", "/me/photos", { token, form });
  check(up.status === 201, `photo upload ${prefix}`);
  return { token, id: reg.data?.user?.id, email };
}

async function main() {
  console.log(`Smoke test against ${API}`);
  const health = await call("GET", "/health");
  check(health.status === 200 && health.data?.status === "ok", "GET /health");

  const a = await member("a", "Anna", "woman", ["man"], [185, 72, 47]);
  const b = await member("b", "Basile", "man", ["woman"], [61, 64, 91]);

  const me = await call("GET", "/me/profile", { token: a.token });
  check(me.data?.isComplete === true, "profile complete");

  const feed = await call("GET", "/discovery?limit=20", { token: a.token });
  check(feed.status === 200 && feed.data.profiles.some((p) => p.userId === b.id), "B appears in A's discovery");

  const like1 = await call("POST", "/discovery/swipes", { token: a.token, body: { targetUserId: b.id, action: "like" } });
  check(like1.status === 201 && !like1.data.matched, "A likes B (no match yet)");
  const like2 = await call("POST", "/discovery/swipes", { token: b.token, body: { targetUserId: a.id, action: "like" } });
  check(like2.status === 201 && like2.data.matched, "B likes A → match");
  const matchId = like2.data?.match?.id;

  const send = await call("POST", `/matches/${matchId}/messages`, { token: a.token, body: { content: "Bonjour !", clientId: "smoke-1" } });
  check(send.status === 201, "A sends a message");
  const read = await call("GET", `/matches/${matchId}/messages`, { token: b.token });
  check(read.data?.messages?.some((m) => m.content === "Bonjour !"), "B reads the message");
  const outsider = await member("c", "Chloe", "woman", ["man"], [107, 143, 113]);
  const forbidden = await call("GET", `/matches/${matchId}/messages`, { token: outsider.token });
  check(forbidden.status === 404, "outsider cannot read the conversation");

  const notif = await call("GET", "/notifications", { token: b.token });
  check(notif.data?.notifications?.some((n) => n.type === "match"), "match notification");

  const block = await call("POST", "/blocks", { token: b.token, body: { userId: a.id } });
  check(block.status === 204, "B blocks A");
  const gone = await call("GET", `/matches/${matchId}/messages`, { token: a.token });
  check(gone.status === 404, "conversation removed after block");

  for (const m of [a, b, outsider]) {
    const del = await call("DELETE", "/me", { token: m.token, body: { password: PASSWORD } });
    check(del.status === 204, `delete account ${m.email.split("_")[1]}`);
  }

  if (failures) {
    console.error(`\n${failures} check(s) failed`);
    process.exit(1);
  }
  console.log("\nAll smoke checks passed");
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
