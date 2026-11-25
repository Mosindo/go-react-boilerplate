const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  return emailPattern.test(normalizeEmail(value)) ? null : "Adresse e-mail invalide.";
}

export function validatePassword(value: string): string | null {
  // The API (bcrypt) accepts at most 72 bytes.
  const bytes = new TextEncoder().encode(value).length;
  if (value.length < 8) {
    return "8 caractères minimum.";
  }
  if (bytes > 72) {
    return "72 caractères maximum.";
  }
  if (!/\p{L}/u.test(value) || !/\d/.test(value)) {
    return "Ajoutez au moins une lettre et un chiffre.";
  }
  return null;
}

export const MIN_AGE = 18;

export type ParsedBirthdate = { iso: string; age: number };

/**
 * Parses a French-formatted birthdate (JJ/MM/AAAA) and checks the age rule.
 * Returns an error message or the ISO date and age.
 */
export function parseBirthdate(input: string, now: Date = new Date()): ParsedBirthdate | string {
  const match = /^(\d{1,2})[/.\- ](\d{1,2})[/.\- ](\d{4})$/.exec(input.trim());
  if (!match) {
    return "Format attendu : JJ/MM/AAAA.";
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return "Cette date n'existe pas.";
  }
  const age = ageFrom(year, month, day, now);
  if (age < MIN_AGE) {
    return "Lueur est réservée aux personnes majeures (18 ans et plus).";
  }
  if (age > 120) {
    return "Date de naissance invalide.";
  }
  const iso = `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
  return { iso, age };
}

export function ageFrom(year: number, month: number, day: number, now: Date = new Date()): number {
  let age = now.getFullYear() - year;
  const m = now.getMonth() + 1;
  if (m < month || (m === month && now.getDate() < day)) {
    age -= 1;
  }
  return age;
}

/** Converts an ISO date (AAAA-MM-JJ) to the JJ/MM/AAAA input format. */
export function isoToFrench(iso: string): string {
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : "";
}

/** Auto-inserts slashes while typing a birthdate. */
export function maskBirthdate(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}
