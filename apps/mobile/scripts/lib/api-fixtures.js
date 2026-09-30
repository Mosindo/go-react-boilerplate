// Shared helpers for API-driven E2E scripts (Node 20+, global fetch/FormData).
const fs = require("fs");
const path = require("path");

const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123";
// Scripts run from apps/mobile (npm run ...).
const PHOTO = path.resolve(process.cwd(), "e2e", "web", "fixtures", "portrait.png");

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.test`;
}

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
    body: form ?? (body !== undefined ? JSON.stringify(body) : undefined)
  });
  const text = await response.text();
  let payload = null;
  try {
    payload = text ? JSON.parse(text) : null;
  } catch {
    payload = text;
  }
  return { status: response.status, payload };
}

async function expectStatus(promise, status, label) {
  const res = await promise;
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

async function register(prefix) {
  const email = uniqueEmail(prefix);
  const payload = await expectStatus(request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }), 201, `register ${prefix}`);
  return { email, token: payload.accessToken, userId: payload.user.id };
}

async function completeProfile(user, { firstName, birthdate, gender, interestedIn }) {
  await expectStatus(request("/profile", { method: "PATCH", token: user.token, body: { firstName, birthdate, gender, bio: `Bonjour, je suis ${firstName}.` } }), 200, "profile");
  await expectStatus(
    request("/profile/preferences", { method: "PUT", token: user.token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 100 } }),
    200,
    "preferences"
  );
  const form = new FormData();
  form.append("photo", new Blob([fs.readFileSync(PHOTO)], { type: "image/png" }), "photo.png");
  await expectStatus(request("/profile/photos", { method: "POST", token: user.token, form }), 201, "photo upload");
}

module.exports = { API_BASE_URL, PASSWORD, assert, completeProfile, expectStatus, register, request };
