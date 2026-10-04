/**
 * End-to-end API smoke test (plain Node 18+, real HTTP, no mocks).
 * register 2 users -> profiles + photos -> like both ways -> match -> message -> read -> notifications.
 *
 * Usage: EXPO_PUBLIC_API_URL=http://localhost:18080 npm run e2e:smoke
 * Auth endpoints are rate limited per IP, so this script only registers two users per run.
 */
const zlib = require("zlib");

const BASE = (process.env.EXPO_PUBLIC_API_URL || process.env.API_URL || "http://localhost:18080").replace(/\/+$/, "");
const run = Date.now().toString(36);
const PASSWORD = "Sm0ke-test-pass!";

function crc32(buf) {
  let c;
  let crc = 0xffffffff;
  for (let n = 0; n < buf.length; n++) {
    c = (crc ^ buf[n]) & 0xff;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    crc = (crc >>> 8) ^ c;
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}

/** Generates a small solid-colour PNG so the script needs no fixture files. */
function makePng(r, g, b, size = 256) {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const row = Buffer.concat([Buffer.from([0]), Buffer.from(Array.from({ length: size }, () => [r, g, b]).flat())]);
  const raw = Buffer.concat(Array.from({ length: size }, () => row));
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function call(token, method, path, body, form) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: form ?? (body === undefined ? undefined : JSON.stringify(body))
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status} ${text}`);
  return data;
}

function check(label, condition) {
  if (!condition) throw new Error(`Check failed: ${label}`);
  console.log(`ok  ${label}`);
}

async function createUser(name, gender, interestedIn, color) {
  const auth = await call(null, "POST", "/auth/register", { email: `${name}.${run}@test.invalid`, password: PASSWORD });
  const token = auth.accessToken;
  await call(token, "PUT", "/profile", {
    displayName: name,
    birthDate: "1994-05-17",
    gender,
    bio: `Hi, I am ${name}.`,
    city: "Lisbon",
    interests: [],
    showDistance: true,
    discoverable: true
  });
  await call(token, "PUT", "/preferences", { interestedIn, minAge: 18, maxAge: 60, maxDistanceKm: 500 });
  const form = new FormData();
  form.append("file", new Blob([makePng(...color)], { type: "image/png" }), "photo.png");
  const photo = await call(token, "POST", "/profile/photos", undefined, form);
  return { name, auth, token, photo, id: auth.user.id };
}

async function main() {
  console.log(`API: ${BASE}`);
  check("health", (await call(null, "GET", "/health")).status === "ok");

  const a = await createUser(`Alice${run}`, "woman", ["man"], [200, 80, 60]);
  const b = await createUser(`Bob${run}`, "man", ["woman"], [60, 120, 200]);
  check(
    "both profiles are completed",
    (await call(a.token, "GET", "/profile")).completed && (await call(b.token, "GET", "/profile")).completed
  );

  const photoRes = await fetch(`${BASE}${a.photo.thumbUrl}`, { headers: { Authorization: `Bearer ${b.token}` } });
  check(
    "photo requires auth and loads with a bearer token",
    photoRes.ok && (photoRes.headers.get("content-type") || "").includes("image")
  );
  check("photo without token is rejected", (await fetch(`${BASE}${a.photo.thumbUrl}`)).status === 401);

  const feed = await call(a.token, "GET", "/discover?limit=20");
  check(
    "alice discovers bob",
    feed.profiles.some((p) => p.userId === b.id)
  );

  const first = await call(a.token, "POST", "/swipes", { targetId: b.id, action: "like" });
  check("first like does not match yet", first.matched === false);
  const second = await call(b.token, "POST", "/swipes", { targetId: a.id, action: "like" });
  check("reciprocal like creates a match", second.matched === true && second.match);
  const again = await call(b.token, "POST", "/swipes", { targetId: a.id, action: "like" });
  check("swipes are idempotent", again.matched === true);

  const matches = await call(a.token, "GET", "/matches?limit=10");
  const match = matches.matches.find((m) => m.user.userId === b.id);
  check("alice sees the match", !!match);

  const sent = await call(a.token, "POST", `/conversations/${match.conversationId}/messages`, { body: "Hello Bob!" });
  check("message is sent", sent.body === "Hello Bob!");
  const unread = await call(b.token, "GET", "/conversations/unread-count");
  check("bob has 1 unread", unread.unreadCount === 1);
  const convs = await call(b.token, "GET", "/conversations?limit=10");
  check(
    "bob sees the conversation",
    convs.conversations.some((c) => c.conversationId === match.conversationId && c.unreadCount === 1)
  );
  await call(b.token, "POST", `/conversations/${match.conversationId}/read`);
  const messages = await call(a.token, "GET", `/conversations/${match.conversationId}/messages?limit=20`);
  check("alice's message shows as read", messages.messages[0].readAt !== null);
  check("bob unread is back to 0", (await call(b.token, "GET", "/conversations/unread-count")).unreadCount === 0);

  const notes = await call(a.token, "GET", "/notifications?limit=20");
  check(
    "alice has a match notification",
    notes.notifications.some((n) => n.type === "match")
  );
  const noteCount = await call(a.token, "GET", "/notifications/unread-count");
  check("unread notification count > 0", noteCount.unreadCount > 0);
  await call(a.token, "POST", "/notifications/read-all");
  check("read-all clears unread", (await call(a.token, "GET", "/notifications/unread-count")).unreadCount === 0);

  const refreshed = await call(null, "POST", "/auth/refresh", { refreshToken: a.auth.refreshToken });
  check("refresh rotates tokens", refreshed.accessToken && refreshed.refreshToken !== a.auth.refreshToken);

  await call(a.token, "DELETE", `/matches/${match.id}`);
  check(
    "unmatch removes the match",
    !(await call(b.token, "GET", "/matches?limit=10")).matches.some((m) => m.id === match.id)
  );

  await call(refreshed.accessToken, "DELETE", "/me", { password: PASSWORD });
  await call(b.token, "DELETE", "/me", { password: PASSWORD });
  console.log("\nSmoke test passed.");
}

main().catch((error) => {
  console.error(`\nSmoke test FAILED: ${error.message}`);
  process.exit(1);
});
