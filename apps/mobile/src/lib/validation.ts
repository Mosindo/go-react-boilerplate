export const MIN_AGE = 18;
export const MAX_AGE = 99;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) return "Saisissez votre adresse e-mail.";
  if (email.length > 254 || !EMAIL_RE.test(email)) return "Cette adresse e-mail n'est pas valide.";
  return null;
}

export function validatePassword(value: string): string | null {
  if (value.length < 8) return "Le mot de passe doit contenir au moins 8 caractères.";
  if (new TextEncoder().encode(value).length > 72) return "Le mot de passe est trop long (72 octets maximum).";
  return null;
}

export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

export function ageOn(year: number, month: number, day: number, now: Date = new Date()): number {
  let age = now.getFullYear() - year;
  const m = now.getMonth() + 1;
  if (m < month || (m === month && now.getDate() < day)) age -= 1;
  return age;
}

/** Validates a day / month / year triple typed by the user and enforces the 18+ rule. */
export function parseBirthDate(dayRaw: string, monthRaw: string, yearRaw: string, now: Date = new Date()): BirthDateResult {
  const day = Number(dayRaw);
  const month = Number(monthRaw);
  const year = Number(yearRaw);
  if (![dayRaw, monthRaw, yearRaw].every((v) => /^\d+$/.test(v.trim())) || yearRaw.trim().length !== 4) {
    return { ok: false, error: "Saisissez votre date de naissance (JJ / MM / AAAA)." };
  }
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const age = ageOn(year, month, day, now);
  if (age < MIN_AGE) return { ok: false, error: "Vous devez avoir 18 ans ou plus pour utiliser Alma." };
  if (age > MAX_AGE) return { ok: false, error: "Cette date de naissance n'est pas valide." };
  const iso = `${String(year).padStart(4, "0")}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
  return { ok: true, iso, age };
}

/** Clamp an age range so min <= max and both stay within 18-99. */
export function clampAgeRange(min: number, max: number, changed: "min" | "max"): [number, number] {
  let lo = Math.min(Math.max(min, MIN_AGE), MAX_AGE);
  let hi = Math.min(Math.max(max, MIN_AGE), MAX_AGE);
  if (lo > hi) {
    if (changed === "min") hi = lo;
    else lo = hi;
  }
  return [lo, hi];
}
