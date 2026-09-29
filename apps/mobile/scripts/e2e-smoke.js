// End-to-end API smoke test for Alba. Requires a running API (default http://localhost:18080).
//   npm run e2e:smoke
const {
  API_BASE_URL,
  PASSWORD,
  assert,
  completeProfile,
  registerUser,
  request,
  uniqueEmail,
  uploadPhoto
} = require("./lib/fixtures");

let stepNumber = 0;
async function step(name, fn) {
  stepNumber += 1;
  process.stdout.write(`[e2e-smoke] ${String(stepNumber).padStart(2, "0")} ${name} ... `);
  await fn();
  process.stdout.write("ok\n");
}

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);
  const emailA = uniqueEmail("smoke_a");
  const emailB = uniqueEmail("smoke_b");
  let a;
  let b;
  let conversationId;
  let messageId;

  await step("health", async () => {
    const res = await request("/health");
    assert(res.status === 200 && res.payload?.status === "ok", `health: ${res.status}`);
  });

  await step("register validation (under 18, weak password)", async () => {
    const underage = await request("/auth/register", {
      method: "POST",
      body: { email: uniqueEmail("kid"), password: PASSWORD, birthDate: "2015-01-01" }
    });
    assert(underage.status === 422, `under 18 should be 422, got ${underage.status}`);
    const weak = await request("/auth/register", {
      method: "POST",
      body: { email: uniqueEmail("weak"), password: "short", birthDate: "1990-01-01" }
    });
    assert(weak.status === 400, `short password should be 400, got ${weak.status}`);
  });

  await step("register two users + duplicate email", async () => {
    a = await registerUser(emailA);
    b = await registerUser(emailB);
    assert(a.user.profileComplete === false, "new account must not be profile-complete");
    const dup = await request("/auth/register", {
      method: "POST",
      body: { email: emailA, password: PASSWORD, birthDate: "1990-01-01" }
    });
    assert(dup.status === 409, `duplicate email should be 409, got ${dup.status}`);
  });

  await step("login, /me, refresh rotation", async () => {
    const login = await request("/auth/login", {
      method: "POST",
      body: { email: emailA, password: PASSWORD }
    });
    assert(login.status === 200 && login.payload.user.id === a.user.id, "login user mismatch");
    const bad = await request("/auth/login", {
      method: "POST",
      body: { email: emailA, password: "wrong-pass-1" }
    });
    assert(bad.status === 401, `wrong password should be 401, got ${bad.status}`);
    const me = await request("/me", { token: login.payload.accessToken });
    assert(me.status === 200 && me.payload.age >= 18, "GET /me failed");
    const refreshed = await request("/auth/refresh", {
      method: "POST",
      body: { refreshToken: login.payload.refreshToken }
    });
    assert(
      refreshed.status === 200 && refreshed.payload.refreshToken !== login.payload.refreshToken,
      "refresh must rotate"
    );
    const reuse = await request("/auth/refresh", {
      method: "POST",
      body: { refreshToken: login.payload.refreshToken }
    });
    assert(
      reuse.status === 401,
      `reusing a rotated refresh token should be 401, got ${reuse.status}`
    );
    const noToken = await request("/me");
    assert(noToken.status === 401, "GET /me without token should be 401");
  });

  await step("interests + profile 404 before onboarding", async () => {
    const interests = await request("/interests", { token: a.accessToken });
    assert(
      interests.status === 200 && interests.payload.interests.length > 0,
      "interests should not be empty"
    );
    const before = await request("/me/profile", { token: a.accessToken });
    assert(before.status === 404, `profile before onboarding should be 404, got ${before.status}`);
  });

  await step("profiles, preferences, location, photo", async () => {
    await completeProfile(a, { firstName: "Alma", gender: "woman", interestedIn: ["man"] });
    await completeProfile(b, { firstName: "Bruno", gender: "man", interestedIn: ["woman"] });
    const me = await request("/me", { token: a.accessToken });
    assert(me.payload.profileComplete === true, "profile should be complete after onboarding");
    const profile = await request("/me/profile", { token: a.accessToken });
    assert(
      profile.payload.hasLocation && profile.payload.isComplete,
      "own profile should be complete"
    );
    assert(profile.payload.photos.length === 1, "one photo expected");
    const prefs = await request("/me/preferences", { token: a.accessToken });
    assert(prefs.payload.interestedIn.includes("man"), "preferences should be saved");
  });

  await step("photo reorder + delete + non-image rejected", async () => {
    const second = await uploadPhoto(a.accessToken, [40, 90, 160]);
    assert(second.status === 201, `second upload: ${second.status}`);
    const before = (await request("/me/profile", { token: a.accessToken })).payload.photos;
    const reversed = [...before].reverse().map((p) => p.id);
    const reorder = await request("/me/photos/order", {
      method: "PUT",
      token: a.accessToken,
      body: { photoIds: reversed }
    });
    assert(
      reorder.status === 200 && reorder.payload.photos[0].id === reversed[0],
      "reorder failed"
    );
    const bogus = new FormData();
    bogus.append("file", new Blob(["not an image"], { type: "image/png" }), "x.png");
    const rejected = await request("/me/photos", {
      method: "POST",
      token: a.accessToken,
      form: bogus
    });
    assert(
      rejected.status === 400 || rejected.status === 422,
      `non-image should be rejected, got ${rejected.status}`
    );
    const del = await request(`/me/photos/${reversed[1]}`, {
      method: "DELETE",
      token: a.accessToken
    });
    assert(del.status === 204, `delete photo: ${del.status}`);
    const after = (await request("/me/profile", { token: a.accessToken })).payload.photos;
    assert(after.length === 1 && after[0].position === 0, "photos should be re-packed");
  });

  await step("discover + signed photo url", async () => {
    const discover = await request("/discover?limit=20", { token: a.accessToken });
    assert(discover.status === 200, `discover: ${discover.status}`);
    const seesB = discover.payload.profiles.find((p) => p.userId === b.user.id);
    assert(seesB, "A should discover B");
    assert(seesB.photos.length > 0, "discovered profile should have photos");
    const photo = await fetch(`${API_BASE_URL}${seesB.photos[0].url}`);
    assert(
      photo.status === 200 && (photo.headers.get("content-type") || "").includes("image/jpeg"),
      "signed photo url must serve JPEG"
    );
  });

  await step("like -> mutual like -> match", async () => {
    const first = await request("/swipes", {
      method: "POST",
      token: a.accessToken,
      body: { userId: b.user.id, action: "like" }
    });
    assert(first.status === 200 && first.payload.matched === false, "first like must not match");
    const dup = await request("/swipes", {
      method: "POST",
      token: a.accessToken,
      body: { userId: b.user.id, action: "like" }
    });
    assert(dup.status === 409, `duplicate swipe should be 409, got ${dup.status}`);
    const second = await request("/swipes", {
      method: "POST",
      token: b.accessToken,
      body: { userId: a.user.id, action: "like" }
    });
    assert(second.status === 200 && second.payload.matched === true, "mutual like must match");
    conversationId = second.payload.conversation.id;
    assert(conversationId, "match must include a conversation");
    const self = await request("/swipes", {
      method: "POST",
      token: a.accessToken,
      body: { userId: a.user.id, action: "like" }
    });
    assert(self.status === 400, `self swipe should be 400, got ${self.status}`);
  });

  await step("conversation list, send, read receipts", async () => {
    const list = await request("/conversations?limit=30", { token: a.accessToken });
    const conv = list.payload.conversations.find((c) => c.id === conversationId);
    assert(
      conv && conv.lastMessage === null,
      "new match should appear as a conversation without messages"
    );
    const sent = await request(`/conversations/${conversationId}/messages`, {
      method: "POST",
      token: a.accessToken,
      body: { body: "  Hello from the smoke test  " }
    });
    assert(
      sent.status === 201 && sent.payload.body === "Hello from the smoke test",
      "message should be trimmed and stored"
    );
    messageId = sent.payload.id;
    const empty = await request(`/conversations/${conversationId}/messages`, {
      method: "POST",
      token: a.accessToken,
      body: { body: "   " }
    });
    assert(empty.status === 400, `blank message should be 400, got ${empty.status}`);
    const listB = await request("/conversations", { token: b.accessToken });
    assert(
      listB.payload.conversations.find((c) => c.id === conversationId)?.unreadCount === 1,
      "B should have 1 unread"
    );
    const read = await request(`/conversations/${conversationId}/read`, {
      method: "POST",
      token: b.accessToken
    });
    assert(read.status === 204, `mark read: ${read.status}`);
    const thread = await request(`/conversations/${conversationId}/messages?limit=30`, {
      token: a.accessToken
    });
    assert(
      thread.payload.messages.find((m) => m.id === messageId)?.readAt,
      "A should see the read receipt"
    );
    const outsider = await registerUser(uniqueEmail("smoke_c"));
    const denied = await request(`/conversations/${conversationId}/messages`, {
      token: outsider.accessToken
    });
    assert(denied.status === 404, `non participant should get 404, got ${denied.status}`);
  });

  await step("notifications", async () => {
    const list = await request("/notifications?limit=30", { token: b.accessToken });
    assert(
      list.status === 200 && list.payload.unreadCount >= 1,
      "B should have unread notifications"
    );
    const types = list.payload.notifications.map((n) => n.type);
    assert(types.includes("match"), "B should have a match notification");
    const one = list.payload.notifications[0];
    const readOne = await request(`/notifications/${one.id}/read`, {
      method: "POST",
      token: b.accessToken
    });
    assert(readOne.status === 204, `read one: ${readOne.status}`);
    const all = await request("/notifications/read-all", { method: "POST", token: b.accessToken });
    assert(all.status === 204, `read all: ${all.status}`);
    const after = await request("/notifications", { token: b.accessToken });
    assert(after.payload.unreadCount === 0, "unread count should be 0 after read-all");
  });

  await step("block, list, report, unblock", async () => {
    const block = await request("/blocks", {
      method: "POST",
      token: a.accessToken,
      body: { userId: b.user.id }
    });
    assert(block.status === 204, `block: ${block.status}`);
    const again = await request("/blocks", {
      method: "POST",
      token: a.accessToken,
      body: { userId: b.user.id }
    });
    assert(again.status === 204, "blocking twice must be idempotent");
    const blocks = await request("/blocks", { token: a.accessToken });
    assert(
      blocks.payload.blocks.some((x) => x.userId === b.user.id),
      "block list should include B"
    );
    const list = await request("/conversations", { token: a.accessToken });
    assert(
      !list.payload.conversations.some((c) => c.id === conversationId),
      "blocking removes the conversation"
    );
    const profile = await request(`/profiles/${b.user.id}`, { token: a.accessToken });
    assert(profile.status === 404, `blocked profile should be 404, got ${profile.status}`);
    const report = await request("/reports", {
      method: "POST",
      token: a.accessToken,
      body: { userId: b.user.id, reason: "spam", details: "smoke test", block: true }
    });
    assert(report.status === 201 && report.payload.id, `report: ${report.status}`);
    const unblock = await request(`/blocks/${b.user.id}`, {
      method: "DELETE",
      token: a.accessToken
    });
    assert(unblock.status === 204, `unblock: ${unblock.status}`);
  });

  await step("change password, forgot (always 202)", async () => {
    const newPassword = `${PASSWORD}-new`;
    const wrong = await request("/me/password", {
      method: "POST",
      token: a.accessToken,
      body: { currentPassword: "definitely-wrong", newPassword }
    });
    assert(
      wrong.status >= 400 && wrong.status < 500,
      `wrong current password should be a 4xx, got ${wrong.status}`
    );
    const changed = await request("/me/password", {
      method: "POST",
      token: a.accessToken,
      body: { currentPassword: PASSWORD, newPassword }
    });
    assert(changed.status === 204, `change password: ${changed.status}`);
    const login = await request("/auth/login", {
      method: "POST",
      body: { email: emailA, password: newPassword }
    });
    assert(login.status === 200, "login with the new password should work");
    const forgot = await request("/auth/forgot", {
      method: "POST",
      body: { email: "nobody@alba.test" }
    });
    assert(forgot.status === 202, `forgot should always be 202, got ${forgot.status}`);
    const logout = await request("/auth/logout", {
      method: "POST",
      body: { refreshToken: login.payload.refreshToken }
    });
    assert(logout.status === 204, `logout: ${logout.status}`);
  });

  await step("delete account", async () => {
    const wrong = await request("/me", {
      method: "DELETE",
      token: b.accessToken,
      body: { password: "nope-nope-1" }
    });
    assert(
      wrong.status >= 400 && wrong.status < 500,
      `wrong password should be a 4xx, got ${wrong.status}`
    );
    const del = await request("/me", {
      method: "DELETE",
      token: b.accessToken,
      body: { password: PASSWORD }
    });
    assert(del.status === 204, `delete account: ${del.status}`);
    const login = await request("/auth/login", {
      method: "POST",
      body: { email: emailB, password: PASSWORD }
    });
    assert(login.status === 401, `deleted account cannot log in, got ${login.status}`);
  });

  console.log("[e2e-smoke] PASS");
}

main().catch((err) => {
  process.stdout.write("FAILED\n");
  console.error(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
