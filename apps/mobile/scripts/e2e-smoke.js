/* End-to-end API journey against a live server (docs/API.md). Plain Node 18+, no dependencies.
 *
 *   MOBILE_E2E_API_URL=http://localhost:18080 node scripts/e2e-smoke.js
 */
const zlib = require("node:zlib");

const BASE_URL = (process.env.MOBILE_E2E_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "e2e-password-1";
const runId = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;

let step = 0;

function log(message) {
  console.log(`[e2e] ${message}`);
}

function check(condition, message) {
  step += 1;
  if (!condition) {
    throw new Error(`#${step} FAILED: ${message}`);
  }
  log(`ok ${String(step).padStart(2, "0")} ${message}`);
}

async function call(method, path, { token, body, form, expect } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) {
    payload = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const response = await fetch(`${BASE_URL}${path}`, { method, headers, body: payload });
  const text = await response.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }
  if (expect !== undefined && response.status !== expect) {
    throw new Error(`${method} ${path} -> ${response.status} (expected ${expect}): ${text}`);
  }
  return { status: response.status, data };
}

/** Builds a valid solid-colour PNG (RGB, 32x32) without any dependency. */
function makePng(seed) {
  const size = 32;
  const crcTable = Array.from({ length: 256 }, (_unused, n) => {
    let c = n;
    for (let k = 0; k < 8; k += 1) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc32 = (buffer) => {
    let c = 0xffffffff;
    for (const byte of buffer) c = crcTable[(c ^ byte) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (type, data) => {
    const length = Buffer.alloc(4);
    length.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
    const crc = Buffer.alloc(4);
    crc.writeUInt32BE(crc32(body));
    return Buffer.concat([length, body, crc]);
  };
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const rows = [];
  for (let y = 0; y < size; y += 1) {
    const row = Buffer.alloc(1 + size * 3);
    for (let x = 0; x < size; x += 1) {
      row[1 + x * 3] = (seed * 60 + x * 4) & 0xff;
      row[2 + x * 3] = (y * 6) & 0xff;
      row[3 + x * 3] = 160;
    }
    rows.push(row);
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", zlib.deflateSync(Buffer.concat(rows))),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function createUser(label, seed, gender) {
  const email = `e2e.${label}.${runId}@e2e.invalid`;
  const { data: auth } = await call("POST", "/auth/register", { body: { email, password: PASSWORD }, expect: 201 });
  const token = auth.accessToken;
  check(Boolean(token && auth.refreshToken && auth.user.id), `${label}: registered (${email})`);

  const interests = (await call("GET", "/interests", { token, expect: 200 })).data.items;
  check(interests.length > 0, `${label}: interest catalog is not empty`);

  await call("PUT", "/profile", {
    token,
    expect: 200,
    body: {
      firstName: `E2E${label}`,
      birthDate: "1992-04-12",
      gender,
      bio: `Compte de test ${runId}`,
      interests: interests.slice(0, 3).map((interest) => interest.slug),
      discoverable: true,
      showDistance: true
    }
  });
  await call("PUT", "/preferences", {
    token,
    expect: 200,
    body: { interestedIn: ["man", "woman", "nonbinary"], minAge: 18, maxAge: 99, maxDistanceKm: null }
  });

  const form = new FormData();
  form.append("file", new Blob([makePng(seed)], { type: "image/png" }), "photo.png");
  const photo = (await call("POST", "/photos", { token, form, expect: 201 })).data;
  check(photo.isPrimary && photo.position === 0, `${label}: photo uploaded as primary`);

  const image = await fetch(`${BASE_URL}${photo.thumbUrl}`, { headers: { Authorization: `Bearer ${token}` } });
  check(
    image.status === 200 && image.headers.get("content-type").includes("image/jpeg"),
    `${label}: thumbnail served with the Bearer header`
  );
  const anonymous = await fetch(`${BASE_URL}${photo.thumbUrl}`);
  check(anonymous.status === 401, `${label}: thumbnail refused without credentials`);

  const state = (await call("GET", "/profile", { token, expect: 200 })).data;
  check(state.complete === true, `${label}: profile is complete`);
  return { label, email, token, refreshToken: auth.refreshToken, id: auth.user.id };
}

async function main() {
  log(`API: ${BASE_URL}`);
  const health = await call("GET", "/health", { expect: 200 });
  check(health.data.status === "ok", "health is ok");

  const underage = await call("POST", "/auth/register", {
    body: { email: `young.${runId}@e2e.invalid`, password: PASSWORD }
  });
  const youngToken = underage.data.accessToken;
  const rejected = await call("PUT", "/profile", {
    token: youngToken,
    body: {
      firstName: "Young",
      birthDate: new Date(Date.now() - 16 * 365 * 86400000).toISOString().slice(0, 10),
      gender: "man",
      bio: "",
      interests: [],
      discoverable: true,
      showDistance: true
    }
  });
  check(rejected.status === 400 && rejected.data.code === "underage", "under-18 profile is rejected");
  await call("DELETE", "/me", { token: youngToken, body: { password: PASSWORD }, expect: 204 });

  const a = await createUser("A", 1, "woman");
  const b = await createUser("B", 2, "man");

  const discovery = (await call("GET", "/discovery?limit=20", { token: a.token, expect: 200 })).data;
  check(Array.isArray(discovery.items), "A: discovery answers");
  const seesB = discovery.items.some((profile) => profile.userId === b.id);
  log(`    (B ${seesB ? "is" : "is not"} in the first discovery page; swiping by id regardless)`);
  check(
    discovery.items.every((profile) => profile.userId !== a.id),
    "A: never sees themself in discovery"
  );

  const liked = (
    await call("POST", "/discovery/swipes", { token: a.token, body: { userId: b.id, action: "like" }, expect: 200 })
  ).data;
  check(liked.result === "liked" && liked.match === null, "A likes B: liked, no match yet");
  const again = (
    await call("POST", "/discovery/swipes", { token: a.token, body: { userId: b.id, action: "pass" }, expect: 200 })
  ).data;
  check(again.result === "already_swiped", "first swipe is final (already_swiped)");

  const matched = (
    await call("POST", "/discovery/swipes", { token: b.token, body: { userId: a.id, action: "like" }, expect: 200 })
  ).data;
  check(matched.result === "matched" && matched.match && matched.match.conversationId, "B likes A: matched");
  const { matchId, conversationId } = matched.match;

  const matchesA = (await call("GET", "/matches", { token: a.token, expect: 200 })).data;
  check(
    matchesA.items.some((match) => match.matchId === matchId && match.profile.userId === b.id),
    "A sees the match with B"
  );

  const sent = (
    await call("POST", `/conversations/${conversationId}/messages`, {
      token: a.token,
      body: { body: "  Bonjour B !  " },
      expect: 201
    })
  ).data;
  check(sent.body === "Bonjour B !" && sent.readAt === null && sent.senderId === a.id, "A sent a (trimmed) message");

  const unreadB = (await call("GET", "/conversations/unread-count", { token: b.token, expect: 200 })).data.unreadCount;
  check(unreadB === 1, "B has 1 unread message");
  const conversationsB = (await call("GET", "/conversations", { token: b.token, expect: 200 })).data;
  const itemB = conversationsB.items.find((item) => item.id === conversationId);
  check(
    itemB && itemB.unreadCount === 1 && itemB.lastMessage.id === sent.id,
    "B's conversation list shows the unread message"
  );

  const notificationsB = (await call("GET", "/notifications", { token: b.token, expect: 200 })).data;
  check(
    notificationsB.unreadCount >= 1 &&
      notificationsB.items.some((n) => n.type === "message" && n.conversationId === conversationId),
    "B received a message notification"
  );
  check(
    notificationsB.items.some((n) => n.type === "match"),
    "B received a match notification"
  );

  const messagesB = (await call("GET", `/conversations/${conversationId}/messages`, { token: b.token, expect: 200 }))
    .data;
  check(messagesB.items.length === 1 && messagesB.items[0].id === sent.id, "B reads the thread");
  await call("POST", `/conversations/${conversationId}/read`, { token: b.token, expect: 204 });
  const unreadAfter = (await call("GET", "/conversations/unread-count", { token: b.token, expect: 200 })).data
    .unreadCount;
  check(unreadAfter === 0, "unread count back to 0 after mark-read");
  const messagesA = (await call("GET", `/conversations/${conversationId}/messages`, { token: a.token, expect: 200 }))
    .data;
  check(messagesA.items[0].readAt !== null, "A sees the read receipt");

  await call("POST", "/notifications/read-all", { token: b.token, expect: 204 });
  const notificationsAfter = (await call("GET", "/notifications", { token: b.token, expect: 200 })).data;
  check(notificationsAfter.unreadCount === 0, "notifications all read");

  const refreshed = (await call("POST", "/auth/refresh", { body: { refreshToken: a.refreshToken }, expect: 200 })).data;
  check(refreshed.accessToken && refreshed.refreshToken !== a.refreshToken, "refresh rotates the refresh token");
  const replay = await call("POST", "/auth/refresh", { body: { refreshToken: a.refreshToken } });
  check(replay.status === 401, "a used refresh token cannot be replayed");
  a.token = refreshed.accessToken;

  await call("POST", "/blocks", { token: b.token, body: { userId: a.id }, expect: 204 });
  const goneB = (await call("GET", "/conversations", { token: b.token, expect: 200 })).data;
  const goneA = (await call("GET", "/conversations", { token: a.token, expect: 200 })).data;
  check(
    !goneB.items.some((item) => item.id === conversationId) && !goneA.items.some((item) => item.id === conversationId),
    "block removes the conversation for both"
  );
  const matchesAfter = (await call("GET", "/matches", { token: a.token, expect: 200 })).data;
  check(!matchesAfter.items.some((match) => match.matchId === matchId), "block removes the match");
  const hidden = await call("GET", `/profiles/${a.id}`, { token: b.token });
  check(hidden.status === 404, "blocked profile is not visible");
  const blocks = (await call("GET", "/blocks", { token: b.token, expect: 200 })).data;
  check(
    blocks.items.some((entry) => entry.userId === a.id),
    "B lists A as blocked"
  );
  await call("DELETE", `/blocks/${a.id}`, { token: b.token, expect: 204 });

  await call("DELETE", "/me", { token: a.token, body: { password: PASSWORD }, expect: 204 });
  await call("DELETE", "/me", { token: b.token, body: { password: PASSWORD }, expect: 204 });
  check(true, "test accounts deleted");

  log(`SUCCESS (${step} checks)`);
}

main().catch((error) => {
  console.error(`[e2e] ${error.message}`);
  process.exit(1);
});
