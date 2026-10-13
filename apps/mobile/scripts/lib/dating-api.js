// Small dependency-free API client shared by the smoke and E2E scripts.
const zlib = require("zlib");

const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.invalid`;
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function request(path, { method = "GET", token, json, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (json !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    body: form ?? (json !== undefined ? JSON.stringify(json) : undefined)
  });
  const text = await response.text();
  let payload = null;
  try {
    payload = text ? JSON.parse(text) : null;
  } catch {
    payload = text;
  }
  return { status: response.status, ok: response.ok, payload };
}

const crcTable = Array.from({ length: 256 }, (_, n) => {
  let c = n;
  for (let k = 0; k < 8; k += 1) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});

function crc32(buf) {
  let c = 0xffffffff;
  for (const b of buf) c = crcTable[(c ^ b) & 255] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

/** A small gradient PNG built in memory, so no fixture files are needed. */
function makePng(width = 300, height = 400) {
  const stride = width * 3 + 1;
  const raw = Buffer.alloc(stride * height);
  for (let y = 0; y < height; y += 1) {
    for (let x = 0; x < width; x += 1) {
      const o = y * stride + 1 + x * 3;
      raw[o] = 210 - Math.floor(y / 4);
      raw[o + 1] = 90 + Math.floor(x / 4);
      raw[o + 2] = 120;
    }
  }
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type), data]);
    const crc = Buffer.alloc(4);
    crc.writeUInt32BE(crc32(body));
    return Buffer.concat([len, body, crc]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function register(email) {
  const res = await request("/auth/register", { method: "POST", json: { email, password: PASSWORD } });
  assert(res.status === 201, `register failed for ${email}: ${res.status} ${JSON.stringify(res.payload)}`);
  return { id: res.payload.user.id, email, token: res.payload.accessToken, refreshToken: res.payload.refreshToken };
}

/** Registers a user and completes profile, preferences, location and one photo. */
async function createMember({ prefix, name, gender, interestedIn, birthDate = "1994-05-17", lat = 48.86, lon = 2.35 }) {
  const user = await register(uniqueEmail(prefix));
  const profile = await request("/profile", {
    method: "PUT",
    token: user.token,
    json: { firstName: name, birthDate, gender, bio: `Bonjour, je suis ${name}.`, interestIds: [1, 2, 3] }
  });
  assert(profile.status === 200, `profile failed: ${profile.status} ${JSON.stringify(profile.payload)}`);
  const prefs = await request("/profile/preferences", {
    method: "PUT",
    token: user.token,
    json: { interestedIn, minAge: 18, maxAge: 70, maxDistanceKm: null }
  });
  assert(prefs.status === 200, `preferences failed: ${prefs.status}`);
  const location = await request("/profile/location", { method: "PUT", token: user.token, json: { latitude: lat, longitude: lon } });
  assert(location.status === 200, `location failed: ${location.status}`);
  const form = new FormData();
  form.append("file", new Blob([makePng()], { type: "image/png" }), "photo.png");
  const photo = await request("/photos", { method: "POST", token: user.token, form });
  assert(photo.status === 201, `photo failed: ${photo.status} ${JSON.stringify(photo.payload)}`);
  return user;
}

module.exports = { API_BASE_URL, PASSWORD, assert, createMember, makePng, register, request, uniqueEmail };
