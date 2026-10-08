/* eslint-disable no-console */
// API smoke test for a running server: register -> profile -> photo -> discover -> like -> match -> chat -> notifications.
// Usage: MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke   (use a database that has no other demo users)
const fs = require("fs");
const path = require("path");

const API = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = "Passw0rd!x";
const PHOTO = fs.readFileSync(path.join(__dirname, "../e2e/fixtures/portrait.jpg"));

async function call(method, route, { token, body, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${route}`, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  return { status: res.status, data: text ? JSON.parse(text) : null };
}

function check(label, condition, detail) {
  if (!condition) throw new Error(`FAILED: ${label}${detail ? ` -> ${JSON.stringify(detail)}` : ""}`);
  console.log(`ok  ${label}`);
}

async function onboard(name, gender) {
  const email = `${name.toLowerCase()}_${Date.now()}_${Math.floor(Math.random() * 1e5)}@smoke.test`;
  const reg = await call("POST", "/auth/register", { body: { email, password: PASSWORD } });
  check(`register ${name}`, reg.status === 201, reg);
  const token = reg.data.accessToken;
  const birthdate = new Date(Date.now() - 30 * 365.25 * 864e5).toISOString().slice(0, 10);
  const profile = await call("PUT", "/me/profile", { token, body: { firstName: name, birthdate, gender, bio: "smoke", interests: ["travel"] } });
  check(`profile ${name}`, profile.status === 200, profile);
  const form = new FormData();
  form.append("file", new Blob([PHOTO], { type: "image/jpeg" }), "p.jpg");
  const photo = await call("POST", "/me/photos", { token, form });
  check(`photo ${name}`, photo.status === 201, photo);
  return { id: reg.data.user.id, token };
}

(async () => {
  const health = await call("GET", "/health");
  check("health", health.status === 200, health);

  const a = await onboard("Alice", "woman");
  const b = await onboard("Bob", "man");

  const deck = await call("GET", "/discover?limit=20", { token: a.token });
  check(
    "discover shows Bob",
    deck.data.profiles.some((p) => p.id === b.id),
    deck
  );

  const like1 = await call("POST", "/swipes", { token: a.token, body: { userId: b.id, action: "like" } });
  check("first like does not match", like1.data.matched === false, like1);
  const like2 = await call("POST", "/swipes", { token: b.token, body: { userId: a.id, action: "like" } });
  check("mutual like matches", like2.data.matched === true && like2.data.conversationId, like2);

  const sent = await call("POST", `/conversations/${like2.data.conversationId}/messages`, { token: b.token, body: { body: "Salut !" } });
  check("send message", sent.status === 201, sent);
  const list = await call("GET", "/conversations", { token: a.token });
  check("recipient sees unread message", list.data.conversations[0]?.unreadCount === 1, list);
  const notes = await call("GET", "/notifications", { token: a.token });
  check(
    "match notification",
    notes.data.notifications.some((n) => n.type === "match"),
    notes
  );

  console.log("\nSmoke test passed.");
})().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
