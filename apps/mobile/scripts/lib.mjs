// Shared helpers for the end-to-end scripts (Node 20+, no dependencies).
import { deflateSync } from "node:zlib";

export const API_URL = (process.env.E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
export const PASSWORD = "Password123";

export function uniqueEmail(prefix) {
  return `${prefix}.${Date.now()}.${Math.floor(Math.random() * 1e6)}@e2e.lueur.test`;
}

export function assert(condition, message) {
  if (!condition) {
    throw new Error(`Assertion failed: ${message}`);
  }
}

export async function api(path, { method = "GET", token, body, form } = {}, attempt = 0) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API_URL}${path}`, { method, headers, body: form ?? (body === undefined ? undefined : JSON.stringify(body)) });
  // Behave like a polite client when the API rate limits us.
  if (res.status === 429 && attempt < 5) {
    const wait = Number(res.headers.get("Retry-After") || "2");
    await new Promise((r) => setTimeout(r, Math.min(wait, 30) * 1000));
    return api(path, { method, token, body, form }, attempt + 1);
  }
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

// Minimal PNG encoder so the scripts can upload real images without dependencies.
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
  const body = Buffer.concat([Buffer.from(type), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}
export function pngBuffer(width = 360, height = 480, rgb = [232, 119, 90]) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const row = y * (width * 3 + 1);
    for (let x = 0; x < width; x++) {
      const shade = 1 - y / (height * 1.6);
      raw[row + 1 + x * 3] = Math.round(rgb[0] * shade);
      raw[row + 2 + x * 3] = Math.round(rgb[1] * shade);
      raw[row + 3 + x * 3] = Math.round(rgb[2] * shade);
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk("IHDR", ihdr), chunk("IDAT", deflateSync(raw)), chunk("IEND", Buffer.alloc(0))]);
}

export async function uploadPhoto(token, rgb) {
  const form = new FormData();
  form.append("photo", new Blob([pngBuffer(360, 480, rgb)], { type: "image/png" }), "photo.png");
  return expectStatus(api("/profile/photos", { method: "POST", token, form }), 201, "upload photo");
}

/** Registers and fully onboards a member through the API. */
export async function createMember({ name, gender, interestedIn, birthdate = "1993-04-12", lat = 48.8566, lng = 2.3522 }) {
  const email = uniqueEmail(name.toLowerCase());
  const session = await expectStatus(api("/auth/register", { method: "POST", body: { email, password: PASSWORD } }), 201, `register ${name}`);
  const token = session.accessToken;
  await expectStatus(
    api("/profile", { method: "PUT", token, body: { firstName: name, birthdate, gender, bio: `Je suis ${name}.`, interestIds: [1, 2, 3] } }),
    200,
    `profile ${name}`
  );
  await expectStatus(
    api("/profile/preferences", { method: "PUT", token, body: { interestedIn, ageMin: 18, ageMax: 99, maxDistanceKm: 100 } }),
    200,
    `prefs ${name}`
  );
  await expectStatus(api("/profile/location", { method: "PUT", token, body: { latitude: lat, longitude: lng } }), 204, `location ${name}`);
  await uploadPhoto(token, [120, 90, 200]);
  return { email, token, id: session.user.id, name };
}
