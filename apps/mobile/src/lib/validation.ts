// Pure helpers (no imports) so they can be unit-tested with `node --test`.

export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidEmail(value: string): boolean {
  const email = normalizeEmail(value);
  return email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email);
}

/** 8-72 characters with at least one letter and one digit (mirrors the API). */
export function passwordProblem(value: string): string | null {
  if (value.length < 8) return "8 caractères minimum.";
  if (value.length > 72) return "72 caractères maximum.";
  if (!/[A-Za-zÀ-ɏ]/.test(value) || !/[0-9]/.test(value)) {
    return "Mélangez lettres et chiffres.";
  }
  return null;
}

export type ParsedDate = { iso: string; age: number } | null;

/** Parses JJ/MM/AAAA (also accepts AAAA-MM-JJ) and computes the age at `now`. */
export function parseBirthDate(input: string, now: Date = new Date()): ParsedDate {
  const text = input.trim();
  let day: number;
  let month: number;
  let year: number;
  let match = /^(\d{1,2})[/.\-\s](\d{1,2})[/.\-\s](\d{4})$/.exec(text);
  if (match) {
    day = Number(match[1]);
    month = Number(match[2]);
    year = Number(match[3]);
  } else {
    match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(text);
    if (!match) return null;
    year = Number(match[1]);
    month = Number(match[2]);
    day = Number(match[3]);
  }
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return null;
  }
  let age = now.getUTCFullYear() - year;
  const beforeBirthday = now.getUTCMonth() < month - 1 || (now.getUTCMonth() === month - 1 && now.getUTCDate() < day);
  if (beforeBirthday) age -= 1;
  const iso = `${String(year).padStart(4, "0")}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
  return { iso, age };
}

export function birthDateProblem(input: string, now: Date = new Date()): string | null {
  const parsed = parseBirthDate(input, now);
  if (!parsed) return "Format attendu : JJ/MM/AAAA.";
  if (parsed.age < MIN_AGE) return "Tandem est réservé aux personnes de 18 ans et plus.";
  if (parsed.age > MAX_AGE) return "Date de naissance invalide.";
  return null;
}

/** Converts an ISO date (AAAA-MM-JJ) to the JJ/MM/AAAA form used in inputs. */
export function isoToDisplayDate(iso: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  return m ? `${m[3]}/${m[2]}/${m[1]}` : iso;
}

export function firstNameProblem(value: string): string | null {
  const name = value.trim();
  if (name.length === 0) return "Indiquez votre prénom.";
  if ([...name].length > 40) return "40 caractères maximum.";
  return null;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
