/**
 * Minimal API client shared by the smoke and UI-fixture scripts. It talks to a
 * running API over HTTP only, exactly like the mobile app does.
 */
const { deflateSync } = require("node:zlib");

const API_BASE_URL = process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080";
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "correct-horse-1";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@smoke.test`;
}

async function request(pathname, { method = "GET", token, body, form } = {}) {
  const headers = { Accept: "application/json" };
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

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function expectStatus(label, expected, promise) {
  const res = await promise;
  assert(res.status === expected, `${label}: expected ${expected}, got ${res.status} ${JSON.stringify(res.payload)}`);
  return res.payload;
}

function crc32(buf) {
  let c = ~0;
  for (const byte of buf) {
    c ^= byte;
    for (let k = 0; k < 8; k++) c = (c >>> 1) ^ (0xedb88320 & -(c & 1));
  }
  return ~c >>> 0;
}

function pngChunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}

function makePng(width = 600, height = 800, tint = 80) {
  const rows = [];
  for (let y = 0; y < height; y++) {
    const row = Buffer.alloc(1 + width * 3);
    for (let x = 0; x < width; x++) {
      row[1 + x * 3] = (x * 255) / width;
      row[2 + x * 3] = (y * 255) / height;
      row[3 + x * 3] = tint;
    }
    rows.push(row);
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8;
  header[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    pngChunk("IHDR", header),
    pngChunk("IDAT", deflateSync(Buffer.concat(rows))),
    pngChunk("IEND", Buffer.alloc(0))
  ]);
}

/** Registers a user, completes the profile and uploads one photo. */
async function createMember(prefix, { name, gender, seeking, born = "1993-03-14" }) {
  const email = uniqueEmail(prefix);
  const session = await expectStatus("register", 201, request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }));
  const token = session.accessToken;
  await expectStatus(
    "profile",
    200,
    request("/me/profile", {
      method: "PUT",
      token,
      body: {
        firstName: name,
        birthDate: born,
        gender,
        bio: "Profil de test automatisé.",
        city: "Lyon",
        latitude: 45.76,
        longitude: 4.84,
        interests: ["travel", "music"],
        preferences: { interestedIn: [seeking], ageMin: 18, ageMax: 60, maxDistanceKm: 50 }
      }
    })
  );
  const form = new FormData();
  form.append("file", new Blob([makePng()], { type: "image/png" }), "photo.png");
  await expectStatus("photo", 201, request("/me/photos", { method: "POST", token, form }));
  return { email, token, id: session.user.id };
}

module.exports = { API_BASE_URL, PASSWORD, assert, createMember, expectStatus, request, uniqueEmail };
