import { deflateSync } from "node:zlib";

export const API = (process.env.E2E_API_URL ?? "http://localhost:18080").replace(/\/+$/, "");
export const PASSWORD = "Password123";

type Res<T> = { status: number; data: T };

export async function call<T = unknown>(
  path: string,
  opts: { method?: string; body?: unknown; token?: string; form?: FormData } = {}
): Promise<Res<T>> {
  const headers: Record<string, string> = {};
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";
  if (opts.token) headers.Authorization = `Bearer ${opts.token}`;
  const res = await fetch(`${API}${path}`, {
    method: opts.method ?? "GET",
    headers,
    body: opts.form ?? (opts.body !== undefined ? JSON.stringify(opts.body) : undefined)
  });
  const text = await res.text();
  return { status: res.status, data: (text ? JSON.parse(text) : null) as T };
}

export function uniqueEmail(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@lueur.test`;
}

/** A valid 400×500 PNG built in memory. */
export function makePng(rgb: [number, number, number] = [240, 169, 121]): Buffer {
  const w = 400;
  const h = 500;
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const o = y * (w * 3 + 1) + 1 + x * 3;
      raw[o] = rgb[0];
      raw[o + 1] = rgb[1];
      raw[o + 2] = rgb[2];
    }
  }
  const table = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (buf: Buffer) => {
    let c = 0xffffffff;
    for (const b of buf) c = table[(c ^ b) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (type: string, data: Buffer) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const td = Buffer.concat([Buffer.from(type), data]);
    const c = Buffer.alloc(4);
    c.writeUInt32BE(crc(td));
    return Buffer.concat([len, td, c]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0))
  ]);
}

export type Member = { id: string; token: string; email: string };

/** Creates a fully onboarded member through the API. */
export async function createMember(name: string, gender: string, interestedIn: string[]): Promise<Member> {
  const email = uniqueEmail(`e2e_${name.toLowerCase()}`);
  const reg = await call<{ accessToken: string; user: { id: string } }>("/auth/register", {
    method: "POST",
    body: { email, password: PASSWORD }
  });
  if (reg.status !== 201) throw new Error(`register ${name}: ${reg.status}`);
  const token = reg.data.accessToken;
  await call("/me/profile", {
    method: "PATCH",
    token,
    body: { firstName: name, birthdate: "1991-03-08", gender, bio: `Je suis ${name}.`, interests: ["coffee", "hiking"] }
  });
  await call("/me/preferences", { method: "PUT", token, body: { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 50 } });
  await call("/me/location", { method: "PUT", token, body: { latitude: 48.86, longitude: 2.34, city: "Paris" } });
  const form = new FormData();
  form.append("photo", new Blob([new Uint8Array(makePng([127, 209, 174]))], { type: "image/png" }), "photo.png");
  const photo = await call("/me/photos", { method: "POST", token, form });
  if (photo.status !== 201) throw new Error(`photo ${name}: ${photo.status}`);
  return { id: reg.data.user.id, token, email };
}

export async function login(email: string): Promise<string> {
  const res = await call<{ accessToken: string }>("/auth/login", { method: "POST", body: { email, password: PASSWORD } });
  return res.data.accessToken;
}

export async function deleteMember(token: string): Promise<void> {
  await call("/me", { method: "DELETE", token, body: { password: PASSWORD } });
}
