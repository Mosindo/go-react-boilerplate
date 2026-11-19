export const MIN_AGE = 18;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(normalizeEmail(value));
}

/** Mirrors the server rule: 8-72 characters, at least one letter and one digit. */
export function passwordProblem(value: string): string | null {
  if (value.length < 8) return "8 caractères minimum.";
  if (value.length > 72) return "72 caractères maximum.";
  if (!/[0-9]/.test(value) || !/[^0-9\s]/.test(value)) return "Utilisez au moins une lettre et un chiffre.";
  return null;
}

export function ageFrom(birthdate: Date, now: Date = new Date()): number {
  let age = now.getFullYear() - birthdate.getFullYear();
  const m = now.getMonth() - birthdate.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < birthdate.getDate())) age--;
  return age;
}

/**
 * Parses a French-formatted date (JJ/MM/AAAA) into an ISO date (AAAA-MM-JJ).
 * Returns null when the date does not exist.
 */
export function parseFrenchDate(value: string): string | null {
  const m = /^(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})$/.exec(value.trim());
  if (!m) return null;
  const [day, month, year] = [Number(m[1]), Number(m[2]), Number(m[3])];
  const d = new Date(Date.UTC(year, month - 1, day));
  if (d.getUTCFullYear() !== year || d.getUTCMonth() !== month - 1 || d.getUTCDate() !== day) return null;
  return `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}

export function birthdateProblem(value: string, now: Date = new Date()): string | null {
  const iso = parseFrenchDate(value);
  if (!iso) return "Format attendu : JJ/MM/AAAA.";
  const [y, mo, d] = iso.split("-").map(Number);
  const age = ageFrom(new Date(y, mo - 1, d), now);
  if (age < MIN_AGE) return "Lueur est réservé aux personnes majeures (18 ans et plus).";
  if (age > 99) return "Vérifiez votre date de naissance.";
  return null;
}

export function isoToFrenchDate(iso: string): string {
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
}

/** Inserts slashes while typing a date: "0103" -> "01/03". */
export function maskDateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) return digits;
  if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

/** True when the string contains at least one letter (any script). Hermes-safe (no \p{} escapes). */
export function hasLetter(value: string): boolean {
  return Array.from(value).some((ch) => ch.toLowerCase() !== ch.toUpperCase() || ch.charCodeAt(0) > 0x2e7f);
}
