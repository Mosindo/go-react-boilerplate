/* eslint-disable no-console */
// API smoke test against a running server (no UI). Exercises the whole product flow:
//   register -> profile -> photo -> discover -> like -> match -> chat -> notifications -> block -> delete
//
// Usage: SMOKE_API_URL=http://<LAN_IP>:18080 npm run e2e:smoke
const API = process.env.SMOKE_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = "Password123";
const JPEG = Buffer.from("/9j/2wCEAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9PjsBCgsLDg0OHBAQHDsoIig7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O//AABEIAFAAQAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/AGFaaVqUrTStbpn08mRFaaVqYrTStUmYSZCVppWpitNK1SZjJkJWmlamK00rVJmMmaJWmlamK00rXKmdsmQlaaVqYrTStUmYyZCVpCtSlaaVqkzCTIitNK1KVppWqTMZM0itNK1KVppWuVM7pMiK00rUxWmlapMwkyErTStTFaaVqkzGTIStNK1MVppWqTMZM0StNK1MVppWuRM7pMhK00rUxWmlapMwkyErSFalK00rVpmMmRFaaVqUrTStUmYSZpFaaVqUrTStciZ3SZEVppWpitNK1SZjJkJWmlamK00rVJmEmQlaaVqYrTStUmYyZ//Z", "base64");

let failures = 0;
function check(name, condition, detail = "") {
  console.log(`${condition ? "  ✓" : "  ✗"} ${name}${condition || !detail ? "" : ` — ${detail}`}`);
  if (!condition) failures += 1;
}

async function call(method, route, token, body) {
  const res = await fetch(`${API}${route}`, {
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

async function member(label, gender, interestedIn, birthDate) {
  const email = `${label}-${Date.now()}-${Math.floor(Math.random() * 1e6)}@smoke.invalid`;
  const reg = await call("POST", "/auth/register", null, { email, password: PASSWORD });
  check(`register ${label}`, reg.status === 201, JSON.stringify(reg.json));
  const token = reg.json.accessToken;
  const profile = await call("PUT", "/me/profile", token, { firstName: label, birthDate, gender, bio: "smoke", occupation: "", city: "Paris", interests: ["music"] });
  check(`profile ${label}`, profile.status === 200, JSON.stringify(profile.json));
  await call("PUT", "/me/preferences", token, { interestedIn, ageMin: 18, ageMax: 99, maxDistanceKm: 20000 });
  await call("PUT", "/me/location", token, { latitude: 48.8566, longitude: 2.3522 });
  const form = new FormData();
  form.append("file", new Blob([JPEG], { type: "image/jpeg" }), "p.jpg");
  const photo = await fetch(`${API}/me/photos`, { method: "POST", headers: { Authorization: `Bearer ${token}` }, body: form });
  check(`photo ${label}`, photo.status === 201, String(photo.status));
  return { token, id: reg.json.user.id, email };
}

async function main() {
  console.log(`Smoke test against ${API}`);
  const health = await call("GET", "/health");
  check("health", health.status === 200);

  const ana = await member("Ana", "woman", ["man"], "1994-03-03");
  const ben = await member("Ben", "man", ["woman"], "1992-04-04");

  const me = await call("GET", "/me/profile", ana.token);
  check("onboarding complete", me.json?.onboardingComplete === true, JSON.stringify(me.json?.missing));

  const feed = await call("GET", "/discover?limit=20", ana.token);
  check("ana discovers ben", feed.json?.items?.some((c) => c.userId === ben.id));
  check("no exact location leaked", !JSON.stringify(feed.json).match(/latitude|longitude|birthDate|email/));

  const like1 = await call("POST", "/swipes", ana.token, { targetId: ben.id, action: "like" });
  check("one-sided like does not match", like1.json?.matched === false);
  const dup = await call("POST", "/swipes", ana.token, { targetId: ben.id, action: "like" });
  check("duplicate like is idempotent", dup.status === 200 && dup.json?.alreadyProcessed === true);
  const like2 = await call("POST", "/swipes", ben.token, { targetId: ana.id, action: "like" });
  check("mutual like matches", like2.json?.matched === true && !!like2.json?.matchId);
  const matchId = like2.json?.matchId;

  const sent = await call("POST", `/matches/${matchId}/messages`, ana.token, { body: "Bonjour Ben" });
  check("send message", sent.status === 201);
  const stranger = await member("Eve", "woman", ["man"], "1993-01-01");
  const spy = await call("GET", `/matches/${matchId}/messages`, stranger.token);
  check("stranger cannot read the conversation", spy.status === 404);
  const list = await call("GET", "/matches", ben.token);
  check("ben sees 1 unread", list.json?.items?.[0]?.unreadCount === 1);
  await call("POST", `/matches/${matchId}/read`, ben.token);
  const after = await call("GET", "/matches", ben.token);
  check("read resets unread", after.json?.items?.[0]?.unreadCount === 0);

  const notifications = await call("GET", "/notifications", ben.token);
  check("notifications exist", (notifications.json?.items?.length ?? 0) >= 1);

  const block = await call("POST", "/blocks", ben.token, { userId: ana.id });
  check("block", block.status === 204);
  const gone = await call("GET", `/matches/${matchId}/messages`, ana.token);
  check("blocked conversation is closed", gone.status === 404);

  for (const [label, u] of [["Ana", ana], ["Ben", ben], ["Eve", stranger]]) {
    const del = await call("DELETE", "/me", u.token, { password: PASSWORD });
    check(`delete account ${label}`, del.status === 204);
  }

  if (failures > 0) {
    console.error(`${failures} check(s) failed`);
    process.exit(1);
  }
  console.log("SMOKE OK");
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
