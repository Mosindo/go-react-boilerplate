/**
 * Fast API smoke test against a running stack (no browser):
 * health -> two members -> discovery -> mutual like -> match -> chat ->
 * read receipt -> notifications -> block -> cleanup.
 *
 *   E2E_API_URL=http://localhost:18080 node e2e/api-smoke.mjs
 */
import { apiClient } from "./helpers.mjs";

const API = process.env.E2E_API_URL ?? process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:18080";
const PASSWORD = "Password123!";
const RUN = Date.now();
const LAT = Number((-40 + Math.random() * 20).toFixed(3));
const LON = Number((60 + Math.random() * 20).toFixed(3));
const api = apiClient(API);

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function main() {
  console.log(`[api-smoke] ${API}`);
  await api.expectStatus(api.call("/health"), 200, "health");

  const a = await api.createMember({
    email: `smoke-a-${RUN}@dating.test`,
    password: PASSWORD,
    firstName: "Anaïs",
    gender: "woman",
    interestedIn: ["man"],
    birthdate: "1994-03-03",
    latitude: LAT,
    longitude: LON
  });
  const b = await api.createMember({
    email: `smoke-b-${RUN}@dating.test`,
    password: PASSWORD,
    firstName: "Basile",
    gender: "man",
    interestedIn: ["woman"],
    birthdate: "1992-02-02",
    latitude: LAT + 0.01,
    longitude: LON
  });

  try {
    const me = await api.expectStatus(api.call("/me/profile", { token: a.token }), 200, "own profile");
    assert(me.isComplete, "member A should be complete");

    const feed = await api.expectStatus(api.call("/discovery", { token: a.token }), 200, "discovery");
    assert(
      feed.profiles.some((p) => p.userId === b.id),
      "A should discover B"
    );

    const first = await api.expectStatus(
      api.call("/swipes", { method: "POST", token: a.token, body: { targetUserId: b.id, action: "like" } }),
      200,
      "A likes B"
    );
    assert(!first.matched, "single like must not match");
    const second = await api.expectStatus(
      api.call("/swipes", { method: "POST", token: b.token, body: { targetUserId: a.id, action: "like" } }),
      200,
      "B likes A"
    );
    assert(second.matched, "mutual like must match");

    await api.expectStatus(
      api.call(`/chats/${b.id}/messages`, { method: "POST", token: a.token, body: { content: "Bonjour !" } }),
      201,
      "send"
    );
    const chats = await api.expectStatus(api.call("/chats", { token: b.token }), 200, "chats");
    assert(chats.chats[0]?.unreadCount === 1, "B should have one unread message");
    await api.expectStatus(api.call(`/chats/${a.id}/read`, { method: "POST", token: b.token }), 200, "read");
    const page = await api.expectStatus(api.call(`/chats/${b.id}/messages`, { token: a.token }), 200, "messages");
    assert(page.peerLastReadAt, "A should see the read receipt");

    const notifications = await api.expectStatus(api.call("/notifications", { token: a.token }), 200, "notifications");
    assert(
      notifications.notifications.some((n) => n.type === "match"),
      "A should have a match notification"
    );

    await api.expectStatus(
      api.call("/blocks", { method: "POST", token: a.token, body: { userId: b.id } }),
      204,
      "block"
    );
    await api.expectStatus(
      api.call(`/chats/${a.id}/messages`, { method: "POST", token: b.token, body: { content: "?" } }),
      404,
      "blocked send"
    );
    console.log("[api-smoke] ✅ all checks passed");
  } finally {
    for (const m of [a, b]) {
      await api.call("/me", { method: "DELETE", token: m.token, body: { password: PASSWORD } });
    }
  }
}

main().catch((error) => {
  console.error(`[api-smoke] ❌ ${error.message}`);
  process.exit(1);
});
