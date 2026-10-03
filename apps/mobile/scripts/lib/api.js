/* eslint-disable no-console */
// Tiny helpers shared by the HTTP-level end-to-end scripts. They talk to a running API only.
const zlib = require("zlib");

const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123";

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.invalid`;
}

async function request(path, { token, json, body, method } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (json !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: method || (json !== undefined || body ? "POST" : "GET"),
    headers,
    body: json !== undefined ? JSON.stringify(json) : body
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

function crc32(buf) {
  let c;
  let crc = 0xffffffff;
  for (let n = 0; n < buf.length; n++) {
    c = (crc ^ buf[n]) & 0xff;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
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

/** A small valid PNG (gradient) so uploads exercise the real image pipeline. */
function samplePng(size = 64) {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const rows = [];
  for (let y = 0; y < size; y++) {
    const row = Buffer.alloc(1 + size * 3);
    for (let x = 0; x < size; x++) {
      row[1 + x * 3] = (x * 4) & 255;
      row[2 + x * 3] = (y * 4) & 255;
      row[3 + x * 3] = 140;
    }
    rows.push(row);
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", zlib.deflateSync(Buffer.concat(rows))),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

async function register(email) {
  const res = await request("/auth/register", { json: { email, password: PASSWORD } });
  assert(res.status === 201, `register ${email}: ${res.status} ${JSON.stringify(res.payload)}`);
  return { id: res.payload.user.id, email, token: res.payload.accessToken };
}

async function uploadPhoto(token) {
  const form = new FormData();
  form.append("file", new Blob([samplePng()], { type: "image/png" }), "e2e.png");
  const res = await request("/me/photos", { token, body: form, method: "POST" });
  assert(res.status === 201, `photo upload: ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

/** Registers a user with a complete, discoverable profile near `origin`. */
async function createMember(prefix, { name, gender, interestedIn, origin }) {
  const user = await register(uniqueEmail(prefix));
  const birthYear = new Date().getFullYear() - 30;
  let res = await request("/me/profile", {
    token: user.token,
    method: "PUT",
    json: {
      firstName: name,
      birthDate: `${birthYear}-03-14`,
      gender,
      bio: "E2E member",
      city: "Testville",
      interests: ["travel", "music"],
      latitude: origin.latitude,
      longitude: origin.longitude
    }
  });
  assert(res.status === 200, `profile: ${res.status} ${JSON.stringify(res.payload)}`);
  res = await request("/me/preferences", {
    token: user.token,
    method: "PUT",
    json: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 50 }
  });
  assert(res.status === 200, `preferences: ${res.status}`);
  await uploadPhoto(user.token);
  return user;
}

/** A random point so concurrent runs and old fixtures never see each other. */
function randomOrigin() {
  return { latitude: -60 + Math.random() * 120, longitude: -170 + Math.random() * 340 };
}

module.exports = { API_BASE_URL, PASSWORD, assert, createMember, randomOrigin, register, request, samplePng, uniqueEmail, uploadPhoto };
