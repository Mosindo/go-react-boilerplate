// Shared helpers for the API smoke test and the web E2E test.
const zlib = require("zlib");

const API_BASE_URL = (process.env.E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Password123";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@e2e.lueur.test`;
}

function crc32(buf) {
  let c;
  let crc = 0xffffffff;
  for (let n = 0; n < buf.length; n++) {
    c = (crc ^ buf[n]) & 0xff;
    for (let k = 0; k < 8; k++) {
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    }
    crc = (crc >>> 8) ^ c;
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}

/** Generates a valid gradient PNG (default 320x400) without dependencies. */
function makePng(width = 320, height = 400, hue = 0) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    raw[row] = 0;
    for (let x = 0; x < width; x++) {
      raw[row + 1 + x * 3] = (180 + hue + (x % 60)) & 0xff;
      raw[row + 2 + x * 3] = (90 + (y % 120)) & 0xff;
      raw[row + 3 + x * 3] = 70;
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function request(path, { method = "GET", body, token, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_BASE_URL}${path}`, {
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

function expectStatus(res, status, label) {
  if (res.status !== status) {
    throw new Error(`${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.payload)}`);
  }
  return res.payload;
}

/** Creates a complete, discoverable member through the public API. */
async function createMember({ prefix, firstName, gender, interestedIn, latitude = 48.8566, longitude = 2.3522, withLocation = true }) {
  const email = uniqueEmail(prefix);
  const auth = expectStatus(await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }), 201, "register");
  const token = auth.accessToken;
  expectStatus(
    await request("/me/profile", {
      method: "PUT",
      token,
      body: { firstName, birthdate: "1994-05-17", gender, bio: "Profil de test E2E", relationshipGoal: "long_term" }
    }),
    200,
    "profile"
  );
  expectStatus(
    await request("/me/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 70, maxDistanceKm: 50 } }),
    200,
    "preferences"
  );
  if (withLocation) {
    expectStatus(await request("/me/location", { method: "PUT", token, body: { latitude, longitude, city: "Paris" } }), 200, "location");
  }
  const form = new FormData();
  form.append("photo", new Blob([makePng()], { type: "image/png" }), "photo.png");
  expectStatus(await request("/me/photos", { method: "POST", token, form }), 201, "photo upload");
  return { email, token, userId: auth.user.id, firstName };
}

module.exports = { API_BASE_URL, PASSWORD, createMember, expectStatus, makePng, request, uniqueEmail };
