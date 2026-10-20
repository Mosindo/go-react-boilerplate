#!/usr/bin/env node
// End-to-end API smoke test against a running stack seeded with `./seed`.
//   API_URL=http://localhost:18080 node scripts/smoke-api.js
// Journey: register -> login -> /me -> discover -> like -> match -> chat ->
// read receipts -> notifications -> block -> account deletion.
const API = (process.env.API_URL || "http://localhost:18080").replace(/\/$/, "");
const PASSWORD = "demo-password-123";

let failures = 0;
function check(name, condition, detail = "") {
  if (condition) {
    console.log(`  ok   ${name}`);
  } else {
    failures += 1;
    console.error(`  FAIL ${name} ${detail}`);
  }
}

async function call(method, path, { token, body } = {}) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined
  });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = null;
  }
  return { status: res.status, json };
}

async function login(slug) {
  const res = await call("POST", "/auth/login", { body: { email: `demo-${slug}@demo.invalid`, password: PASSWORD } });
  if (res.status !== 200) throw new Error(`login ${slug} failed (${res.status}); did you run ./seed?`);
  return { token: res.json.accessToken, id: res.json.user.id };
}

async function main() {
  console.log(`Smoke testing ${API}`);
  const health = await call("GET", "/health");
  check("health", health.status === 200);

  // Registration + auth
  const email = `smoke-${Date.now()}@example.org`;
  const reg = await call("POST", "/auth/register", { body: { email, password: "Smoke-pass-123" } });
  check("register", reg.status === 201 && reg.json.user.profileComplete === false);
  const me = await call("GET", "/me", { token: reg.json.accessToken });
  check("me", me.status === 200 && me.json.email === email);
  check("anonymous /me is rejected", (await call("GET", "/me")).status === 401);

  // Matching + chat between two demo accounts
  const hugo = await login("hugo");
  const ines = await login("ines");
  const discover = await call("GET", "/discover?limit=20", { token: hugo.token });
  check("discover returns ines", discover.status === 200 && discover.json.profiles.some((p) => p.userId === ines.id));
  check("discover hides exact location", !JSON.stringify(discover.json).match(/latitude|longitude/));

  const first = await call("POST", "/swipes", { token: hugo.token, body: { targetUserId: ines.id, action: "like" } });
  check("first like", first.status === 200 || first.status === 409);
  const second = await call("POST", "/swipes", { token: ines.token, body: { targetUserId: hugo.id, action: "like" } });
  check("reciprocal like", second.status === 200 || second.status === 409);

  const matches = await call("GET", "/matches", { token: ines.token });
  const match = matches.json.matches.find((m) => m.user.userId === hugo.id);
  check("match exists", Boolean(match));
  if (match) {
    const sent = await call("POST", `/conversations/${match.conversationId}/messages`, { token: hugo.token, body: { body: "Hello from the smoke test" } });
    check("send message", sent.status === 201);
    const convs = await call("GET", "/conversations", { token: ines.token });
    const conv = convs.json.conversations.find((c) => c.id === match.conversationId);
    check("recipient sees unread message", conv && conv.unreadCount >= 1);
    check("read", (await call("POST", `/conversations/${match.conversationId}/read`, { token: ines.token })).status === 204);
    const thread = await call("GET", `/conversations/${match.conversationId}/messages`, { token: hugo.token });
    check("read receipt", thread.json.otherLastReadAt !== null);
    const outsider = await call("GET", `/conversations/${match.conversationId}/messages`, { token: reg.json.accessToken });
    check("outsider cannot read conversation", outsider.status === 404);
  }
  const notes = await call("GET", "/notifications", { token: ines.token });
  check("notifications", notes.status === 200 && notes.json.notifications.length > 0);

  // Safety + account deletion
  check("block", (await call("POST", "/blocks", { token: reg.json.accessToken, body: { userId: hugo.id } })).status === 204);
  check("report", (await call("POST", "/reports", { token: reg.json.accessToken, body: { userId: hugo.id, reason: "spam" } })).status === 201);
  check("delete account", (await call("DELETE", "/me", { token: reg.json.accessToken, body: { password: "Smoke-pass-123" } })).status === 204);
  check("deleted account cannot login", (await call("POST", "/auth/login", { body: { email, password: "Smoke-pass-123" } })).status === 401);

  if (failures > 0) {
    console.error(`\n${failures} check(s) failed`);
    process.exit(1);
  }
  console.log("\nSmoke test passed");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
