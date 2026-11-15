/* Shared helpers for the API and web E2E scripts (no dependencies). */
const { Buffer } = require("buffer");
const zlib = require("zlib");

const API_URL = (process.env.E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");

async function request(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) payload = form;
  else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(API_URL + path, { method, headers, body: payload });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  return { status: res.status, data };
}

function expectStatus(res, status, label) {
  if (res.status !== status) {
    throw new Error(`${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`);
  }
  return res.data;
}

const crcTable = Array.from({ length: 256 }, (_, n) => {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});
function crc32(buf) {
  let c = 0xffffffff;
  for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}
function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const td = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(td));
  return Buffer.concat([len, td, crc]);
}

/** Generates a gradient PNG so tests never depend on binary fixtures. */
function gradientPng(width = 480, height = 640, hue = [242, 145, 95]) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    raw[row] = 0;
    for (let x = 0; x < width; x++) {
      const t = y / height;
      raw[row + 1 + x * 3] = Math.round(hue[0] * (1 - t) + 40 * t);
      raw[row + 2 + x * 3] = Math.round(hue[1] * (1 - t) + 20 * t) ^ (x & 16);
      raw[row + 3 + x * 3] = Math.round(hue[2] * (1 - t) + 50 * t);
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

function uniqueEmail(prefix) {
  return `${prefix}.${Date.now()}.${Math.floor(Math.random() * 1e5)}@e2e.test`;
}

/** Creates a fully onboarded user through the public API. */
async function createCompleteUser({
  name,
  gender,
  seeking,
  birthdate = "1993-05-20",
  lat = 48.8566,
  lng = 2.3522,
  password = "Secret123"
}) {
  const email = uniqueEmail(name.toLowerCase());
  const session = expectStatus(
    await request("POST", "/auth/register", { body: { email, password, acceptTerms: true } }),
    201,
    `register ${name}`
  );
  const token = session.accessToken;
  expectStatus(
    await request("PUT", "/me/profile", {
      token,
      body: { firstName: name, birthdate, gender, bio: `Bonjour, moi c'est ${name}.`, interestIds: [1, 2, 4] }
    }),
    200,
    `profile ${name}`
  );
  expectStatus(
    await request("PUT", "/me/preferences", { token, body: { interestedIn: seeking, minAge: 18, maxAge: 99, maxDistanceKm: 100 } }),
    200,
    `prefs ${name}`
  );
  expectStatus(
    await request("PUT", "/me/location", { token, body: { latitude: lat, longitude: lng, city: "Paris" } }),
    200,
    `location ${name}`
  );
  const form = new FormData();
  form.append("photo", new Blob([gradientPng()], { type: "image/png" }), "photo.png");
  expectStatus(await request("POST", "/me/photos", { token, form }), 201, `photo ${name}`);
  return { email, password, token, id: session.user.id, name };
}

module.exports = { API_URL, request, expectStatus, gradientPng, uniqueEmail, createCompleteUser };
