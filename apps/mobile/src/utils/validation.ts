import { strings } from "../i18n/strings";

export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const MAX_PHOTOS = 6;
export const MAX_INTERESTS = 10;
export const FIRST_NAME_MAX = 40;
export const BIO_MAX = 500;
export const CITY_MAX = 80;
export const PASSWORD_MIN = 8;
export const PASSWORD_MAX = 128;
export const MAX_DISTANCE_MIN_KM = 1;
export const MAX_DISTANCE_MAX_KM = 500;

/** Every validator returns an error message, or null when the value is valid. */
export type ValidationResult = string | null;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function validateEmail(value: string): ValidationResult {
  const email = normalizeEmail(value);
  if (!email) {
    return strings.validation.emailRequired;
  }
  return EMAIL_PATTERN.test(email) && email.length <= 254 ? null : strings.validation.emailInvalid;
}

export function validatePassword(value: string): ValidationResult {
  if (value.length < PASSWORD_MIN) {
    return strings.validation.passwordShort;
  }
  return value.length > PASSWORD_MAX ? strings.validation.passwordLong : null;
}

export function validateResetCode(value: string): ValidationResult {
  return value.trim().length === 0 ? strings.validation.codeRequired : null;
}

export function validateFirstName(value: string): ValidationResult {
  const name = value.trim();
  if (!name) {
    return strings.validation.nameRequired;
  }
  return name.length > FIRST_NAME_MAX ? strings.validation.nameLong : null;
}

export function validateBio(value: string): ValidationResult {
  return value.trim().length > BIO_MAX ? strings.validation.bioLong : null;
}

export function validateCity(value: string): ValidationResult {
  return value.trim().length > CITY_MAX ? strings.validation.cityLong : null;
}

export function validateAgeRange(minAge: number, maxAge: number): ValidationResult {
  if (!Number.isInteger(minAge) || !Number.isInteger(maxAge)) {
    return strings.validation.ageRangeInvalid;
  }
  if (minAge < MIN_AGE || maxAge > MAX_AGE || minAge > maxAge) {
    return strings.validation.ageRangeInvalid;
  }
  return null;
}

export function validateDistance(km: number): ValidationResult {
  return Number.isInteger(km) && km >= MAX_DISTANCE_MIN_KM && km <= MAX_DISTANCE_MAX_KM
    ? null
    : strings.validation.distanceInvalid;
}

function pad(value: number, size: number): string {
  return String(value).padStart(size, "0");
}

/**
 * Builds a YYYY-MM-DD string from separate day/month/year inputs.
 * Returns null when the parts do not form a real calendar date (e.g. 31/02/2000).
 */
export function buildBirthDate(day: string, month: string, year: string): string | null {
  if (!/^\d{1,2}$/.test(day.trim()) || !/^\d{1,2}$/.test(month.trim()) || !/^\d{4}$/.test(year.trim())) {
    return null;
  }
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return null;
  }
  return `${pad(y, 4)}-${pad(m, 2)}-${pad(d, 2)}`;
}

export function splitBirthDate(birthDate: string): { day: string; month: string; year: string } {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(birthDate);
  return match ? { year: match[1], month: match[2], day: match[3] } : { day: "", month: "", year: "" };
}

/** Whole years between an ISO birth date and `now` (local calendar date). Null if unparsable. */
export function calculateAge(birthDate: string, now: Date = new Date()): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(birthDate);
  if (!match) {
    return null;
  }
  const y = Number(match[1]);
  const m = Number(match[2]);
  const d = Number(match[3]);
  let age = now.getFullYear() - y;
  const hadBirthday = now.getMonth() + 1 > m || (now.getMonth() + 1 === m && now.getDate() >= d);
  if (!hadBirthday) {
    age -= 1;
  }
  return age;
}

export function validateBirthDate(birthDate: string | null, now: Date = new Date()): ValidationResult {
  if (!birthDate) {
    return strings.validation.birthDateInvalid;
  }
  const age = calculateAge(birthDate, now);
  if (age === null || age < 0 || age > 120) {
    return strings.validation.birthDateInvalid;
  }
  return age < MIN_AGE ? strings.validation.underage : null;
}

export function firstError(...results: ValidationResult[]): ValidationResult {
  return results.find((result) => result !== null) ?? null;
}
