import { Buffer } from "node:buffer";
import { deflateSync } from "node:zlib";

const CRC_TABLE = Array.from({ length: 256 }, (_, n) => {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});

function crc32(buf) {
  let c = 0xffffffff;
  for (const b of buf) c = CRC_TABLE[(c ^ b) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}

/** Minimal dependency-free PNG encoder for test fixtures (RGB gradient). */
export function gradientPng(width, height, [r1, g1, b1], [r2, g2, b2]) {
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) {
    const t = y / height;
    const row = y * (width * 3 + 1);
    raw[row] = 0;
    for (let x = 0; x < width; x++) {
      const i = row + 1 + x * 3;
      raw[i] = Math.round(r1 + (r2 - r1) * t);
      raw[i + 1] = Math.round(g1 + (g2 - g1) * t);
      raw[i + 2] = Math.round(b1 + (b2 - b1) * t);
    }
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8;
  header[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

export function apiClient(baseUrl) {
  async function call(path, { method = "GET", token, body, form } = {}) {
    const headers = {};
    if (token) headers.Authorization = `Bearer ${token}`;
    if (body !== undefined) headers["Content-Type"] = "application/json";
    const res = await fetch(`${baseUrl}${path}`, {
      method,
      headers,
      body: form ?? (body !== undefined ? JSON.stringify(body) : undefined)
    });
    const text = await res.text();
    let data = null;
    try {
      data = text ? JSON.parse(text) : null;
    } catch {
      data = text;
    }
    return { status: res.status, data };
  }

  async function expectStatus(promise, status, label) {
    const res = await promise;
    if (res.status !== status) {
      throw new Error(`${label}: expected ${status}, got ${res.status} ${JSON.stringify(res.data)}`);
    }
    return res.data;
  }

  /** Registers and fully onboards a member through the public API. */
  async function createMember({ email, password, firstName, gender, interestedIn, birthdate, latitude, longitude }) {
    const auth = await expectStatus(
      call("/auth/register", { method: "POST", body: { email, password } }),
      201,
      "register"
    );
    const token = auth.accessToken;
    await expectStatus(
      call("/me/profile", {
        method: "PATCH",
        token,
        body: { firstName, birthdate, gender, bio: `Bonjour, je suis ${firstName}.`, interests: ["hiking", "jazz"] }
      }),
      200,
      "profile"
    );
    await expectStatus(
      call("/me/preferences", {
        method: "PATCH",
        token,
        body: { interestedIn, ageMin: 18, ageMax: 70, maxDistanceKm: 30 }
      }),
      200,
      "preferences"
    );
    await expectStatus(
      call("/me/location", { method: "PUT", token, body: { latitude, longitude, city: "Testville" } }),
      200,
      "location"
    );
    const form = new FormData();
    form.append(
      "photo",
      new Blob([gradientPng(600, 750, [46, 74, 98], [156, 197, 161])], { type: "image/png" }),
      "p.png"
    );
    await expectStatus(call("/me/photos", { method: "POST", token, form }), 201, "photo");
    return { id: auth.user.id, token, email, password };
  }

  return { call, expectStatus, createMember };
}
