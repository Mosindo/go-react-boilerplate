export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim());
}

/** Mirrors the server policy: 8+ characters with a letter and a digit. */
export function passwordProblem(value: string): string | null {
  if (value.length < 8) return "8 caractères minimum.";
  if (!/\p{L}/u.test(value)) return "Ajoutez au moins une lettre.";
  if (!/\d/.test(value)) return "Ajoutez au moins un chiffre.";
  return null;
}

export function isValidFirstName(value: string): boolean {
  const trimmed = value.trim();
  return trimmed.length > 0 && trimmed.length <= 40 && /^[\p{L} .'’-]+$/u.test(trimmed) && /\p{L}/u.test(trimmed);
}

export function ageOn(birthdate: Date, now: Date = new Date()): number {
  let age = now.getFullYear() - birthdate.getFullYear();
  const beforeBirthday =
    now.getMonth() < birthdate.getMonth() || (now.getMonth() === birthdate.getMonth() && now.getDate() < birthdate.getDate());
  if (beforeBirthday) age -= 1;
  return age;
}

export type BirthdateCheck = { ok: true; iso: string; age: number } | { ok: false; error: string };

/** Validates a day/month/year triple typed by the user. */
export function checkBirthdate(day: string, month: string, year: string, now: Date = new Date()): BirthdateCheck {
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
  const age = ageOn(date, now);
  if (age < MIN_AGE) return { ok: false, error: "Lueur est réservée aux personnes de 18 ans et plus." };
  if (age > 110) return { ok: false, error: "Vérifiez l'année de naissance." };
  const iso = `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return { ok: true, iso, age };
}
