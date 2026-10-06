export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function isValidEmail(value: string): boolean {
  const email = value.trim();
  return email.length <= 254 && /^[^\s@<>,;"]+@[^\s@<>,;"]+\.[^\s@<>,;"]+$/.test(email) && !email.endsWith(".");
}

/** bcrypt ignores everything past 72 bytes, so the server refuses longer passwords. */
export function passwordProblem(value: string): string | null {
  if (value.length < 8) {
    return "Au moins 8 caractères.";
  }
  if (new TextEncoder().encode(value).length > 72) {
    return "72 caractères maximum.";
  }
  return null;
}

/** Inserts the slashes while the user types: "14031994" -> "14/03/1994". */
export function formatBirthdateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  const parts = [digits.slice(0, 2), digits.slice(2, 4), digits.slice(4, 8)].filter((part) => part.length > 0);
  return parts.join("/");
}

export type BirthdateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

/** Parses "JJ/MM/AAAA", checks the calendar date and the 18+ rule. */
export function parseBirthdate(input: string, now: Date = new Date()): BirthdateResult {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) {
    return { ok: false, error: "Format attendu : JJ/MM/AAAA." };
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const age = ageOn(year, month, day, now);
  if (age < 0 || age > MAX_AGE) {
    return { ok: false, error: "Date de naissance invalide." };
  }
  if (age < MIN_AGE) {
    return { ok: false, error: "Vous devez avoir 18 ans ou plus." };
  }
  const iso = `${match[3]}-${match[2]}-${match[1]}`;
  return { ok: true, iso, age };
}

export function ageOn(year: number, month: number, day: number, now: Date): number {
  let age = now.getFullYear() - year;
  const beforeBirthday = now.getMonth() + 1 < month || (now.getMonth() + 1 === month && now.getDate() < day);
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

/** "1994-03-14" -> "14/03/1994" for editing. */
export function isoToBirthdateInput(iso: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  return match ? `${match[3]}/${match[2]}/${match[1]}` : "";
}

export function validFirstName(value: string): string | null {
  const name = value.trim().replace(/\s+/g, " ");
  if (name.length === 0) {
    return "Indiquez votre prénom.";
  }
  if (name.length > 40) {
    return "40 caractères maximum.";
  }
  if (!/^[\p{L}][\p{L} '’.-]*$/u.test(name)) {
    return "Lettres, espaces, tirets et apostrophes uniquement.";
  }
  return null;
}

export function clampAgeRange(min: number, max: number): { min: number; max: number } {
  const lo = Math.min(Math.max(min, MIN_AGE), MAX_AGE);
  const hi = Math.min(Math.max(max, MIN_AGE), MAX_AGE);
  return lo <= hi ? { min: lo, max: hi } : { min: hi, max: lo };
}
