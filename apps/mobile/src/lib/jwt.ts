/** Seconds-since-epoch expiry of a JWT, or null if it cannot be read. Never verifies the signature. */
export function jwtExpiry(token: string): number | null {
  const part = token.split(".")[1];
  if (!part) {
    return null;
  }
  try {
    const base64 = part.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
    const json = decodeBase64(padded);
    const exp = (JSON.parse(json) as { exp?: unknown }).exp;
    return typeof exp === "number" ? exp : null;
  } catch {
    return null;
  }
}

/** True when the token is missing, unreadable or expires within `skewSeconds`. */
export function isExpiringSoon(token: string, nowMs: number, skewSeconds = 30): boolean {
  const exp = jwtExpiry(token);
  return exp === null || exp * 1000 - nowMs < skewSeconds * 1000;
}

const B64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

function decodeBase64(input: string): string {
  let bits = 0;
  let value = 0;
  const bytes: number[] = [];
  for (const ch of input) {
    if (ch === "=") {
      break;
    }
    const idx = B64.indexOf(ch);
    if (idx < 0) {
      throw new Error("invalid base64");
    }
    value = (value << 6) | idx;
    bits += 6;
    if (bits >= 8) {
      bits -= 8;
      bytes.push((value >> bits) & 0xff);
    }
  }
  let out = "";
  for (let i = 0; i < bytes.length;) {
    const b = bytes[i++];
    if (b < 0x80) {
      out += String.fromCharCode(b);
    } else if (b < 0xe0) {
      out += String.fromCharCode(((b & 0x1f) << 6) | (bytes[i++] & 0x3f));
    } else {
      out += String.fromCharCode(((b & 0x0f) << 12) | ((bytes[i++] & 0x3f) << 6) | (bytes[i++] & 0x3f));
    }
  }
  return out;
}
