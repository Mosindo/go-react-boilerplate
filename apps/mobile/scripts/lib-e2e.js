/* eslint-disable no-console */
// Shared helpers for the API-level end-to-end scripts (plain Node 20+, no dependencies).

const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123!";

// A 256x320 JPEG, so scripts need no image tooling.
const SAMPLE_JPEG_BASE64 =
  "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";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.invalid`;
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

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function expectStatus(res, status, what) {
  assert(res.status === status, `${what}: expected ${status}, got ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

function birthdate(age) {
  const d = new Date();
  d.setFullYear(d.getFullYear() - age);
  d.setDate(d.getDate() - 3);
  return d.toISOString().slice(0, 10);
}

async function registerAndOnboard({ prefix, name, gender, interestedIn, age = 29 }) {
  const email = uniqueEmail(prefix);
  const auth = expectStatus(
    await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }),
    201,
    `register ${email}`
  );
  const token = auth.accessToken;
  expectStatus(
    await request("/profile", {
      method: "PUT",
      token,
      body: { firstName: name, birthdate: birthdate(age), gender, bio: "Profil de test e2e" }
    }),
    200,
    "save profile"
  );
  expectStatus(
    await request("/profile/location", { method: "PUT", token, body: { latitude: 48.8566, longitude: 2.3522, city: "Paris" } }),
    200,
    "save location"
  );
  expectStatus(
    await request("/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 60, maxDistanceKm: 50 } }),
    200,
    "save preferences"
  );
  const form = new FormData();
  form.append("file", new Blob([Buffer.from(SAMPLE_JPEG_BASE64, "base64")], { type: "image/jpeg" }), "photo.jpg");
  expectStatus(await request("/photos", { method: "POST", token, form }), 201, "upload photo");
  return { email, token, refreshToken: auth.refreshToken, id: auth.user.id };
}

module.exports = { API_BASE_URL, PASSWORD, request, assert, expectStatus, registerAndOnboard, uniqueEmail };
