export const MIN_AGE = 18;

export function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim());
}

export function passwordProblem(value: string): string | null {
  if (value.length < 8) return "8 caractères minimum.";
  if (value.length > 72) return "72 caractères maximum.";
  if (!/\p{L}/u.test(value) || !/\d/.test(value)) return "Ajoute au moins une lettre et un chiffre.";
  return null;
}

export function isValidFirstName(value: string): boolean {
  const trimmed = value.trim();
  return trimmed.length >= 1 && trimmed.length <= 40 && /^[\p{L}][\p{L}\s'’-]*$/u.test(trimmed);
}

export function ageOn(birth: Date, now: Date): number {
  let age = now.getFullYear() - birth.getFullYear();
  const beforeBirthday = now.getMonth() < birth.getMonth() || (now.getMonth() === birth.getMonth() && now.getDate() < birth.getDate());
  if (beforeBirthday) age -= 1;
  return age;
}

export type BirthdateCheck = { ok: true; iso: string; age: number } | { ok: false; reason: string };

/** Validates a DD / MM / YYYY birthdate and the 18+ rule. */
export function checkBirthdate(day: string, month: string, year: string, now: Date = new Date()): BirthdateCheck {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { ok: false, reason: "Saisis une date complète (JJ / MM / AAAA)." };
  }
  const date = new Date(y, m - 1, d);
  if (date.getFullYear() !== y || date.getMonth() !== m - 1 || date.getDate() !== d) {
    return { ok: false, reason: "Cette date n'existe pas." };
  }
  const age = ageOn(date, now);
  if (date > now || age > 120) {
    return { ok: false, reason: "Cette date de naissance n'est pas valide." };
  }
  if (age < MIN_AGE) {
    return { ok: false, reason: "Lueur est réservée aux personnes majeures." };
  }
  const iso = `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return { ok: true, iso, age };
}

export function roundCoordinate(value: number): number {
  return Math.round(value * 100) / 100;
}
