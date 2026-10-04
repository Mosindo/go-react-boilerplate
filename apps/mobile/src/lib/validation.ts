export const PASSWORD_MIN = 8;
/** The backend (bcrypt) accepts at most 72 bytes. */
export const PASSWORD_MAX_BYTES = 72;

export function byteLength(value: string): number {
  return new TextEncoder().encode(value).length;
}

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) return "Enter your email.";
  if (email.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email)) return "Enter a valid email address.";
  return null;
}

export function validatePassword(value: string): string | null {
  if (!value) return "Enter a password.";
  if (value.length < PASSWORD_MIN) return `Use at least ${PASSWORD_MIN} characters.`;
  if (byteLength(value) > PASSWORD_MAX_BYTES)
    return `Use at most ${PASSWORD_MAX_BYTES} bytes (about ${PASSWORD_MAX_BYTES} characters).`;
  return null;
}

export function validateDisplayName(value: string): string | null {
  const name = value.trim();
  if (name.length < 2) return "Your name needs at least 2 characters.";
  if (name.length > 40) return "Your name can have at most 40 characters.";
  return null;
}

export const BIO_MAX = 500;
export const CITY_MAX = 80;
export const MESSAGE_MAX = 2000;
export const MAX_INTERESTS = 10;
export const REPORT_DETAILS_MAX = 1000;

export function validateCity(value: string): string | null {
  const city = value.trim();
  if (!city) return "Tell people roughly where you are.";
  if (city.length > CITY_MAX) return `At most ${CITY_MAX} characters.`;
  return null;
}

export type AgeRange = { minAge: number; maxAge: number };

/** Keeps the range inside 18..99 and ordered (min <= max). `changed` says which bound moved last. */
export function clampAgeRange(range: AgeRange, changed: "min" | "max"): AgeRange {
  let { minAge, maxAge } = range;
  minAge = Math.min(99, Math.max(18, Math.round(minAge)));
  maxAge = Math.min(99, Math.max(18, Math.round(maxAge)));
  if (minAge > maxAge) {
    if (changed === "min") maxAge = minAge;
    else minAge = maxAge;
  }
  return { minAge, maxAge };
}

export function clampDistance(km: number): number {
  return Math.min(500, Math.max(1, Math.round(km)));
}
