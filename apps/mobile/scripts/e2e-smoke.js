/**
 * API smoke test of the critical dating flow against a running API:
 * register -> profile -> preferences -> photo -> discover -> like -> match
 * -> chat -> read receipt -> notifications -> block.
 *
 *   MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
 */
const { gradientPng: png } = require("./lib/png");

const API = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Password123";

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function call(method, path, token, body) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
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

async function expectStatus(promise, status, label) {
  const res = await promise;
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`);
  return res.data;
}

async function createMember(name, gender, interestedIn, rgb) {
  const email = `smoke.${name.toLowerCase()}.${Date.now()}.${Math.floor(Math.random() * 1e6)}@smoke.test`;
  const auth = await expectStatus(call("POST", "/auth/register", null, { email, password: PASSWORD }), 201, `register ${name}`);
  const token = auth.accessToken;
  await expectStatus(
    call("PUT", "/me/profile", token, {
      firstName: name,
      birthdate: "1993-04-12",
      gender,
      bio: `Smoke test ${name}`,
      lookingFor: "relationship",
      interestIds: [1, 2]
    }),
    200,
    `profile ${name}`
  );
  await expectStatus(call("PUT", "/me/preferences", token, { interestedIn, ageMin: 18, ageMax: 99, maxDistanceKm: 100 }), 200, `preferences ${name}`);
  await expectStatus(call("PUT", "/me/location", token, { latitude: 48.85, longitude: 2.35 }), 204, `location ${name}`);
  const form = new FormData();
  form.append("photo", new Blob([png(320, 400, rgb)], { type: "image/png" }), "photo.png");
  await expectStatus(call("POST", "/me/photos", token, form), 201, `photo ${name}`);
  return { name, email, token, id: auth.user.id };
}

async function main() {
  console.log(`[smoke] API ${API}`);
  await expectStatus(call("GET", "/health"), 200, "health");

  const a = await createMember("Alice", "woman", ["man"], [240, 138, 93]);
  const b = await createMember("Bruno", "man", ["woman"], [120, 92, 150]);
  console.log("[smoke] members created");

  const deck = await expectStatus(call("GET", "/discover?limit=30", a.token), 200, "discover");
  assert(
    deck.profiles.some((p) => p.userId === b.id),
    "Bruno should appear in Alice's deck"
  );
  assert(!JSON.stringify(deck).includes("latitude"), "deck must not leak coordinates");

  const first = await expectStatus(call("POST", "/swipes", a.token, { targetUserId: b.id, action: "like" }), 200, "like A->B");
  assert(!first.matched, "one-sided like must not match");
  const second = await expectStatus(call("POST", "/swipes", b.token, { targetUserId: a.id, action: "like" }), 200, "like B->A");
  assert(second.matched && second.match, "mutual like must match");
  const matchId = second.match.id;
  console.log(`[smoke] match ${matchId}`);

  await expectStatus(call("POST", `/matches/${matchId}/messages`, a.token, { body: "Salut Bruno !" }), 201, "send");
  const page = await expectStatus(call("GET", `/matches/${matchId}/messages`, b.token), 200, "read messages");
  assert(page.messages[0].body === "Salut Bruno !", "message must be delivered");
  await expectStatus(call("POST", `/matches/${matchId}/read`, b.token), 204, "mark read");

  const notifs = await expectStatus(call("GET", "/notifications", b.token), 200, "notifications");
  assert(
    notifs.notifications.some((n) => n.type === "match"),
    "match notification expected"
  );

  await expectStatus(call("POST", "/blocks", a.token, { userId: b.id }), 204, "block");
  await expectStatus(call("GET", `/matches/${matchId}/messages`, b.token), 404, "blocked chat");

  for (const m of [a, b]) {
    await expectStatus(call("DELETE", "/me", m.token, { password: PASSWORD }), 204, `delete ${m.name}`);
  }
  console.log("[smoke] OK");
}

main().catch((err) => {
  console.error(`[smoke] FAILED: ${err.message}`);
  process.exit(1);
});
