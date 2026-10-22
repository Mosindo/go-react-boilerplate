// Shared helpers for the API-level smoke test and the Maestro fixture setup.
const { solidPng } = require("./png");

const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "e2e-password-123";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.test`;
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
  const res = await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  assert(res.status === 201, `register ${email}: HTTP ${res.status}`);
  return { token: res.payload.accessToken, id: res.payload.user.id, email };
}

/** Completes profile, preferences and one photo so the account can use discovery. */
async function onboard(account, { name, gender, lookingFor, latitude = 45.5, longitude = -73.57, color = [180, 90, 110] }) {
  const birthYear = new Date().getFullYear() - 30;
  let res = await request("/profile", {
    method: "PUT",
    token: account.token,
    body: {
      firstName: name,
      birthdate: `${birthYear}-04-10`,
      gender,
      bio: "Compte de test automatisé",
      city: "Montréal",
      latitude,
      longitude,
      interestIds: [1, 2]
    }
  });
  assert(res.status === 200, `profile ${account.email}: HTTP ${res.status} ${JSON.stringify(res.payload)}`);
  res = await request("/preferences", {
    method: "PUT",
    token: account.token,
    body: { interestedIn: [lookingFor], minAge: 18, maxAge: 60, maxDistanceKm: 50 }
  });
  assert(res.status === 200, `preferences ${account.email}: HTTP ${res.status}`);

  const form = new FormData();
  form.append("photo", new Blob([solidPng(480, 600, color)], { type: "image/png" }), "photo.png");
  res = await request("/photos", { method: "POST", token: account.token, form });
  assert(res.status === 201, `photo ${account.email}: HTTP ${res.status} ${JSON.stringify(res.payload)}`);
  return account;
}

module.exports = { API_BASE_URL, PASSWORD, uniqueEmail, request, assert, register, onboard };
