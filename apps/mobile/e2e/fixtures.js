// Shared helpers for the API smoke test and the web UI flow.
const zlib = require("zlib");

const API_URL = (process.env.E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(
  /\/+$/,
  ""
);

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

/** Generates a gradient PNG (no dependency) used as a test profile photo. */
function makePng(width = 480, height = 640, hue = [196, 85, 59]) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    raw[y * (width * 3 + 1)] = 0;
    for (let x = 0; x < width; x++) {
      const i = y * (width * 3 + 1) + 1 + x * 3;
      const t = y / height;
      raw[i] = Math.round(hue[0] * (1 - t) + 40 * t);
      raw[i + 1] = Math.round(hue[1] * (1 - t) + 30 * t);
      raw[i + 2] = Math.round(hue[2] * (1 - t) + 60 * t);
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
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@e2e-test.dev`;
}

async function call(method, path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: form ?? (body !== undefined ? JSON.stringify(body) : undefined)
  });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = text;
  }
  return { status: res.status, json };
}

function expect(res, status, label) {
  if (res.status !== status) {
    throw new Error(`${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.json)}`);
  }
  return res.json;
}

async function registerMember(prefix) {
  const email = uniqueEmail(prefix);
  const password = "Password123";
  const out = expect(await call("POST", "/auth/register", { body: { email, password } }), 201, `register ${prefix}`);
  return { email, password, id: out.user.id, token: out.accessToken };
}

async function onboardMember(m, { firstName, gender, interestedIn, lat, lng, birthdate = "1994-06-15" }) {
  expect(
    await call("PUT", "/me/profile", {
      token: m.token,
      body: { firstName, birthdate, gender, bio: `Bonjour, je suis ${firstName}.`, jobTitle: "Testeur", city: "Lyon" }
    }),
    200,
    `profile ${firstName}`
  );
  expect(
    await call("PUT", "/me/preferences", {
      token: m.token,
      body: { interestedIn, ageMin: 18, ageMax: 70, maxDistanceKm: 50 }
    }),
    200,
    "preferences"
  );
  expect(
    await call("PUT", "/me/interests", { token: m.token, body: { interests: ["hiking", "music", "coffee"] } }),
    200,
    "interests"
  );
  expect(
    await call("PUT", "/me/location", { token: m.token, body: { latitude: lat, longitude: lng } }),
    200,
    "location"
  );
  const form = new FormData();
  form.append("photo", new Blob([makePng()], { type: "image/png" }), "photo.png");
  expect(await call("POST", "/me/photos", { token: m.token, form }), 201, "photo");
}

module.exports = { API_URL, makePng, uniqueEmail, call, expect, registerMember, onboardMember };
