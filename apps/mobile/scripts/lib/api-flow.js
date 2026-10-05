// Shared helpers for the API-level E2E scripts (Node >= 20: global fetch, FormData and Blob).
const fs = require("fs");
const path = require("path");

const API_BASE_URL = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "e2e-password-123";
const PHOTO = path.join(__dirname, "../../e2e/fixtures/portrait.jpg");
const PARIS = { latitude: 48.8566, longitude: 2.3522 };

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function request(pathname, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_BASE_URL}${pathname}`, {
    method,
    headers,
    body: form ?? (body === undefined ? undefined : JSON.stringify(body))
  });
  const text = await response.text();
  try {
    return { status: response.status, payload: text ? JSON.parse(text) : null };
  } catch {
    return { status: response.status, payload: text };
  }
}

function expectStatus(res, status, label) {
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

const letters = (n) => Array.from({ length: n }, () => String.fromCharCode(97 + Math.floor(Math.random() * 26))).join("");

/** Registers a user and completes onboarding through the public API (profile, preferences, location, photo). */
async function createOnboardedUser({ label, gender, interestedIn, birthDate = "1992-06-15" }) {
  const email = `${label}.${Date.now()}.${letters(4)}@e2e.test`;
  const session = expectStatus(
    await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }),
    201,
    `register ${label}`
  );
  const token = session.accessToken;
  const name = label.charAt(0).toUpperCase() + label.slice(1) + letters(5);

  expectStatus(
    await request("/profile/me", {
      method: "PUT",
      token,
      body: { firstName: name, birthDate, gender, bio: `Bio de ${name}`, occupation: "Testeur", interests: ["travel", "music"] }
    }),
    200,
    `profile ${label}`
  );
  expectStatus(
    await request("/profile/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 60, maxDistanceKm: 50 } }),
    200,
    `preferences ${label}`
  );
  expectStatus(await request("/profile/location", { method: "PUT", token, body: { ...PARIS, city: "Paris" } }), 200, `location ${label}`);

  const form = new FormData();
  form.append("photo", new Blob([fs.readFileSync(PHOTO)], { type: "image/jpeg" }), "portrait.jpg");
  expectStatus(await request("/photos", { method: "POST", token, form }), 201, `photo ${label}`);

  return { id: session.user.id, email, name, token, refreshToken: session.refreshToken };
}

module.exports = { API_BASE_URL, PASSWORD, assert, request, expectStatus, createOnboardedUser };
