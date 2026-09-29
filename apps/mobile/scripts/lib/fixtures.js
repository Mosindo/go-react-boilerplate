// Shared helpers for the Node-based end-to-end scripts (API smoke + UI fixtures).
const zlib = require("zlib");

const API_BASE_URL = (
  process.env.MOBILE_E2E_API_URL ||
  process.env.EXPO_PUBLIC_API_URL ||
  "http://localhost:18080"
).replace(/\/+$/, "");
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123";
const BIRTH_DATE = "1992-04-15";
// Two nearby points (Lyon centre). The server rounds coordinates to 2 decimals.
const LOCATION = { latitude: 45.764, longitude: 4.8357 };

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@alba.test`;
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

/** Minimal JSON/multipart client. `form` wins over `body`. */
async function request(path, { method = "GET", token, body, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  let payloadBody;
  if (form) {
    payloadBody = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payloadBody = JSON.stringify(body);
  }
  const response = await fetch(`${API_BASE_URL}${path}`, { method, headers, body: payloadBody });
  const raw = await response.text();
  let payload = null;
  try {
    payload = raw ? JSON.parse(raw) : null;
  } catch {
    payload = raw;
  }
  return { status: response.status, ok: response.ok, payload, headers: response.headers };
}

function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit += 1) {
      crc = crc & 1 ? (crc >>> 1) ^ 0xedb88320 : crc >>> 1;
    }
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function pngChunk(type, data) {
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const typeAndData = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(typeAndData));
  return Buffer.concat([length, typeAndData, crc]);
}

/** Builds a valid solid-colour RGB PNG (default 64x64). */
function makePng(width = 64, height = 64, [red, green, blue] = [194, 69, 45]) {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // colour type: truecolour
  const row = Buffer.alloc(1 + width * 3);
  for (let x = 0; x < width; x += 1) {
    row[1 + x * 3] = red;
    row[2 + x * 3] = green;
    row[3 + x * 3] = blue;
  }
  const raw = Buffer.concat(Array.from({ length: height }, () => row));
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    pngChunk("IHDR", header),
    pngChunk("IDAT", zlib.deflateSync(raw)),
    pngChunk("IEND", Buffer.alloc(0))
  ]);
}

async function uploadPhoto(token, color) {
  const form = new FormData();
  form.append("file", new Blob([makePng(64, 64, color)], { type: "image/png" }), "photo.png");
  return request("/me/photos", { method: "POST", token, form });
}

async function registerUser(email, { password = PASSWORD, birthDate = BIRTH_DATE } = {}) {
  const res = await request("/auth/register", { method: "POST", body: { email, password, birthDate } });
  assert(res.status === 201, `register failed for ${email}: ${res.status} ${JSON.stringify(res.payload)}`);
  const accessToken = res.payload.accessToken || res.payload.token;
  assert(accessToken && res.payload.refreshToken, "register must return access and refresh tokens");
  return { ...res.payload, accessToken, token: accessToken };
}

/** Fills the whole onboarding through the API so the account is discoverable. */
async function completeProfile(auth, { firstName, gender, interestedIn, city = "Lyon" }) {
  const token = auth.accessToken;
  const profile = await request("/me/profile", {
    method: "PUT",
    token,
    body: {
      firstName,
      gender,
      bio: `Hi, I'm ${firstName}. Testing Alba.`,
      city,
      interests: [],
      showDistance: true,
      showAge: true,
      discoverable: true
    }
  });
  assert(profile.status === 200, `PUT /me/profile failed: ${profile.status} ${JSON.stringify(profile.payload)}`);
  const prefs = await request("/me/preferences", {
    method: "PUT",
    token,
    body: { interestedIn, ageMin: 18, ageMax: 99, maxDistanceKm: 50 }
  });
  assert(prefs.status === 200, `PUT /me/preferences failed: ${prefs.status}`);
  const location = await request("/me/location", { method: "PUT", token, body: LOCATION });
  assert(location.status === 204, `PUT /me/location failed: ${location.status}`);
  const photo = await uploadPhoto(token);
  assert(photo.status === 201, `POST /me/photos failed: ${photo.status} ${JSON.stringify(photo.payload)}`);
  return { profile: profile.payload, photo: photo.payload };
}

module.exports = {
  API_BASE_URL,
  BIRTH_DATE,
  LOCATION,
  PASSWORD,
  assert,
  completeProfile,
  makePng,
  registerUser,
  request,
  uniqueEmail,
  uploadPhoto
};
