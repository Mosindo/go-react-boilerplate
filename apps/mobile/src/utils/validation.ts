export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function isValidEmail(value: string): boolean {
  const email = value.trim();
  return email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email);
}

/** Mirrors the API rule (8-72 characters; bcrypt ignores anything past 72 bytes). */
export function passwordProblem(value: string): string | null {
  if (value.length < 8) {
    return "Au moins 8 caractères.";
  }
  if (value.length > 72) {
    return "72 caractères maximum.";
  }
  return null;
}

/** Turns "12031990" or "12/03/1990" typing into "12/03/1990" as the user types. */
export function formatBirthDateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  const parts = [digits.slice(0, 2), digits.slice(2, 4), digits.slice(4, 8)].filter(Boolean);
  return parts.join("/");
}

export function ageOn(birth: Date, now: Date): number {
  let age = now.getFullYear() - birth.getFullYear();
  const beforeBirthday = now.getMonth() < birth.getMonth() || (now.getMonth() === birth.getMonth() && now.getDate() < birth.getDate());
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

/** Parses JJ/MM/AAAA, rejects impossible dates and enforces the 18+ rule. */
export function parseBirthDate(input: string, now: Date = new Date()): BirthDateResult {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) {
    return { ok: false, error: "Format attendu : JJ/MM/AAAA." };
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(year, month - 1, day);
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const age = ageOn(date, now);
  if (age < MIN_AGE) {
    return { ok: false, error: "Vous devez avoir au moins 18 ans pour utiliser Lumen." };
  }
  if (age > MAX_AGE) {
    return { ok: false, error: "Date de naissance invalide." };
  }
  const iso = `${match[3]}-${match[2]}-${match[1]}`;
  return { ok: true, iso, age };
}

/** API birth date (YYYY-MM-DD) -> display form. */
export function isoToDisplayDate(iso: string): string {
  const [year, month, day] = iso.split("-");
  return year && month && day ? `${day}/${month}/${year}` : iso;
}
