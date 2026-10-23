/**
 * Live-API smoke test of the critical flow, with two throwaway users:
 * register -> profile/preferences/photo -> discover -> mutual like -> match -> WebSocket message.new
 * -> read receipt -> notifications -> block -> account deletion.
 *
 * Usage: API_URL=http://localhost:18080 npm run e2e:smoke   (EXPO_PUBLIC_API_URL also works)
 */
const {
  API_URL,
  PASSWORD,
  assert,
  call,
  deleteUser,
  expectStatus,
  onboardUser,
  preferenceInput,
  profileInput,
  registerUser
} = require("./lib/api");

function wsUrl() {
  return `${API_URL.replace(/^http/i, "ws")}/ws`;
}

/** Opens the realtime socket, authenticates in onopen and resolves once the server says ready. */
function openSocket(token) {
  return new Promise((resolve, reject) => {
    const events = [];
    const waiters = [];
    const socket = new WebSocket(wsUrl());
    const timer = setTimeout(() => reject(new Error("websocket: no ready frame within 8s")), 8000);
    socket.onopen = () => socket.send(JSON.stringify({ type: "auth", token }));
    socket.onerror = () => reject(new Error("websocket: connection error"));
    socket.onmessage = (event) => {
      const frame = JSON.parse(String(event.data));
      if (frame.type === "ready") {
        clearTimeout(timer);
        resolve({
          events,
          close: () => socket.close(),
          ping: () => socket.send(JSON.stringify({ type: "ping" })),
          waitFor: (type, ms = 8000) =>
            new Promise((ok, fail) => {
              const found = events.find((item) => item.type === type);
              if (found) return ok(found);
              const timeout = setTimeout(
                () => fail(new Error(`websocket: no ${type} within ${ms}ms`)),
                ms
              );
              waiters.push({
                type,
                ok: (item) => {
                  clearTimeout(timeout);
                  ok(item);
                }
              });
            })
        });
        return;
      }
      events.push(frame);
      waiters.filter((w) => w.type === frame.type).forEach((w) => w.ok(frame));
    };
  });
}

function step(label) {
  console.log(`[e2e-smoke] ${label}`);
}

async function main() {
  console.log(`[e2e-smoke] API: ${API_URL}`);
  const users = [];
  let socket = null;
  try {
    step("health");
    expectStatus(await call("/health"), 200, "health");

    step("register x2");
    const a = await registerUser("smoke_a");
    const b = await registerUser("smoke_b");
    users.push(a, b);
    expectStatus(
      await call("/auth/login", { method: "POST", body: { email: a.email, password: PASSWORD } }),
      200,
      "login"
    );
    const me = expectStatus(await call("/me", { token: a.token }), 200, "me");
    assert(me.onboarding.completed === false, "fresh account should not be onboarded");

    step("interests + profiles + preferences + photo");
    const interests = expectStatus(
      await call("/interests", { token: a.token }),
      200,
      "interests"
    ).interests;
    assert(Array.isArray(interests) && interests.length > 0, "interests list should not be empty");
    const slugs = interests.slice(0, 2).map((item) => item.slug);
    await onboardUser(
      a,
      profileInput("Alix", "woman", "1995-04-12", slugs),
      preferenceInput(["man"]),
      1
    );
    await onboardUser(
      b,
      profileInput("Boris", "man", "1993-09-30", slugs),
      preferenceInput(["woman"]),
      2
    );
    const meAfter = expectStatus(await call("/me", { token: a.token }), 200, "me after onboarding");
    assert(
      meAfter.onboarding.completed === true,
      "onboarding should be completed after profile + photo"
    );

    const photos = expectStatus(
      await call("/me/photos", { token: a.token }),
      200,
      "list photos"
    ).photos;
    assert(photos.length === 1 && photos[0].position === 0, "expected one main photo");

    step("discover");
    const discoverA = expectStatus(
      await call("/discover?limit=20", { token: a.token }),
      200,
      "discover A"
    );
    const cardB = discoverA.profiles.find((card) => card.userId === b.id);
    assert(cardB, "A should discover B");
    assert(cardB.age >= 18 && cardB.photos.length === 1, "card shape (age, photos)");

    step("authenticated photo fetch");
    const imageResponse = await fetch(`${API_URL}${cardB.photos[0].thumbUrl}`, {
      headers: { Authorization: `Bearer ${a.token}` }
    });
    assert(imageResponse.status === 200, `photo thumb status ${imageResponse.status}`);
    assert(
      (imageResponse.headers.get("content-type") || "").startsWith("image/"),
      "photo should be an image"
    );
    const anonymous = await fetch(`${API_URL}${cardB.photos[0].thumbUrl}`);
    assert(anonymous.status === 401, `photo without token should be 401, got ${anonymous.status}`);

    step("mutual like -> match");
    const first = expectStatus(
      await call("/swipes", {
        method: "POST",
        token: a.token,
        body: { targetUserId: b.id, action: "like" }
      }),
      200,
      "A likes B"
    );
    assert(first.matched === false, "first like must not match");
    const repeat = expectStatus(
      await call("/swipes", {
        method: "POST",
        token: a.token,
        body: { targetUserId: b.id, action: "like" }
      }),
      200,
      "repeat like"
    );
    assert(repeat.alreadyProcessed === true, "repeated swipe should be idempotent");
    const discoverB = expectStatus(
      await call("/discover?limit=20", { token: b.token }),
      200,
      "discover B"
    );
    assert(
      discoverB.profiles.some((card) => card.userId === a.id),
      "B should discover A"
    );

    socket = await openSocket(a.token);
    const second = expectStatus(
      await call("/swipes", {
        method: "POST",
        token: b.token,
        body: { targetUserId: a.id, action: "like" }
      }),
      200,
      "B likes A"
    );
    assert(second.matched === true && second.match?.id, "second like must create a match");
    const matchId = second.match.id;
    await socket.waitFor("match.new");

    step("message over REST, delivered over WebSocket");
    const sent = expectStatus(
      await call(`/matches/${matchId}/messages`, {
        method: "POST",
        token: b.token,
        body: { content: "Bonjour depuis le smoke test" }
      }),
      201,
      "send message"
    );
    assert(sent.senderId === b.id && sent.readAt === null, "message shape");
    const live = await socket.waitFor("message.new");
    assert(
      live.data.id === sent.id && live.data.content === sent.content,
      "message.new payload mismatch"
    );
    socket.ping();

    step("matches, unread, read receipt");
    const matches = expectStatus(
      await call("/matches?limit=20", { token: a.token }),
      200,
      "matches"
    ).matches;
    const summary = matches.find((item) => item.id === matchId);
    assert(summary && summary.unreadCount === 1, "A should have 1 unread message");
    assert(summary.lastMessage?.content === sent.content, "lastMessage should be the sent message");
    const history = expectStatus(
      await call(`/matches/${matchId}/messages?limit=30`, { token: a.token }),
      200,
      "messages"
    );
    assert(history.messages[0].id === sent.id, "history is newest first");
    expectStatus(
      await call(`/matches/${matchId}/read`, { method: "POST", token: a.token }),
      204,
      "mark read"
    );
    const afterRead = expectStatus(
      await call("/matches?limit=20", { token: a.token }),
      200,
      "matches 2"
    );
    assert(
      afterRead.matches.find((m) => m.id === matchId).unreadCount === 0,
      "unread should reset"
    );
    const bHistory = expectStatus(
      await call(`/matches/${matchId}/messages?limit=30`, { token: b.token }),
      200,
      "messages B"
    );
    assert(bHistory.messages[0].readAt, "B's message should carry readAt");

    step("notifications");
    const notifications = expectStatus(
      await call("/notifications?limit=20", { token: a.token }),
      200,
      "notifications"
    );
    assert(notifications.notifications.length > 0, "A should have notifications (match / message)");
    assert(
      notifications.notifications.every((n) => n.type === "match" || n.type === "message"),
      "notification types"
    );
    if (notifications.notifications.some((n) => !n.readAt)) {
      const unread = notifications.notifications.find((n) => !n.readAt);
      expectStatus(
        await call(`/notifications/${unread.id}/read`, { method: "POST", token: a.token }),
        204,
        "read one"
      );
    }
    expectStatus(
      await call("/notifications/read-all", { method: "POST", token: a.token }),
      204,
      "read all"
    );
    const cleared = expectStatus(
      await call("/notifications?limit=20", { token: a.token }),
      200,
      "notifications 2"
    );
    assert(cleared.unreadCount === 0, "unreadCount should be 0 after read-all");

    step("block");
    expectStatus(
      await call("/blocks", { method: "POST", token: a.token, body: { userId: b.id } }),
      204,
      "block"
    );
    const blocks = expectStatus(await call("/blocks", { token: a.token }), 200, "blocks").blocks;
    assert(
      blocks.some((item) => item.userId === b.id),
      "blocks should list B"
    );
    const afterBlock = expectStatus(
      await call("/matches?limit=20", { token: a.token }),
      200,
      "matches 3"
    );
    assert(!afterBlock.matches.some((m) => m.id === matchId), "block should remove the match");
    expectStatus(
      await call(`/blocks/${b.id}`, { method: "DELETE", token: a.token }),
      204,
      "unblock"
    );

    step("token refresh");
    const refreshed = expectStatus(
      await call("/auth/refresh", { method: "POST", body: { refreshToken: a.refreshToken } }),
      200,
      "refresh"
    );
    assert(
      refreshed.accessToken && refreshed.refreshToken !== a.refreshToken,
      "refresh token should rotate"
    );
    a.token = refreshed.accessToken;

    step("account deletion");
    for (const user of users) {
      expectStatus(await deleteUser(user), 204, `delete ${user.email}`);
    }
    users.length = 0;
    const gone = await call("/auth/login", {
      method: "POST",
      body: { email: a.email, password: PASSWORD }
    });
    assert(gone.status === 401, `login after deletion should be 401, got ${gone.status}`);

    console.log(
      "[e2e-smoke] PASS health/auth/profile/photo/discover/match/chat/ws/notifications/block/delete"
    );
  } finally {
    if (socket) socket.close();
    for (const user of users) {
      try {
        await deleteUser(user);
      } catch {
        // best-effort cleanup of throwaway users
      }
    }
  }
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
