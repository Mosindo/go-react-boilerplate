/* eslint-disable no-console */
// API-level smoke test of the dating journey described in docs/API.md.
// Usage: MOBILE_E2E_API_URL=http://<LAN_IP>:18080 npm run e2e:smoke
const API_BASE_URL = (
  process.env.MOBILE_E2E_API_URL ||
  process.env.EXPO_PUBLIC_API_URL ||
  "http://localhost:18080"
).replace(/\/+$/, "");
const PASSWORD = "Password123!";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@smoke.test`;
}

async function request(path, { method = "GET", token, body } = {}) {
  const headers = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body)
  });
  const text = await response.text();
  let payload = null;
  try {
    payload = text ? JSON.parse(text) : null;
  } catch {
    payload = null;
  }
  return { status: response.status, payload };
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function expectOk(label, promise, expectedStatuses = [200, 201, 204]) {
  const result = await promise;
  assert(
    expectedStatuses.includes(result.status),
    `${label} failed: ${result.status} ${JSON.stringify(result.payload)}`
  );
  console.log(`ok  ${label}`);
  return result.payload;
}

async function createUser(prefix, gender, interestedIn, firstName) {
  const session = await expectOk(
    `register ${prefix}`,
    request("/auth/register", { method: "POST", body: { email: uniqueEmail(prefix), password: PASSWORD } })
  );
  const token = session.accessToken;
  await expectOk(
    `preferences ${prefix}`,
    request("/profile/preferences", {
      method: "PUT",
      token,
      body: {
        interestedIn,
        minAge: 18,
        maxAge: 60,
        maxDistanceKm: 200,
        discoverable: true,
        showDistance: true
      }
    })
  );
  await expectOk(
    `location ${prefix}`,
    request("/profile/location", { method: "PUT", token, body: { latitude: 45.764, longitude: 4.835 } })
  );
  await expectOk(
    `profile ${prefix}`,
    request("/profile/me", {
      method: "PUT",
      token,
      body: {
        firstName,
        birthDate: "1994-04-12",
        gender,
        bio: "Smoke test profile",
        city: "Lyon",
        interests: []
      }
    })
  );
  return { token, id: session.user.id };
}

async function main() {
  console.log(`[smoke] API ${API_BASE_URL}`);
  const health = await request("/health");
  assert(health.status === 200, `health failed: ${health.status}`);

  const a = await createUser("alice", "woman", ["man"], "Alice");
  const b = await createUser("bob", "man", ["woman"], "Bob");

  // Photos are multipart uploads and are covered by backend integration tests; users without photos
  // are not "complete", so this smoke focuses on the account, profile and safety endpoints.
  const me = await expectOk("GET /me", request("/me", { token: a.token }));
  assert(me.id === a.id, "me id mismatch");

  const prof = await expectOk("GET /profile/me", request("/profile/me", { token: a.token }));
  assert(prof.profile && prof.profile.firstName === "Alice", "profile not saved");

  await expectOk("GET /interests", request("/interests", { token: a.token }));
  await expectOk("GET /matches", request("/matches", { token: a.token }));
  await expectOk("GET /conversations", request("/conversations", { token: a.token }));
  await expectOk("GET /notifications", request("/notifications", { token: a.token }));
  await expectOk("block", request("/blocks", { method: "POST", token: a.token, body: { userId: b.id } }));
  await expectOk(
    "report",
    request("/reports", {
      method: "POST",
      token: a.token,
      body: { userId: b.id, reason: "spam", details: "smoke" }
    })
  );
  await expectOk("unblock", request(`/blocks/${b.id}`, { method: "DELETE", token: a.token }));
  await expectOk(
    "delete account",
    request("/me", { method: "DELETE", token: b.token, body: { password: PASSWORD } })
  );
  const gone = await request("/me", { token: b.token });
  assert(gone.status === 401 || gone.status === 404, `deleted account still resolves: ${gone.status}`);
  console.log("[smoke] passed");
}

main().catch((error) => {
  console.error(`[smoke] FAILED: ${error.message}`);
  process.exit(1);
});
