/* Shared helpers for the node-based e2e scripts (live API, no mocks). */
const zlib = require("zlib");
const { Buffer } = require("buffer");

const API_URL = (
  process.env.EXPO_PUBLIC_API_URL ||
  process.env.API_URL ||
  "http://localhost:18080"
).replace(/\/+$/, "");
const PASSWORD = "Cleo-test-2025";

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@cleo.test`;
}

async function call(path, { method = "GET", token, body, form } = {}) {
  const headers = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) {
    payload = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const response = await fetch(`${API_URL}${path}`, { method, headers, body: payload });
  const text = await response.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = null;
  }
  return { status: response.status, data, headers: response.headers };
}

function expectStatus(res, status, label) {
  assert(
    res.status === status,
    `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`
  );
  return res.data;
}

/* ---------------------------- tiny PNG generator ---------------------------- */

const CRC_TABLE = (() => {
  const table = new Uint32Array(256);
  for (let n = 0; n < 256; n += 1) {
    let c = n;
    for (let k = 0; k < 8; k += 1) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    table[n] = c >>> 0;
  }
  return table;
})();

function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) crc = CRC_TABLE[(crc ^ byte) & 0xff] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([length, body, crc]);
}

/** Valid RGB PNG (default 160x160 gradient: server requires >= 64 px per side). */
function makePng(width = 160, height = 160, seed = 0) {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  const rows = [];
  for (let y = 0; y < height; y += 1) {
    const row = Buffer.alloc(1 + width * 3);
    for (let x = 0; x < width; x += 1) {
      row[1 + x * 3] = (x * 255) / width;
      row[2 + x * 3] = (y * 255) / height;
      row[3 + x * 3] = (seed * 60) % 256;
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

function photoForm(seed = 0) {
  const form = new FormData();
  form.append("photo", new Blob([makePng(160, 160, seed)], { type: "image/png" }), "photo.png");
  return form;
}

/* --------------------------------- fixtures --------------------------------- */

async function registerUser(prefix) {
  const email = uniqueEmail(prefix);
  const auth = expectStatus(
    await call("/auth/register", { method: "POST", body: { email, password: PASSWORD } }),
    201,
    `register ${email}`
  );
  assert(auth.accessToken && auth.refreshToken && auth.user?.id, "register: tokens/user missing");
  return { email, token: auth.accessToken, refreshToken: auth.refreshToken, id: auth.user.id };
}

/** Creates a complete, discoverable profile (profile + preferences + one photo). */
async function onboardUser(user, profile, preferences, seed) {
  expectStatus(
    await call("/me/profile", { method: "PUT", token: user.token, body: profile }),
    200,
    `profile ${user.email}`
  );
  expectStatus(
    await call("/me/preferences", { method: "PUT", token: user.token, body: preferences }),
    200,
    `preferences ${user.email}`
  );
  const photo = expectStatus(
    await call("/me/photos", { method: "POST", token: user.token, form: photoForm(seed) }),
    201,
    `photo ${user.email}`
  );
  user.photo = photo;
  return photo;
}

const CITY = "Cleo-Ville";

function profileInput(firstName, gender, birthDate, interests = []) {
  return {
    firstName,
    birthDate,
    gender,
    bio: "Profil de test automatique.",
    city: CITY,
    interests
  };
}

function preferenceInput(interestedIn) {
  return {
    interestedIn,
    minAge: 18,
    maxAge: 99,
    maxDistanceKm: 500,
    discoverable: true,
    showDistance: true
  };
}

async function deleteUser(user) {
  return call("/me", { method: "DELETE", token: user.token, body: { password: PASSWORD } });
}

module.exports = {
  API_URL,
  PASSWORD,
  assert,
  call,
  expectStatus,
  makePng,
  photoForm,
  registerUser,
  onboardUser,
  profileInput,
  preferenceInput,
  deleteUser
};
