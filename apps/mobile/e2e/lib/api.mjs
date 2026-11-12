// Tiny dependency-free helpers shared by the API smoke test and the UI E2E test.
import { Buffer } from "node:buffer";
import { deflateSync } from "node:zlib";

export const API_URL = (process.env.E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
export const PASSWORD = "Password123";

export function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@e2e.lueur.invalid`;
}

export function assert(condition, message) {
  if (!condition) throw new Error(`Assertion failed: ${message}`);
}

export async function request(path, { method = "GET", body, token, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API_URL}${path}`, { method, headers, body: form ?? (body !== undefined ? JSON.stringify(body) : undefined) });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  return { status: res.status, data };
}

export async function expectStatus(promise, status, label) {
  const res = await promise;
  assert(res.status === status, `${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`);
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

/** Generates a valid RGB PNG (gradient) so tests need no binary fixtures. */
export function makePng(width = 400, height = 500, hue = 20) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    raw[row] = 0;
    for (let x = 0; x < width; x++) {
      const i = row + 1 + x * 3;
      raw[i] = (hue * 3 + y / 3) & 0xff;
      raw[i + 1] = (x / 2) & 0xff;
      raw[i + 2] = 140;
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk("IHDR", ihdr), chunk("IDAT", deflateSync(raw)), chunk("IEND", Buffer.alloc(0))]);
}

export async function uploadPhoto(token, hue = 20) {
  const form = new FormData();
  form.append("photo", new Blob([makePng(400, 500, hue)], { type: "image/png" }), "photo.png");
  return request("/profile/photos", { method: "POST", token, form });
}

/** Registers and fully onboards a member through the API. */
export async function createMember(prefix, { firstName, gender, interestedIn, birthdate = "1994-05-10" }) {
  const email = uniqueEmail(prefix);
  const auth = await expectStatus(request("/auth/register", { method: "POST", body: { email, password: PASSWORD } }), 201, `register ${prefix}`);
  const token = auth.accessToken;
  await expectStatus(
    request("/profile", {
      method: "PUT",
      token,
      body: { firstName, birthdate, gender, bio: "Profil de test E2E", relationshipGoal: "long_term", city: "Lyon" }
    }),
    200,
    "profile"
  );
  await expectStatus(
    request("/profile/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 100 } }),
    200,
    "preferences"
  );
  await expectStatus(uploadPhoto(token), 201, "photo");
  return { email, token, id: auth.user.id, firstName };
}
