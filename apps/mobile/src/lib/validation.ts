// Pure helpers shared by forms. No React Native imports: unit tested with node:test.

export const MIN_AGE = 18;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidEmail(value: string): boolean {
  const email = normalizeEmail(value);
  return email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

/** Mirrors the API policy: 8+ characters with at least one letter and one digit. */
export function passwordProblem(password: string): string | null {
  if (password.length < 8) {
    return "8 caractères minimum.";
  }
  if (password.length > 128) {
    return "128 caractères maximum.";
  }
  if (!/\p{L}/u.test(password) || !/\p{N}/u.test(password)) {
    return "Au moins une lettre et un chiffre.";
  }
  return null;
}

export function ageOn(birthdate: Date, today: Date): number {
  let age = today.getFullYear() - birthdate.getFullYear();
  const beforeBirthday = today.getMonth() < birthdate.getMonth() || (today.getMonth() === birthdate.getMonth() && today.getDate() < birthdate.getDate());
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

export type BirthdateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

/** Validates a day/month/year triple typed by the user. */
export function parseBirthdate(day: string, month: string, year: string, today: Date = new Date()): BirthdateResult {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return { ok: false, error: "Indique ta date au format JJ / MM / AAAA." };
  }
  const date = new Date(y, m - 1, d);
  if (date.getFullYear() !== y || date.getMonth() !== m - 1 || date.getDate() !== d) {
    return { ok: false, error: "Cette date n’existe pas." };
  }
  const age = ageOn(date, today);
  if (age < MIN_AGE) {
    return { ok: false, error: "Lueur est réservée aux personnes majeures (18 ans et plus)." };
  }
  if (age > 120) {
    return { ok: false, error: "Vérifie l’année de naissance." };
  }
  const iso = `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return { ok: true, iso, age };
}

export function splitIsoDate(iso: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day, month, year };
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Toggles a value in a list, optionally capping the selection size. */
export function toggle<T>(list: readonly T[], value: T, max = Infinity): T[] {
  if (list.includes(value)) {
    return list.filter((item) => item !== value);
  }
  if (list.length >= max) {
    return [...list];
  }
  return [...list, value];
}
