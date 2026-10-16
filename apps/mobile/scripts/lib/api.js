// Tiny API helper shared by the Node-based end-to-end scripts (no dependencies).
const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123";

// 240x300 JPEG, generated once: the server requires at least 200x200.
const SAMPLE_JPEG_BASE64 =
  "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";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.lien.test`;
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function request(path, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payloadBody;
  if (form) {
    payloadBody = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payloadBody = JSON.stringify(body);
  }
  const response = await fetch(`${API_BASE_URL}${path}`, { method, headers, body: payloadBody });
  let payload = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  return { status: response.status, ok: response.ok, payload };
}

async function register(email) {
  const res = await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  assert(res.status === 201, `register failed for ${email}: ${res.status} ${JSON.stringify(res.payload)}`);
  return { token: res.payload.accessToken, id: res.payload.user.id, email };
}

async function uploadSamplePhoto(token) {
  const form = new FormData();
  form.append("photo", new Blob([Buffer.from(SAMPLE_JPEG_BASE64, "base64")], { type: "image/jpeg" }), "sample.jpg");
  const res = await request("/me/photos", { method: "POST", token, form });
  assert(res.status === 201, `photo upload failed: ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

async function onboard(user, { name, gender, seeking, birthDate = "1994-04-12", lat = 48.8566, lon = 2.3522, interests = ["music", "travel"] }) {
  let res = await request("/me/profile", {
    method: "PUT",
    token: user.token,
    body: { firstName: name, birthDate, gender, bio: "Profil de test e2e", city: "Paris", latitude: lat, longitude: lon, interests }
  });
  assert(res.status === 200, `profile failed for ${name}: ${res.status} ${JSON.stringify(res.payload)}`);
  res = await request("/me/preferences", {
    method: "PUT",
    token: user.token,
    body: { interestedIn: seeking, minAge: 18, maxAge: 60, maxDistanceKm: 50 }
  });
  assert(res.status === 200, `preferences failed for ${name}: ${res.status}`);
  await uploadSamplePhoto(user.token);
}

module.exports = { API_BASE_URL, PASSWORD, SAMPLE_JPEG_BASE64, assert, onboard, register, request, uniqueEmail, uploadSamplePhoto };
