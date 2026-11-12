// End-to-end API smoke test against a running stack:
// register → onboarding → discover → like → match → chat → read → notifications → block → delete.
// Usage: E2E_API_URL=http://localhost:18080 node e2e/api-smoke.mjs
import { API_URL, assert, createMember, expectStatus, request } from "./lib/api.mjs";

async function main() {
  console.log(`[api-smoke] ${API_URL}`);
  await expectStatus(request("/health"), 200, "health");

  const ana = await createMember("smoke_ana", { firstName: "Ana", gender: "woman", interestedIn: ["man"] });
  const leo = await createMember("smoke_leo", { firstName: "Leo", gender: "man", interestedIn: ["woman"] });

  const deck = await expectStatus(request("/discover?limit=30", { token: leo.token }), 200, "discover");
  assert(
    deck.profiles.some((p) => p.userId === ana.id),
    "leo should discover ana"
  );
  assert(!JSON.stringify(deck).includes("@"), "discovery must not leak emails");

  const first = await expectStatus(
    request("/swipes", { method: "POST", token: leo.token, body: { targetUserId: ana.id, decision: "like" } }),
    200,
    "leo likes ana"
  );
  assert(!first.matched, "one-sided like must not match");
  await expectStatus(request(`/chats/${ana.id}/messages`, { method: "POST", token: leo.token, body: { content: "too early" } }), 404, "chat before match");
  const second = await expectStatus(
    request("/swipes", { method: "POST", token: ana.token, body: { targetUserId: leo.id, decision: "like" } }),
    200,
    "ana likes leo"
  );
  assert(second.matched, "mutual like must match");

  await expectStatus(request(`/chats/${ana.id}/messages`, { method: "POST", token: leo.token, body: { content: "Salut Ana !" } }), 201, "send");
  const chats = await expectStatus(request("/chats", { token: ana.token }), 200, "chats");
  assert(chats.chats[0]?.unreadCount === 1, "ana has one unread message");
  await expectStatus(request(`/chats/${leo.id}/read`, { method: "POST", token: ana.token }), 200, "read");
  const page = await expectStatus(request(`/chats/${ana.id}/messages`, { token: leo.token }), 200, "messages");
  assert(page.otherLastReadAt, "leo sees the read receipt");

  const notifications = await expectStatus(request("/notifications", { token: ana.token }), 200, "notifications");
  assert(
    notifications.notifications.some((n) => n.type === "match"),
    "match notification"
  );
  await expectStatus(request("/notifications/read-all", { method: "POST", token: ana.token }), 204, "read all");

  await expectStatus(request("/blocks", { method: "POST", token: ana.token, body: { userId: leo.id } }), 204, "block");
  await expectStatus(request(`/chats/${ana.id}/messages`, { method: "POST", token: leo.token, body: { content: "?" } }), 404, "chat after block");

  for (const m of [ana, leo]) {
    await expectStatus(request("/me", { method: "DELETE", token: m.token, body: { password: "Password123" } }), 204, "delete account");
  }
  await expectStatus(request("/me", { token: ana.token }), 401, "token revoked after deletion");
  console.log("[api-smoke] OK");
}

main().catch((error) => {
  console.error(`[api-smoke] FAILED: ${error.message}`);
  process.exit(1);
});
