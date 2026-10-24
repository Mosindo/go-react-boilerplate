#!/usr/bin/env node
// End-to-end smoke test against a running API (no dependencies, Node 20+).
//   node scripts/smoke.mjs [http://localhost:18080]
// Flow: register x2 -> profile -> photo -> discover -> like -> match -> chat -> read -> notifications.

const base = (process.argv[2] ?? process.env.API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
const stamp = Date.now();

// 1x1 PNG
const PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
  "base64",
);

async function call(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) payload = form;
  else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(base + path, { method, headers, body: payload });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    /* non-JSON body */
  }
  return { status: res.status, json };
}

function expect(label, res, status) {
  if (res.status !== status) {
    console.error(`FAIL ${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.json)}`);
    process.exit(1);
  }
  console.log(`ok   ${label}`);
  return res.json;
}

async function onboard(name, gender, interestedIn) {
  const email = `smoke-${name.toLowerCase()}-${stamp}@smoke.invalid`;
  const auth = expect(`register ${name}`, await call("POST", "/auth/register", { body: { email, password: "Smoke-pass-123" } }), 201);
  const token = auth.accessToken;
  const birth = new Date(Date.now() - 30 * 365.25 * 864e5).toISOString().slice(0, 10);
  expect(`profile ${name}`, await call("PUT", "/me/profile", { token, body: { firstName: name, birthDate: birth, gender, bio: "Smoke test", city: "Paris", interests: ["voyage"] } }), 200);
  expect(`preferences ${name}`, await call("PUT", "/me/preferences", { token, body: { interestedIn, ageMin: 18, ageMax: 60 } }), 200);
  const form = new FormData();
  form.append("photo", new Blob([PNG], { type: "image/png" }), "p.png");
  expect(`photo ${name}`, await call("POST", "/me/photos", { token, form }), 201);
  return { id: auth.user.id, token, email };
}

expect("health", await call("GET", "/health"), 200);
const ana = await onboard("Ana", "woman", ["man"]);
const ben = await onboard("Ben", "man", ["woman"]);

const feed = expect("discover (Ana)", await call("GET", "/discover?limit=20", { token: ana.token }), 200);
if (!feed.profiles.some((p) => p.userId === ben.id)) {
  console.error("FAIL discover: Ben not in Ana's feed");
  process.exit(1);
}
const first = expect("like (Ana -> Ben)", await call("POST", "/swipes", { token: ana.token, body: { targetId: ben.id, action: "like" } }), 200);
if (first.matched) {
  console.error("FAIL: match before reciprocal like");
  process.exit(1);
}
const second = expect("like back (Ben -> Ana)", await call("POST", "/swipes", { token: ben.token, body: { targetId: ana.id, action: "like" } }), 200);
if (!second.matched) {
  console.error("FAIL: expected a match");
  process.exit(1);
}
const conv = second.match.conversationId;
expect("send message", await call("POST", `/conversations/${conv}/messages`, { token: ben.token, body: { content: "Salut !" } }), 201);
const list = expect("conversations (Ana)", await call("GET", "/conversations", { token: ana.token }), 200);
if (list.conversations[0]?.unreadCount !== 1) {
  console.error("FAIL: expected 1 unread message");
  process.exit(1);
}
expect("mark read", await call("POST", `/conversations/${conv}/read`, { token: ana.token }), 200);
const notes = expect("notifications (Ana)", await call("GET", "/notifications", { token: ana.token }), 200);
if (!notes.notifications.some((n) => n.type === "match")) {
  console.error("FAIL: missing match notification");
  process.exit(1);
}
for (const u of [ana, ben]) {
  expect(`delete account ${u.email}`, await call("DELETE", "/me", { token: u.token, body: { password: "Smoke-pass-123" } }), 204);
}
console.log("\nSmoke test passed.");
