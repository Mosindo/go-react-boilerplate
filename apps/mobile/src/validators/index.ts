import { fr } from '../i18n/fr';

export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

/** Returns an error message or null. */
export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) return fr.validation.emailRequired;
  if (email.length > 254 || !EMAIL_RE.test(email)) return fr.validation.emailInvalid;
  return null;
}

export function validatePassword(value: string): string | null {
  if (!value) return fr.validation.passwordRequired;
  if (value.length < 8) return fr.validation.passwordShort;
  if (value.length > 128) return fr.validation.passwordLong;
  return null;
}

export function validateFirstName(value: string): string | null {
  const name = value.trim();
  if (!name) return fr.validation.firstNameRequired;
  if (name.length > 40 || !/^[\p{L}][\p{L}\p{M} '’-]*$/u.test(name))
    return fr.validation.firstNameInvalid;
  return null;
}

export function validateBio(value: string): string | null {
  return value.length > 500 ? fr.validation.bioTooLong : null;
}

export function validateResetCode(value: string): string | null {
  const code = value.trim();
  if (!code) return fr.validation.codeRequired;
  if (code.length !== 8) return fr.validation.codeInvalid;
  return null;
}

export function validateMessageBody(value: string): string | null {
  const body = value.trim();
  if (!body) return fr.validation.messageEmpty;
  if (body.length > 2000) return fr.validation.messageTooLong;
  return null;
}

export function validateAgeRange(min: number, max: number): string | null {
  if (min < MIN_AGE || max > MAX_AGE || min > max) return fr.validation.ageRange;
  return null;
}

export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

/** Full years between `birth` (y/m/d) and `today`. */
export function computeAge(year: number, month: number, day: number, today: Date): number {
  let age = today.getFullYear() - year;
  const m = today.getMonth() + 1;
  if (m < month || (m === month && today.getDate() < day)) age -= 1;
  return age;
}

/** Validates a day/month/year entry (strings from text inputs). */
export function parseBirthDate(
  dayStr: string,
  monthStr: string,
  yearStr: string,
  today: Date = new Date(),
): BirthDateResult {
  const d = dayStr.trim();
  const m = monthStr.trim();
  const y = yearStr.trim();
  if (!d || !m || y.length !== 4 || !/^\d+$/.test(d + m + y)) {
    return { ok: false, error: fr.validation.birthDateIncomplete };
  }
  const day = Number(d);
  const month = Number(m);
  const year = Number(y);
  if (month < 1 || month > 12 || day < 1 || day > daysInMonth(year, month) || year < 1900) {
    return { ok: false, error: fr.validation.birthDateInvalid };
  }
  const candidate = new Date(year, month - 1, day);
  if (candidate.getTime() > today.getTime()) {
    return { ok: false, error: fr.validation.birthDateInvalid };
  }
  const age = computeAge(year, month, day, today);
  if (age < MIN_AGE) return { ok: false, error: fr.validation.underage };
  if (age > MAX_AGE) return { ok: false, error: fr.validation.tooOld };
  const iso = `${y}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
  return { ok: true, iso, age };
}

/** Splits an ISO `YYYY-MM-DD` date into text-input parts. */
export function splitIsoDate(iso: string): { day: string; month: string; year: string } {
  const [year = '', month = '', day = ''] = iso.slice(0, 10).split('-');
  return { day, month, year };
}
