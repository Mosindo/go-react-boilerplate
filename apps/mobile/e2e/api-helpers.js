// Small API client + PNG generator used to drive the "other member" side of
// the end-to-end scenario without a second browser.
const zlib = require("node:zlib");

const API = process.env.E2E_API_URL || "http://localhost:18080";

function crc32(buf) {
  let c;
  const table = crc32.table || (crc32.table = Array.from({ length: 256 }, (_, n) => {
    c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  }));
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

/** Generates an RGB gradient PNG (valid input for the photo pipeline). */
function gradientPng(width = 480, height = 600, hue = 0) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    raw[y * (width * 3 + 1)] = 0;
    for (let x = 0; x < width; x++) {
      const i = y * (width * 3 + 1) + 1 + x * 3;
      raw[i] = (200 + hue) % 256;
      raw[i + 1] = Math.floor((x / width) * 160);
      raw[i + 2] = Math.floor((y / height) * 200);
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

async function call(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";
  const res = await fetch(API + path, { method, headers, body: form ?? (body ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status} ${text}`);
  return data;
}

/** Creates a complete member through the public API. */
async function createMember({ email, password, firstName, gender, interestedIn, lat, lng }) {
  const session = await call("POST", "/auth/register", { body: { email, password } });
  const token = session.accessToken;
  await call("PUT", "/me/profile", {
    token,
    body: { firstName, birthdate: "1992-03-14", gender, bio: "Membre de test end-to-end.", lookingFor: "relationship", interests: ["hiking", "coffee"] }
  });
  await call("PUT", "/me/preferences", { token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 100 } });
  await call("PUT", "/me/location", { token, body: { latitude: lat, longitude: lng, city: "Lyon" } });
  const form = new FormData();
  form.append("photo", new Blob([gradientPng(480, 600, firstName.length * 30)], { type: "image/png" }), "photo.png");
  await call("POST", "/me/photos", { token, form });
  return { token, id: session.user.id, email, password };
}

module.exports = { API, call, createMember, gradientPng };
