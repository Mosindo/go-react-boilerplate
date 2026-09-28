// Critical-flow smoke test against a running API (see docs/API.md). Node 20+, no dependencies.
//   API_URL=http://localhost:18080 node scripts/e2e-smoke.js
// Falls back to EXPO_PUBLIC_API_URL, then http://localhost:18080. Exits non-zero on failure.

const API_URL = (process.env.API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(
  /\/+$/,
  ""
);
const PASSWORD = "Password123!";
const PARIS = { latitude: 48.8566, longitude: 2.3522, label: "Paris" };

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@lumen.test`;
}

// --- Minimal baseline JPEG generator (grayscale, flat 8x8 blocks) -------------------------------

function buildJpeg(width, height) {
  const bytes = [];
  const push = (...values) => bytes.push(...values);
  const word = (value) => push((value >> 8) & 0xff, value & 0xff);

  push(0xff, 0xd8); // SOI
  push(0xff, 0xdb); // DQT: all 8s
  word(67);
  push(0x00, ...new Array(64).fill(8));
  push(0xff, 0xc0); // SOF0: 8 bit, 1 component
  word(11);
  push(8);
  word(height);
  word(width);
  push(1, 1, 0x11, 0);
  // DHT DC: twelve 4-bit codes (categories 0..11)
  push(0xff, 0xc4);
  word(2 + 1 + 16 + 12);
  push(0x00, 0, 0, 0, 12, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0);
  for (let symbol = 0; symbol < 12; symbol += 1) push(symbol);
  // DHT AC: a single 1-bit code for end-of-block
  push(0xff, 0xc4);
  word(2 + 1 + 16 + 1);
  push(0x10, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0);
  push(0x00);
  push(0xff, 0xda); // SOS
  word(8);
  push(1, 1, 0x00, 0, 63, 0);

  let bitBuffer = 0;
  let bitCount = 0;
  const writeBits = (value, length) => {
    for (let index = length - 1; index >= 0; index -= 1) {
      bitBuffer = (bitBuffer << 1) | ((value >> index) & 1);
      bitCount += 1;
      if (bitCount === 8) {
        push(bitBuffer);
        if (bitBuffer === 0xff) push(0x00);
        bitBuffer = 0;
        bitCount = 0;
      }
    }
  };

  let previousDc = 0;
  const blocksX = Math.ceil(width / 8);
  const blocksY = Math.ceil(height / 8);
  for (let by = 0; by < blocksY; by += 1) {
    for (let bx = 0; bx < blocksX; bx += 1) {
      const level = 60 + ((bx * 37 + by * 61) % 140); // varied gray levels
      const dc = level - 128;
      const diff = dc - previousDc;
      previousDc = dc;
      const magnitude = Math.abs(diff);
      const category = magnitude === 0 ? 0 : Math.floor(Math.log2(magnitude)) + 1;
      writeBits(category, 4);
      if (category > 0) {
        writeBits(diff > 0 ? diff : diff + (1 << category) - 1, category);
      }
      writeBits(0, 1); // end of block
    }
  }
  if (bitCount > 0) writeBits(0xff, 8 - bitCount);
  push(0xff, 0xd9); // EOI
  return Buffer.from(bytes);
}

// --- HTTP helpers ----------------------------------------------------------------------------------

async function request(method, path, { token, body, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) {
    payload = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const response = await fetch(`${API_URL}${path}`, { method, headers, body: payload });
  const contentType = response.headers.get("content-type") || "";
  let data = null;
  if (contentType.includes("application/json")) {
    data = await response.json().catch(() => null);
  } else {
    await response.arrayBuffer().catch(() => null);
  }
  return { status: response.status, data };
}

async function expectStatus(expected, method, path, options) {
  const result = await request(method, path, options);
  assert(
    result.status === expected,
    `${method} ${path} expected ${expected}, got ${result.status} ${JSON.stringify(result.data)}`
  );
  return result.data;
}

let stepCount = 0;
async function step(name, fn) {
  stepCount += 1;
  try {
    const result = await fn();
    console.log(`  ok ${String(stepCount).padStart(2, "0")} ${name}`);
    return result;
  } catch (error) {
    throw new Error(`step "${name}" failed: ${error.message}`);
  }
}

async function createUser(label, gender, interestedIn, interestIds) {
  const email = uniqueEmail(`e2e_${label}`);
  const session = await expectStatus(201, "POST", "/auth/register", { body: { email, password: PASSWORD } });
  assert(session.accessToken && session.refreshToken, "register must return accessToken and refreshToken");
  const token = session.accessToken;

  await expectStatus(200, "PUT", "/me/profile", {
    token,
    body: {
      firstName: `E2E${label}`,
      birthDate: "1994-04-12",
      gender,
      bio: `Smoke test profile ${label}`,
      interestIds
    }
  });
  await expectStatus(200, "PUT", "/me/location", { token, body: PARIS });
  await expectStatus(200, "PUT", "/me/preferences", {
    token,
    body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 100 }
  });
  const form = new FormData();
  form.append("file", new Blob([buildJpeg(32, 32)], { type: "image/jpeg" }), "smoke.jpg");
  const photo = await expectStatus(201, "POST", "/me/photos", { token, form });
  assert(photo.id && photo.url, "photo upload must return id and url");

  const me = await expectStatus(200, "GET", "/me", { token });
  assert(me.profileComplete === true, "profileComplete should be true after profile, location and a photo");
  return { email, token, refreshToken: session.refreshToken, id: session.user.id, photo, me };
}

function openSocket(token) {
  if (typeof WebSocket === "undefined") return null;
  const events = [];
  const socket = new WebSocket(`${API_URL.replace(/^http/, "ws")}/ws`);
  const ready = new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("websocket did not become ready")), 5000);
    socket.onopen = () => socket.send(JSON.stringify({ type: "auth", token }));
    socket.onmessage = (event) => {
      const frame = JSON.parse(String(event.data));
      if (frame.type === "ready") {
        clearTimeout(timer);
        resolve();
      } else {
        events.push(frame);
      }
    };
    socket.onerror = () => reject(new Error("websocket error"));
  });
  return { ready, events, close: () => socket.close() };
}

async function waitFor(check, message, timeoutMs = 5000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (check()) return;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error(message);
}

async function main() {
  console.log(`[e2e-smoke] API: ${API_URL}`);
  const users = [];
  let socket = null;
  try {
    await step("health", async () => {
      const data = await expectStatus(200, "GET", "/health");
      assert(data.status === "ok", "health status should be ok");
    });

    const interests = await step("interests catalogue", async () => {
      const data = await expectStatus(200, "GET", "/interests");
      assert(Array.isArray(data.items) && data.items.length >= 3, "expected at least 3 interests");
      return data.items;
    });
    const shared = interests.slice(0, 3).map((item) => item.id);

    const a = await step("register A, profile, location, preferences, photo", async () => {
      const user = await createUser("A", "man", ["woman"], shared);
      users.push(user);
      return user;
    });
    const b = await step("register B, profile, location, preferences, photo", async () => {
      const user = await createUser("B", "woman", ["man"], shared);
      users.push(user);
      return user;
    });

    await step("login and refresh rotation", async () => {
      const login = await expectStatus(200, "POST", "/auth/login", { body: { email: a.email, password: PASSWORD } });
      assert(login.user.id === a.id, "login should return the same user");
      const refreshed = await expectStatus(200, "POST", "/auth/refresh", {
        body: { refreshToken: login.refreshToken }
      });
      assert(refreshed.refreshToken && refreshed.refreshToken !== login.refreshToken, "refresh token must rotate");
      const reused = await request("POST", "/auth/refresh", { body: { refreshToken: login.refreshToken } });
      assert(
        reused.status === 401 || reused.status === 400,
        `reusing a rotated refresh token must fail, got ${reused.status}`
      );
    });

    await step("protected photo is served to a permitted viewer only", async () => {
      const anonymous = await request("GET", b.photo.url);
      assert(
        anonymous.status === 401 || anonymous.status === 404,
        `anonymous photo fetch must be refused, got ${anonymous.status}`
      );
      const allowed = await fetch(`${API_URL}${b.photo.url}`, { headers: { Authorization: `Bearer ${a.token}` } });
      assert(
        allowed.status === 200 && (allowed.headers.get("content-type") || "").includes("image/jpeg"),
        "viewer should get a JPEG"
      );
    });

    socket = openSocket(b.token);
    if (socket) {
      await step("websocket auth", () => socket.ready);
    }

    const candidate = await step("A discovers B", async () => {
      const data = await expectStatus(200, "GET", "/discover?limit=20", { token: a.token });
      const found = data.items.find((item) => item.userId === b.id);
      assert(found, "B should appear in A's discover queue");
      assert(found.photos.length === 1, "candidate should expose the photo");
      return found;
    });

    await step("A likes B (no match yet)", async () => {
      const data = await expectStatus(200, "POST", "/swipes", {
        token: a.token,
        body: { targetUserId: candidate.userId, action: "like" }
      });
      assert(data.matched === false, "first like must not match");
      const again = await expectStatus(200, "POST", "/swipes", {
        token: a.token,
        body: { targetUserId: b.id, action: "like" }
      });
      assert(again.matched === false, "repeated swipe is idempotent");
    });

    const match = await step("B likes A (match)", async () => {
      const data = await expectStatus(200, "POST", "/swipes", {
        token: b.token,
        body: { targetUserId: a.id, action: "like" }
      });
      assert(data.matched === true && data.match && data.match.conversationId, "second like must create a match");
      return data.match;
    });

    await step("matches list for both users", async () => {
      for (const user of [a, b]) {
        const data = await expectStatus(200, "GET", "/matches", { token: user.token });
        assert(
          data.items.some((item) => item.matchId === match.matchId),
          "match should be listed"
        );
      }
    });

    const sent = await step("A sends a message", async () => {
      const message = await expectStatus(201, "POST", `/conversations/${match.conversationId}/messages`, {
        token: a.token,
        body: { body: "  Hello from the smoke test  " }
      });
      assert(message.body === "Hello from the smoke test", "body should be trimmed");
      assert(message.readAt === null, "new message is unread");
      return message;
    });

    if (socket) {
      await step("B receives message.new over websocket", () =>
        waitFor(
          () => socket.events.some((frame) => frame.type === "message.new" && frame.data.id === sent.id),
          "message.new not received"
        )
      );
    }

    await step("B sees unread count, reads the thread", async () => {
      const list = await expectStatus(200, "GET", "/matches", { token: b.token });
      assert(list.items.find((item) => item.matchId === match.matchId).unreadCount === 1, "B should have 1 unread");
      const thread = await expectStatus(200, "GET", `/conversations/${match.conversationId}/messages?limit=30`, {
        token: b.token
      });
      assert(
        thread.items.some((item) => item.id === sent.id),
        "B should see the message"
      );
      await expectStatus(204, "POST", `/conversations/${match.conversationId}/read`, { token: b.token });
      const after = await expectStatus(200, "GET", "/matches", { token: b.token });
      assert(after.items.find((item) => item.matchId === match.matchId).unreadCount === 0, "unread should be cleared");
    });

    await step("A sees the read receipt", async () => {
      const thread = await expectStatus(200, "GET", `/conversations/${match.conversationId}/messages`, {
        token: a.token
      });
      const message = thread.items.find((item) => item.id === sent.id);
      assert(message && message.readAt, "message should be marked read for the sender");
    });

    await step("notifications: list, read one, read all", async () => {
      const list = await expectStatus(200, "GET", "/notifications", { token: b.token });
      assert(
        list.items.some((item) => item.type === "match"),
        "B should have a match notification"
      );
      assert(list.unreadCount >= 1, "B should have unread notifications");
      await expectStatus(204, "POST", `/notifications/${list.items[0].id}/read`, { token: b.token });
      await expectStatus(204, "POST", "/notifications/read-all", { token: b.token });
      const after = await expectStatus(200, "GET", "/notifications", { token: b.token });
      assert(after.unreadCount === 0, "unread count should be 0 after read-all");
    });

    await step("A blocks B: hidden everywhere, then unblocks", async () => {
      await expectStatus(204, "POST", "/blocks", { token: a.token, body: { userId: b.id } });
      await expectStatus(204, "POST", "/blocks", { token: a.token, body: { userId: b.id } });
      const blocked = await expectStatus(200, "GET", "/blocks", { token: a.token });
      assert(
        blocked.items.some((item) => item.userId === b.id),
        "blocked list should include B"
      );
      const matches = await expectStatus(200, "GET", "/matches", { token: a.token });
      assert(!matches.items.some((item) => item.matchId === match.matchId), "blocked match must be hidden");
      const forbidden = await request("POST", `/conversations/${match.conversationId}/messages`, {
        token: b.token,
        body: { body: "hi" }
      });
      assert(
        forbidden.status === 403 || forbidden.status === 404,
        `blocked user must not message, got ${forbidden.status}`
      );
      const photo = await fetch(`${API_URL}${b.photo.url}`, { headers: { Authorization: `Bearer ${a.token}` } });
      assert(photo.status === 404, `blocked photo must be 404, got ${photo.status}`);
      await expectStatus(204, "DELETE", `/blocks/${b.id}`, { token: a.token });
    });

    await step("report and unmatch", async () => {
      await expectStatus(201, "POST", "/reports", {
        token: a.token,
        body: { userId: b.id, reason: "other", details: "smoke test" }
      });
      await expectStatus(204, "DELETE", `/matches/${match.matchId}`, { token: a.token });
      const matches = await expectStatus(200, "GET", "/matches", { token: b.token });
      assert(!matches.items.some((item) => item.matchId === match.matchId), "unmatched pair must disappear for both");
    });
  } finally {
    if (socket) socket.close();
    for (const user of users) {
      const result = await request("DELETE", "/me", { token: user.token, body: { password: PASSWORD } }).catch(
        (error) => ({ status: 0, data: error.message })
      );
      if (result.status !== 204) {
        console.error(`  cleanup failed for ${user.email}: ${result.status} ${JSON.stringify(result.data)}`);
        process.exitCode = 1;
      }
    }
  }
  if (users.length === 2) {
    const login = await request("POST", "/auth/login", { body: { email: users[0].email, password: PASSWORD } });
    assert(login.status === 401, `deleted account must not log in, got ${login.status}`);
    console.log("  ok    accounts deleted");
  }
  console.log("[e2e-smoke] PASS");
}

main().catch((error) => {
  console.error(`[e2e-smoke] FAIL: ${error.message}`);
  process.exit(1);
});
