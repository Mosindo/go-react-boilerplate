 
const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = "Password123";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@boilerplate.test`;
}

async function request(path, init = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers || {})
    }
  });

  let payload = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }

  return { status: response.status, ok: response.ok, payload };
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function register(email) {
  const res = await request("/auth/register", {
    method: "POST",
    body: JSON.stringify({ email, password: PASSWORD })
  });
  assert(res.status === 201, `register failed for ${email}: ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

async function main() {
  console.log(`[e2e-smoke] API: ${API_BASE_URL}`);

  const email1 = uniqueEmail("mobile_e2e_a");
  const email2 = uniqueEmail("mobile_e2e_b");

  const auth1 = await register(email1);
  const auth2 = await register(email2);
  assert(auth1?.token && auth2?.token, "expected tokens after register");

  const login1 = await request("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email: email1, password: PASSWORD })
  });
  assert(login1.status === 200, `login failed: ${login1.status} ${JSON.stringify(login1.payload)}`);
  assert(login1.payload?.user?.id === auth1.user.id, "login user mismatch");

  const users = await request("/users", {
    method: "GET",
    headers: { Authorization: `Bearer ${auth1.token}` }
  });
  assert(users.status === 200, `users failed: ${users.status}`);
  const seesUser2 = Array.isArray(users.payload?.users) && users.payload.users.some((u) => u.id === auth2.user.id);
  assert(seesUser2, "users list should include second user");

  const createPost = await request("/posts", {
    method: "POST",
    headers: { Authorization: `Bearer ${auth1.token}` },
    body: JSON.stringify({ title: "Smoke post", body: "Validating the generic post feed." })
  });
  assert(createPost.status === 201, `create post failed: ${createPost.status}`);

  const listPosts = await request("/posts", {
    method: "GET",
    headers: { Authorization: `Bearer ${auth1.token}` }
  });
  assert(listPosts.status === 200, `list posts failed: ${listPosts.status}`);
  const seesNewPost =
    Array.isArray(listPosts.payload?.posts) &&
    listPosts.payload.posts.some((post) => post.id === createPost.payload?.id);
  assert(seesNewPost, "posts list should include created post");

  const send = await request(`/chats/${auth2.user.id}/messages`, {
    method: "POST",
    headers: { Authorization: `Bearer ${auth1.token}` },
    body: JSON.stringify({ content: "hello from mobile e2e smoke" })
  });
  assert(send.status === 201, `send message failed: ${send.status}`);

  const listMessages = await request(`/chats/${auth2.user.id}/messages`, {
    method: "GET",
    headers: { Authorization: `Bearer ${auth1.token}` }
  });
  assert(listMessages.status === 200, `list messages failed: ${listMessages.status}`);
  assert(
    Array.isArray(listMessages.payload?.messages) && listMessages.payload.messages.length > 0,
    "messages should not be empty"
  );

  const createNotification = await request("/notifications", {
    method: "POST",
    headers: { Authorization: `Bearer ${auth1.token}` },
    body: JSON.stringify({
      type: "system",
      title: "Smoke notification",
      body: "Validating the generic notification flow."
    })
  });
  assert(createNotification.status === 201, `create notification failed: ${createNotification.status}`);

  const listNotifications = await request("/notifications", {
    method: "GET",
    headers: { Authorization: `Bearer ${auth1.token}` }
  });
  assert(listNotifications.status === 200, `list notifications failed: ${listNotifications.status}`);
  const seesNotification =
    Array.isArray(listNotifications.payload?.notifications) &&
    listNotifications.payload.notifications.some((item) => item.id === createNotification.payload?.id);
  assert(seesNotification, "notifications list should include created notification");

  const markRead = await request(`/notifications/${createNotification.payload.id}/read`, {
    method: "POST",
    headers: { Authorization: `Bearer ${auth1.token}` }
  });
  assert(markRead.status === 200, `mark notification read failed: ${markRead.status}`);
  assert(markRead.payload?.isRead === true, "notification should be marked as read");

  console.log("[e2e-smoke] PASS register/login/users/posts/chat/notifications");
}

main().catch((err) => {
  console.error(`[e2e-smoke] FAIL ${err instanceof Error ? err.message : String(err)}`);
  process.exit(1);
});
