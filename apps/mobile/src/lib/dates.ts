export const MIN_AGE = 18;
export const MAX_AGE = 120;

export type BirthDateResult =
  | { ok: true; iso: string; age: number }
  | { ok: false; error: string };

/** Keeps digits only (max 8) and inserts the slashes of a DD/MM/YYYY mask. */
export function maskBirthDateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

export function isValidCalendarDate(year: number, month: number, day: number): boolean {
  if (!Number.isInteger(year) || !Number.isInteger(month) || !Number.isInteger(day)) {
    return false;
  }
  if (month < 1 || month > 12 || day < 1 || year < 1) {
    return false;
  }
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
}

export function computeAge(year: number, month: number, day: number, today: Date = new Date()): number {
  let age = today.getFullYear() - year;
  const beforeBirthday =
    today.getMonth() + 1 < month || (today.getMonth() + 1 === month && today.getDate() < day);
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

function pad(value: number, length: number): string {
  return String(value).padStart(length, "0");
}

export function toIsoDate(year: number, month: number, day: number): string {
  return `${pad(year, 4)}-${pad(month, 2)}-${pad(day, 2)}`;
}

/** Validates a "DD/MM/YYYY" string. The server enforces the same rules; this is for instant feedback. */
export function validateBirthDate(masked: string, today: Date = new Date()): BirthDateResult {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(masked.trim());
  if (!match) {
    return { ok: false, error: "Enter your birth date as DD/MM/YYYY." };
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  if (!isValidCalendarDate(year, month, day)) {
    return { ok: false, error: "That date does not exist. Check the day, month and year." };
  }
  const birth = new Date(year, month - 1, day);
  if (birth.getTime() > today.getTime()) {
    return { ok: false, error: "Your birth date cannot be in the future." };
  }
  const age = computeAge(year, month, day, today);
  if (age > MAX_AGE) {
    return { ok: false, error: "Please enter a valid birth date." };
  }
  if (age < MIN_AGE) {
    return { ok: false, error: `You must be at least ${MIN_AGE} to use Alba.` };
  }
  return { ok: true, iso: toIsoDate(year, month, day), age };
}
