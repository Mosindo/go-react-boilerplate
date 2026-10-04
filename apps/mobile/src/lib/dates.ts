export const MIN_AGE = 18;
export const MAX_AGE = 99;

export type BirthDateResult = { iso: string; age: number } | { error: string };

/** Auto-inserts slashes while typing a DD/MM/YYYY date. Keeps digits only. */
export function maskBirthInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) return digits;
  if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

/** Whole years between an ISO date (YYYY-MM-DD) and `now`. */
export function ageFromIso(iso: string, now: Date = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  const hadBirthday = now.getMonth() + 1 > m || (now.getMonth() + 1 === m && now.getDate() >= d);
  if (!hadBirthday) age -= 1;
  return age;
}

/** Parses and validates DD/MM/YYYY: real calendar date, not in the future, 18..99 years old. */
export function parseBirthDate(input: string, now: Date = new Date()): BirthDateResult {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) return { error: "Enter your birth date as DD/MM/YYYY." };
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return { error: "That date doesn't exist." };
  }
  const iso = `${match[3]}-${match[2]}-${match[1]}`;
  const age = ageFromIso(iso, now);
  if (age < 0) return { error: "Birth date can't be in the future." };
  if (age < MIN_AGE) return { error: `Lumen is for adults only. You must be ${MIN_AGE} or older.` };
  if (age > MAX_AGE) return { error: "Please check the year of your birth date." };
  return { iso, age };
}

/** YYYY-MM-DD -> DD/MM/YYYY (empty string for invalid input). */
export function isoToBirthInput(iso: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
  return match ? `${match[3]}/${match[2]}/${match[1]}` : "";
}
