export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const MAX_BIO = 500;
export const MAX_PHOTOS = 6;
export const MAX_INTERESTS = 10;

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidEmail(value: string): boolean {
  const email = normalizeEmail(value);
  return email.length <= 254 && EMAIL_PATTERN.test(email);
}

/** Returns a user-facing message when the password is unacceptable, otherwise null. */
export function passwordIssue(value: string): string | null {
  if (value.length < 8) {
    return "Le mot de passe doit contenir au moins 8 caractères.";
  }
  if (new TextEncoder().encode(value).length > 72) {
    return "Le mot de passe est trop long (72 caractères maximum).";
  }
  return null;
}

export function ageFromBirthdate(isoDate: string, now: Date = new Date()): number {
  const [year, month, day] = isoDate.split("-").map(Number);
  let age = now.getFullYear() - year;
  const beforeBirthday = now.getMonth() + 1 < month || (now.getMonth() + 1 === month && now.getDate() < day);
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

export type BirthdateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

/** Validates day/month/year text fields and applies the 18+ rule. */
export function parseBirthdate(day: string, month: string, year: string, now: Date = new Date()): BirthdateResult {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { ok: false, error: "Entrez une date de naissance valide (JJ / MM / AAAA)." };
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const iso = `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  const age = ageFromBirthdate(iso, now);
  if (age < MIN_AGE) {
    return { ok: false, error: "Vous devez avoir au moins 18 ans pour utiliser l'application." };
  }
  if (age > MAX_AGE) {
    return { ok: false, error: "Entrez une date de naissance valide." };
  }
  return { ok: true, iso, age };
}

/** Coordinates are rounded client-side too: the exact position never leaves the device. */
export function roundCoordinate(value: number): number {
  return Math.round(value * 100) / 100;
}
