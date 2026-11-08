// Shared helpers for the end-to-end scripts (plain Node >= 18, no dependency).
const zlib = require("zlib");

const API_URL = (
  process.env.E2E_API_URL ||
  process.env.EXPO_PUBLIC_API_URL ||
  "http://localhost:18080"
).replace(/\/+$/, "");
const PASSWORD = "Password123!";

function assert(condition, message) {
  if (!condition) throw new Error(`ASSERTION FAILED: ${message}`);
}

async function request(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: form ?? (body === undefined ? undefined : JSON.stringify(body))
  });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = null;
  }
  return { status: res.status, json };
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

/** Minimal RGB PNG encoder so the scripts can upload a real image without dependencies. */
function makePng(width, height, [r, g, b]) {
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
  const row = Buffer.alloc(1 + width * 3);
  for (let x = 0; x < width; x++) {
    row[1 + x * 3] = (r + x) % 256;
    row[2 + x * 3] = g;
    row[3 + x * 3] = b;
  }
  const raw = Buffer.concat(Array.from({ length: height }, () => row));
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

let counter = 0;
function uniqueEmail(prefix) {
  counter += 1;
  return `${prefix}.${Date.now()}.${counter}@e2e.invalid`;
}

/** Registers an account through the API and completes its profile (photo, location, preferences). */
async function createCompleteUser({
  firstName,
  gender,
  interestedIn,
  latitude,
  longitude,
  birthDate = "1992-04-18"
}) {
  const email = uniqueEmail(firstName.toLowerCase());
  const reg = await request("POST", "/auth/register", {
    body: { email, password: PASSWORD, firstName, birthDate }
  });
  assert(reg.status === 201, `register ${firstName}: ${reg.status} ${JSON.stringify(reg.json)}`);
  const token = reg.json.accessToken;
  const put = async (path, body) =>
    assert((await request("PUT", path, { token, body })).status === 200, `PUT ${path} failed`);
  await put("/profile", { gender, bio: `Salut, moi c'est ${firstName}.`, interests: ["music", "hiking"] });
  await put("/profile/location", { latitude, longitude, city: "Nantes" });
  await put("/preferences", { interestedIn, ageMin: 18, ageMax: 60, maxDistanceKm: 50 });
  const form = new FormData();
  form.append("photo", new Blob([makePng(400, 500, [200, 90, 60])], { type: "image/png" }), "p.png");
  const photo = await request("POST", "/photos", { token, form });
  assert(photo.status === 201, `photo upload ${firstName}: ${photo.status}`);
  return { email, token, id: reg.json.user.id, firstName };
}

module.exports = { API_URL, PASSWORD, assert, request, makePng, uniqueEmail, createCompleteUser };
