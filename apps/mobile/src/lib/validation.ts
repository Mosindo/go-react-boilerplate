import { t } from "../i18n/fr";
import { isAdult, parseBirthDate } from "./dates";

/** Each validator returns a French error message or null when valid. */
export type FieldError = string | null;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function validateEmail(value: string): FieldError {
  const email = normalizeEmail(value);
  if (!email) return t.validation.required;
  if (email.length > 254 || !EMAIL_PATTERN.test(email)) return t.validation.emailInvalid;
  return null;
}

export function validatePassword(value: string): FieldError {
  if (!value) return t.validation.required;
  if (value.length < 8 || value.length > 72) return t.validation.passwordLength;
  if (!/[A-Za-z]/.test(value) || !/\d/.test(value)) return t.validation.passwordComposition;
  return null;
}

export function validatePasswordConfirmation(password: string, confirmation: string): FieldError {
  return password === confirmation ? null : t.validation.passwordMismatch;
}

export function validateRequired(value: string): FieldError {
  return value.trim() ? null : t.validation.required;
}

export function validateFirstName(value: string): FieldError {
  const trimmed = value.trim();
  if (!trimmed) return t.validation.required;
  return trimmed.length > 40 ? t.validation.nameTooLong : null;
}

export function validateCity(value: string): FieldError {
  const trimmed = value.trim();
  if (!trimmed) return t.validation.required;
  return trimmed.length > 80 ? t.validation.cityTooLong : null;
}

export function validateBio(value: string): FieldError {
  return value.trim().length > 500 ? t.validation.bioTooLong : null;
}

export function validateMessage(value: string): FieldError {
  const trimmed = value.trim();
  if (!trimmed) return t.validation.required;
  return trimmed.length > 2000 ? t.validation.messageTooLong : null;
}

/** Validates the three birth date inputs; the user must be at least 18 on `now`. */
export function validateBirthDate(
  day: string,
  month: string,
  year: string,
  now: Date = new Date()
): FieldError {
  const iso = parseBirthDate(day, month, year, now);
  if (!iso) return t.validation.birthInvalid;
  if (now.getFullYear() - Number(year) > 120) return t.validation.birthTooOld;
  if (!isAdult(iso, now)) return t.validation.birthUnderage;
  return null;
}
