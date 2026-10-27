#!/usr/bin/env node
// Critical-path smoke test against a RUNNING API:
// register -> profile -> photo -> discover -> like -> match -> realtime chat -> read -> block -> delete.
// Usage: API_URL=http://localhost:18080 npm run e2e:api
import { deflateSync } from "node:zlib";

const API = (process.env.API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
const stamp = Date.now();

function crc32(buf) {
  let c;
  const table = [];
  for (let n = 0; n < 256; n++) {
    c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    table[n] = c >>> 0;
  }
  let crc = 0xffffffff;
  for (const b of buf) crc = table[(crc ^ b) & 0xff] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}

function png(width, height) {
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type), data]);
    const crc = Buffer.alloc(4);
    crc.writeUInt32BE(crc32(body));
    return Buffer.concat([len, body, crc]);
  };
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    for (let x = 0; x < width; x++) {
      raw[row + 1 + x * 3] = x % 256;
      raw[row + 2 + x * 3] = y % 256;
      raw[row + 3 + x * 3] = 120;
    }
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

let step = 0;
function check(condition, message) {
  step += 1;
  if (!condition) {
    console.error(`✗ ${step}. ${message}`);
    process.exit(1);
  }
  console.log(`✓ ${step}. ${message}`);
}

async function call(method, path, { token, body, form } = {}) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(body ? { "Content-Type": "application/json" } : {})
    },
    body: form ?? (body ? JSON.stringify(body) : undefined)
  });
  const text = await res.text();
  return { status: res.status, data: text ? JSON.parse(text) : null };
}

async function createUser(name, gender, interestedIn) {
  const email = `e2e_${name.toLowerCase()}_${stamp}@e2e.test`;
  const reg = await call("POST", "/auth/register", { body: { email, password: "Password123" } });
  check(reg.status === 201, `${name} registers`);
  const token = reg.data.accessToken;
  const profile = await call("PUT", "/profile/me", {
    token,
    body: {
      firstName: name,
      birthDate: "1994-05-05",
      gender,
      bio: "E2E user",
      city: "Paris",
      latitude: 48.8566,
      longitude: 2.3522,
      interests: ["music", "travel"]
    }
  });
  check(profile.status === 200, `${name} creates a profile`);
  const prefs = await call("PUT", "/profile/preferences", {
    token,
    body: { interestedIn, ageMin: 18, ageMax: 60, maxDistanceKm: 100 }
  });
  check(prefs.status === 200, `${name} sets preferences`);
  const form = new FormData();
  form.append("photo", new Blob([png(400, 500)], { type: "image/png" }), "photo.png");
  const photo = await call("POST", "/photos", { token, form });
  check(photo.status === 201, `${name} uploads a photo`);
  return { name, email, token, id: reg.data.user.id };
}

function openSocket(user) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`${API.replace(/^http/, "ws")}/ws`);
    const events = [];
    const waiters = [];
    ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token: user.token }));
    ws.onerror = () => reject(new Error("websocket error"));
    ws.onmessage = (e) => {
      const event = JSON.parse(e.data);
      if (event.type === "ready") return resolve({ ws, next });
      const waiter = waiters.shift();
      if (waiter) waiter(event);
      else events.push(event);
    };
    function next(timeout = 4000) {
      if (events.length) return Promise.resolve(events.shift());
      return new Promise((res, rej) => {
        const timer = setTimeout(
          () => rej(new Error("timed out waiting for realtime event")),
          timeout
        );
        waiters.push((event) => {
          clearTimeout(timer);
          res(event);
        });
      });
    }
  });
}

const alice = await createUser("Alice", "woman", ["man"]);
const bob = await createUser("Bob", "man", ["woman"]);
const bobSocket = await openSocket(bob);
check(true, "Bob opens a realtime connection");

const feed = await call("GET", "/discover?limit=20", { token: alice.token });
check(
  feed.status === 200 && feed.data.profiles.some((p) => p.userId === bob.id),
  "Alice discovers Bob"
);
check(!JSON.stringify(feed.data).includes("latitude"), "discovery never exposes coordinates");

const aliceLike = await call("POST", "/swipes", {
  token: alice.token,
  body: { targetId: bob.id, action: "like" }
});
check(
  aliceLike.status === 200 && aliceLike.data.matched === false,
  "Alice likes Bob (no match yet)"
);
const duplicate = await call("POST", "/swipes", {
  token: alice.token,
  body: { targetId: bob.id, action: "like" }
});
check(duplicate.status === 409, "a repeated like is rejected");

const bobLike = await call("POST", "/swipes", {
  token: bob.token,
  body: { targetId: alice.id, action: "like" }
});
check(
  bobLike.status === 200 && bobLike.data.matched === true,
  "Bob likes Alice back: it's a match"
);
const matchEvent = await bobSocket.next();
check(matchEvent.type === "match", "Bob is notified of the match in realtime");

const conversationId = bobLike.data.conversationId;
const sent = await call("POST", `/conversations/${conversationId}/messages`, {
  token: alice.token,
  body: { body: "Hello Bob 👋" }
});
check(sent.status === 201, "Alice sends a message");
const messageEvent = await bobSocket.next();
check(
  messageEvent.type === "message" && messageEvent.payload.body === "Hello Bob 👋",
  "Bob receives the message in realtime"
);

const read = await call("POST", `/conversations/${conversationId}/read`, { token: bob.token });
check(read.status === 200, "Bob marks the conversation as read");
const history = await call("GET", `/conversations/${conversationId}/messages`, {
  token: alice.token
});
check(history.data.otherLastReadId === sent.data.id, "Alice sees the message as read");

const intruder = await createUser("Mallory", "man", ["woman"]);
const spy = await call("GET", `/conversations/${conversationId}/messages`, {
  token: intruder.token
});
check(spy.status === 404, "a third user cannot read the conversation");

const blocked = await call("POST", "/blocks", { token: alice.token, body: { userId: bob.id } });
check(blocked.status === 204, "Alice blocks Bob");
const afterBlock = await call("POST", `/conversations/${conversationId}/messages`, {
  token: bob.token,
  body: { body: "hello?" }
});
check(afterBlock.status === 404, "Bob can no longer message Alice");

for (const user of [alice, bob, intruder]) {
  const del = await call("DELETE", "/me", { token: user.token, body: { password: "Password123" } });
  check(del.status === 204, `${user.name} deletes the account`);
}
bobSocket.ws.close();
console.log(`\nAll ${step} checks passed against ${API}`);
process.exit(0);
