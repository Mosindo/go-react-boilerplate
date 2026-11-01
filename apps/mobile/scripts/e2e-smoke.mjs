// Smoke test de bout en bout contre une API en marche (parcours critique):
// inscription -> profil -> photo -> decouverte -> like -> match -> conversation temps reel.
//
//   EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npm run e2e:smoke
//
// Utilise uniquement fetch / WebSocket natifs (Node >= 22). Cree 2 comptes jetables.
import { deflateSync } from "node:zlib";

const BASE = (process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
const stamp = Date.now();
let failures = 0;

function step(name, ok, detail = "") {
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? ` — ${detail}` : ""}`);
  if (!ok) failures += 1;
}

async function call(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(BASE + path, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    /* binaire */
  }
  return { status: res.status, json };
}

// PNG 300x300 minimal genere sans dependance.
function png(r, g, b) {
  const w = 300;
  const h = 300;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) {
    const row = y * (w * 3 + 1);
    for (let x = 0; x < w; x++) {
      raw[row + 1 + x * 3] = r;
      raw[row + 2 + x * 3] = g;
      raw[row + 3 + x * 3] = b;
    }
  }
  const crcTable = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (buf) => {
    let c = 0xffffffff;
    for (const byte of buf) c = crcTable[(c ^ byte) & 0xff] ^ (c >>> 8);
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
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

async function signup(prefix, name, gender, seeking) {
  const email = `${prefix}${stamp}@smoke.example`;
  const reg = await call("POST", "/auth/register", { body: { email, password: "Smoke-Passw0rd" } });
  step(`register ${name}`, reg.status === 201, `status ${reg.status}`);
  const token = reg.json.accessToken;
  const form = new FormData();
  form.append("file", new Blob([png(200, 90, 110)], { type: "image/png" }), "p.png");
  step(`photo ${name}`, (await call("POST", "/me/photos", { token, form })).status === 201);
  const profile = await call("PUT", "/me/profile", {
    token,
    body: {
      firstName: name,
      birthDate: "1993-05-05",
      gender,
      bio: "Smoke",
      city: "Paris",
      latitude: 48.85,
      longitude: 2.35,
      interests: ["music"],
    },
  });
  step(`profile ${name}`, profile.status === 200, `status ${profile.status}`);
  const settings = await call("PUT", "/me/settings", {
    token,
    body: { interestedIn: seeking, minAge: 18, maxAge: 99, maxDistanceKm: 50, discoverable: true, showDistance: true },
  });
  step(`settings ${name}`, settings.status === 200);
  const login = await call("POST", "/auth/login", { body: { email, password: "Smoke-Passw0rd" } });
  step(`login ${name}`, login.status === 200);
  return { token: login.json.accessToken, id: reg.json.user.id };
}

function listen(token) {
  return new Promise((resolve, reject) => {
    const events = [];
    const ws = new WebSocket(BASE.replace(/^http/, "ws") + "/ws");
    ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
    ws.onmessage = (m) => {
      const ev = JSON.parse(m.data);
      events.push(ev);
      if (ev.type === "ready") resolve({ ws, events });
    };
    ws.onerror = () => reject(new Error("websocket error"));
    setTimeout(() => reject(new Error("websocket timeout")), 5000);
  });
}

const waitFor = async (predicate, ms = 4000) => {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    if (predicate()) return true;
    await new Promise((r) => setTimeout(r, 50));
  }
  return false;
};

const health = await call("GET", "/health");
step("health", health.status === 200, BASE);
if (health.status !== 200) process.exit(1);

const alice = await signup("alice", "Alice", "woman", ["man"]);
const bob = await signup("bob", "Bob", "man", ["woman"]);
const wsAlice = await listen(alice.token);
const wsBob = await listen(bob.token);

const feed = await call("GET", "/discover", { token: alice.token });
step(
  "discover shows Bob",
  feed.json.items.some((c) => c.userId === bob.id),
);
step(
  "like (no match yet)",
  (await call("POST", `/profiles/${bob.id}/like`, { token: alice.token })).json.matched === false,
);
const match = await call("POST", `/profiles/${alice.id}/like`, { token: bob.token });
step("reciprocal like creates a match", match.json.matched === true);
step(
  "realtime match event (both)",
  (await waitFor(() => wsAlice.events.some((e) => e.type === "match"))) &&
    (await waitFor(() => wsBob.events.some((e) => e.type === "match"))),
);

const conv = match.json.conversationId;
const sent = await call("POST", `/conversations/${conv}/messages`, { token: alice.token, body: { body: "Bonjour !" } });
step("send message", sent.status === 201);
step(
  "realtime message delivered to Bob",
  await waitFor(() => wsBob.events.some((e) => e.type === "message" && e.payload.body === "Bonjour !")),
);
step("Bob sees unread", (await call("GET", "/conversations", { token: bob.token })).json.items[0]?.unreadCount === 1);
await call("POST", `/conversations/${conv}/read`, { token: bob.token });
step("read receipt pushed to Alice", await waitFor(() => wsAlice.events.some((e) => e.type === "read")));
step("notifications readable", (await call("GET", "/notifications", { token: bob.token })).status === 200);

for (const u of [alice, bob]) {
  const del = await call("DELETE", "/me", { token: u.token, body: { password: "Smoke-Passw0rd" } });
  step("cleanup account", del.status === 204);
}
wsAlice.ws.close();
wsBob.ws.close();

console.log(failures === 0 ? "\nSMOKE OK" : `\nSMOKE FAILED (${failures})`);
process.exit(failures === 0 ? 0 : 1);
