// Client-side checks mirror the server rules to give instant feedback.
// The server remains the source of truth and re-validates everything.

export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function normalizeEmail(email: string): string {
  return email.trim().toLowerCase();
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizeEmail(email));
}

/** A character is treated as a letter when it has distinct upper/lower case forms (Latin, Greek, Cyrillic…). */
function isCasedLetter(char: string): boolean {
  return char.toLowerCase() !== char.toUpperCase();
}

export function passwordProblem(password: string): string | null {
  if (password.length < 8) return "8 caractères minimum.";
  if (![...password].some(isCasedLetter) || !/\d/.test(password)) return "Ajoutez au moins une lettre et un chiffre.";
  return null;
}

export function ageOn(birthdate: Date, today = new Date()): number {
  let age = today.getFullYear() - birthdate.getFullYear();
  const m = today.getMonth() - birthdate.getMonth();
  if (m < 0 || (m === 0 && today.getDate() < birthdate.getDate())) age--;
  return age;
}

export type BirthdateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

/** Validates a day/month/year triple typed by the user and returns an ISO date (YYYY-MM-DD). */
export function parseBirthdate(day: string, month: string, year: string, today = new Date()): BirthdateResult {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { ok: false, error: "Saisissez une date complète (JJ / MM / AAAA)." };
  }
  const date = new Date(y, m - 1, d);
  if (date.getFullYear() !== y || date.getMonth() !== m - 1 || date.getDate() !== d) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const age = ageOn(date, today);
  if (age < MIN_AGE) return { ok: false, error: "Lueur est réservé aux personnes majeures (18 ans et plus)." };
  if (age > MAX_AGE) return { ok: false, error: "Vérifiez l'année de naissance." };
  const iso = `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return { ok: true, iso, age };
}

export function firstNameProblem(name: string): string | null {
  const trimmed = name.trim().replace(/\s+/g, " ");
  if (trimmed.length === 0) return "Indiquez votre prénom.";
  if (trimmed.length > 40) return "40 caractères maximum.";
  // Light client check; the server applies the full Unicode letter rule.
  if (/[\d<>{}[\]()@#$%^&*_=+|\\/~`!?.,;:"]/.test(trimmed)) return "Lettres, espaces, tirets et apostrophes uniquement.";
  return null;
}
