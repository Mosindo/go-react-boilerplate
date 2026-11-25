// Shared helpers for the end-to-end scripts (no dependencies beyond Node 20).
const zlib = require("zlib");

const API_URL = (process.env.E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = "Password123";

function uniqueEmail(prefix) {
  return `${prefix}.${Date.now()}.${Math.floor(Math.random() * 1e6)}@e2e.lueur.test`;
}

async function request(path, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) {
    payload = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(`${API_URL}${path}`, { method, headers, body: payload });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  return { status: res.status, data };
}

function assert(condition, message) {
  if (!condition) throw new Error(`Assertion failed: ${message}`);
}

async function expectStatus(promise, status, label) {
  const res = await promise;
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`);
  return res.data;
}

// Minimal PNG encoder used to generate valid test photos.
function crc32(buf) {
  let c;
  const table = [];
  for (let n = 0; n < 256; n++) {
    c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    table[n] = c >>> 0;
  }
  let crc = 0xffffffff;
  for (const b of buf) crc = table[(crc ^ b) & 0xff] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const td = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(td));
  return Buffer.concat([len, td, crc]);
}

function makePng(width = 480, height = 600, [r, g, b] = [238, 122, 95]) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    raw[row] = 0;
    for (let x = 0; x < width; x++) {
      const i = row + 1 + x * 3;
      raw[i] = Math.min(255, r + (x % 64));
      raw[i + 1] = Math.min(255, g + (y % 64));
      raw[i + 2] = b;
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

async function register(prefix) {
  const email = uniqueEmail(prefix);
  const data = await expectStatus(
    request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }),
    201,
    `register ${prefix}`
  );
  return { email, token: data.accessToken, id: data.user.id };
}

async function onboard(user, { firstName, gender, interestedIn, latitude, longitude, birthdate = "1993-06-15" }) {
  await expectStatus(
    request("/me/profile", {
      method: "PUT",
      token: user.token,
      body: { firstName, birthdate, gender, bio: "Profil de test E2E", interestIds: [1, 2, 3] }
    }),
    200,
    "profile"
  );
  await expectStatus(
    request("/me/preferences", { method: "PUT", token: user.token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 50 } }),
    200,
    "preferences"
  );
  const form = new FormData();
  form.append("photo", new Blob([makePng()], { type: "image/png" }), "photo.png");
  await expectStatus(request("/me/photos", { method: "POST", token: user.token, form }), 201, "photo");
  await expectStatus(request("/me/location", { method: "PUT", token: user.token, body: { latitude, longitude } }), 200, "location");
}

module.exports = { API_URL, PASSWORD, assert, expectStatus, makePng, onboard, register, request, uniqueEmail };
