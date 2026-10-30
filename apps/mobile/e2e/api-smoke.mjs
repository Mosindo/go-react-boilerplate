// API smoke test: run it against a live API to confirm the stack is ready.
//   register -> profile -> photo -> discovery -> like -> match -> chat -> notifications -> cleanup
// Usage: API_URL=http://<LAN_IP>:18080 npm run e2e:api
import { solidPng } from "./png.mjs";

const API_URL = (process.env.API_URL ?? process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:18080").replace(
  /\/+$/,
  ""
);
const PASSWORD = "CorrectHorse42";
const run = Date.now();
const LOCATION = {
  latitude: -50 + ((run % 100000) / 100000) * 100,
  longitude: -170 + ((Math.floor(run / 100000) % 100000) / 100000) * 340
};

async function call(path, { method = "GET", body, token, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_URL}${path}`, { method, headers, body: body ? JSON.stringify(body) : form });
  const text = await response.text();
  return { status: response.status, data: text ? JSON.parse(text) : undefined };
}

function expect(response, status, label) {
  if (response.status !== status) {
    throw new Error(`${label}: expected ${status}, got ${response.status} ${JSON.stringify(response.data)}`);
  }
  console.log(`✔ ${label}`);
  return response.data;
}

async function createUser(name, gender, interestedIn) {
  const email = `${name.toLowerCase()}-${run}@smoke.invalid`;
  const auth = expect(
    await call("/auth/register", { method: "POST", body: { email, password: PASSWORD } }),
    201,
    `register ${name}`
  );
  const token = auth.accessToken;
  expect(
    await call("/profile", {
      method: "PUT",
      token,
      body: {
        displayName: name,
        birthDate: "1990-01-01",
        gender,
        bio: "Smoke test",
        city: "Test",
        ...LOCATION,
        interests: ["voyage"]
      }
    }),
    200,
    `${name}: save profile`
  );
  expect(
    await call("/preferences", {
      method: "PUT",
      token,
      body: { interestedIn: [interestedIn], minAge: 18, maxAge: 60, maxDistanceKm: 50 }
    }),
    200,
    `${name}: save preferences`
  );
  const form = new FormData();
  form.append("file", new Blob([solidPng(60, 80, [120, 60, 200])], { type: "image/png" }), "smoke.png");
  expect(await call("/photos", { method: "POST", token, form }), 201, `${name}: upload photo`);
  return { id: auth.user.id, token };
}

const health = await call("/health");
expect(health, 200, `API healthy at ${API_URL}`);

const alice = await createUser("Alice", "woman", "man");
const bob = await createUser("Bob", "man", "woman");

const aliceFeed = expect(await call("/discover?limit=5", { token: alice.token }), 200, "Alice discovers profiles");
if (!aliceFeed.profiles.some((p) => p.userId === bob.id)) throw new Error("Bob missing from Alice's discovery");

const first = expect(
  await call("/swipes", { method: "POST", token: alice.token, body: { targetUserId: bob.id, action: "like" } }),
  200,
  "Alice likes Bob"
);
if (first.matched) throw new Error("unexpected match after a single like");
const second = expect(
  await call("/swipes", { method: "POST", token: bob.token, body: { targetUserId: alice.id, action: "like" } }),
  200,
  "Bob likes Alice"
);
if (!second.matched) throw new Error("expected a match");
const conversationId = second.match.conversationId;

expect(
  await call(`/conversations/${conversationId}/messages`, {
    method: "POST",
    token: alice.token,
    body: { body: "Bonjour Bob" }
  }),
  201,
  "Alice sends a message"
);
const inbox = expect(await call("/conversations", { token: bob.token }), 200, "Bob lists conversations");
if (inbox.conversations[0]?.unreadCount !== 1) throw new Error("expected one unread message for Bob");
expect(
  await call(`/conversations/${conversationId}/read`, { method: "POST", token: bob.token }),
  204,
  "Bob marks it read"
);
const notifications = expect(await call("/notifications", { token: bob.token }), 200, "Bob reads notifications");
if (!notifications.notifications.some((n) => n.type === "match")) throw new Error("match notification missing");

for (const [name, user] of [
  ["Alice", alice],
  ["Bob", bob]
]) {
  expect(
    await call("/me", { method: "DELETE", token: user.token, body: { password: PASSWORD } }),
    204,
    `${name}: delete account (cleanup)`
  );
}
console.log("\n✔ API smoke test passed");
