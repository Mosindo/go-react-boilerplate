"use strict";
// Shared helpers for the API smoke test and the UI fixtures (plain Node, no dependencies).

const zlib = require("zlib");

const API_BASE_URL = (process.env.MOBILE_E2E_API_URL || process.env.EXPO_PUBLIC_API_URL || "http://localhost:18080").replace(/\/+$/, "");
const PASSWORD = process.env.MOBILE_E2E_PASSWORD || "Password123";

function uniqueEmail(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}@e2e.invalid`;
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function request(path, { method = "GET", token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload;
  if (form) {
    payload = form;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const response = await fetch(`${API_BASE_URL}${path}`, { method, headers, body: payload });
  const text = await response.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = null;
  }
  return { status: response.status, ok: response.ok, json };
}

const crcTable = (() => {
  const table = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    table[n] = c >>> 0;
  }
  return table;
})();

function crc32(buffer) {
  let c = 0xffffffff;
  for (const byte of buffer) c = crcTable[(c ^ byte) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([length, body, crc]);
}

/** A small gradient PNG (the API accepts images of at least 200x200). */
function gradientPng(width = 400, height = 500, hue = 200) {
  const rows = [];
  for (let y = 0; y < height; y++) {
    const row = Buffer.alloc(1 + width * 3);
    for (let x = 0; x < width; x++) {
      row[1 + x * 3] = (hue + (x * 40) / width) % 256;
      row[2 + x * 3] = (80 + (y * 120) / height) % 256;
      row[3 + x * 3] = 150;
    }
    rows.push(row);
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 2; // RGB
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", zlib.deflateSync(Buffer.concat(rows))),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

function birthDateForAge(age) {
  const d = new Date();
  d.setFullYear(d.getFullYear() - age);
  d.setDate(d.getDate() - 30);
  return d.toISOString().slice(0, 10);
}

/** Registers a user and completes onboarding (profile, preferences, location, one photo). */
async function createCompleteUser({ prefix, name, gender, seeks, age = 30, hue = 200, lat = 48.8566, lng = 2.3522 }) {
  const email = uniqueEmail(prefix);
  const registered = await request("/auth/register", { method: "POST", body: { email, password: PASSWORD } });
  assert(registered.status === 201, `register ${email} failed: ${registered.status} ${JSON.stringify(registered.json)}`);
  const token = registered.json.accessToken;

  const steps = [
    ["PUT", "/me/profile", { firstName: name, birthDate: birthDateForAge(age), gender, bio: `Bonjour, je suis ${name}.`, interests: ["cinema", "voyages"] }],
    ["PUT", "/me/preferences", { interestedIn: seeks, minAge: 18, maxAge: 60, maxDistanceKm: 50 }],
    ["PUT", "/me/location", { latitude: lat, longitude: lng, city: "Paris" }]
  ];
  for (const [method, path, body] of steps) {
    const res = await request(path, { method, token, body });
    assert(res.ok, `${method} ${path} failed: ${res.status} ${JSON.stringify(res.json)}`);
  }

  const form = new FormData();
  form.append("file", new Blob([gradientPng(400, 500, hue)], { type: "image/png" }), "photo.png");
  const photo = await request("/photos", { method: "POST", token, form });
  assert(photo.status === 201, `photo upload failed: ${photo.status} ${JSON.stringify(photo.json)}`);

  return { id: registered.json.user.id, email, token, name };
}

module.exports = { API_BASE_URL, PASSWORD, assert, birthDateForAge, createCompleteUser, gradientPng, request, uniqueEmail };
