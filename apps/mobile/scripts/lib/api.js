/* eslint-disable no-console */
// Shared helpers for the Node-driven end-to-end scripts (no dependencies).
const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123";

// 256x256 gradient JPEG (valid for the API: >= 200 px, JPEG).
const PHOTO_BASE64 =
  "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";

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.invalid`;
}

async function request(path, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    body: form ?? (body === undefined ? undefined : JSON.stringify(body))
  });
  let payload = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  return { status: response.status, ok: response.ok, payload };
}

async function expectStatus(label, promise, status) {
  const res = await promise;
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

function birthDateForAge(age) {
  const d = new Date();
  d.setFullYear(d.getFullYear() - age);
  d.setDate(d.getDate() - 2);
  return d.toISOString().slice(0, 10);
}

function photoForm() {
  const form = new FormData();
  form.append("photo", new Blob([Buffer.from(PHOTO_BASE64, "base64")], { type: "image/jpeg" }), "photo.jpg");
  return form;
}

/** Registers and fully onboards a member; returns { email, token, id }. */
async function createMember({ name, gender, interestedIn, lat, lng, interests = [] }) {
  const email = uniqueEmail(name.toLowerCase());
  const auth = await expectStatus(
    `register ${name}`,
    request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }),
    201
  );
  const token = auth.accessToken;
  await expectStatus(
    `profile ${name}`,
    request("/me/profile", {
      method: "PUT",
      token,
      body: { firstName: name, birthDate: birthDateForAge(30), gender, bio: `Bio de ${name}`, city: "E2E-ville", interests }
    }),
    200
  );
  await expectStatus(
    `preferences ${name}`,
    request("/me/preferences", { method: "PUT", token, body: { interestedIn, ageMin: 18, ageMax: 60, maxDistanceKm: 5 } }),
    200
  );
  await expectStatus(`location ${name}`, request("/me/location", { method: "PUT", token, body: { latitude: lat, longitude: lng } }), 204);
  await expectStatus(`photo ${name}`, request("/me/photos", { method: "POST", token, form: photoForm() }), 201);
  return { email, token, id: auth.user.id };
}

/** A random spot on Earth so every run gets its own isolated discovery pool. */
function randomSpot() {
  return { lat: -60 + Math.random() * 120, lng: -170 + Math.random() * 340 };
}

module.exports = {
  API_BASE_URL,
  PASSWORD,
  assert,
  uniqueEmail,
  request,
  expectStatus,
  createMember,
  randomSpot,
  photoForm,
  birthDateForAge
};
