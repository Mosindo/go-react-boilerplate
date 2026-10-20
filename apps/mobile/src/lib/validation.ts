export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const MAX_BIO_LENGTH = 500;
export const MAX_INTERESTS = 10;
export const MAX_MESSAGE_LENGTH = 2000;

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) return "Enter your email address.";
  if (email.length > 254 || !EMAIL_PATTERN.test(email)) return "Enter a valid email address.";
  return null;
}

export function validatePassword(value: string): string | null {
  if (!value) return "Enter a password.";
  if (value.length < 8) return "Use at least 8 characters.";
  if (value.length > 128) return "Use at most 128 characters.";
  return null;
}

export function validateFirstName(value: string): string | null {
  const name = value.trim();
  if (!name) return "Enter your first name.";
  if (name.length > 40) return "Your name is too long.";
  return null;
}

export function validateBio(value: string): string | null {
  return value.trim().length > MAX_BIO_LENGTH ? `Keep your bio under ${MAX_BIO_LENGTH} characters.` : null;
}

export function validateMessageBody(value: string): string | null {
  const body = value.trim();
  if (!body) return "Write something first.";
  if (body.length > MAX_MESSAGE_LENGTH) return `Messages are limited to ${MAX_MESSAGE_LENGTH} characters.`;
  return null;
}

export function validateAgeRange(min: number, max: number): string | null {
  if (min < MIN_AGE || max > MAX_AGE) return `Ages must be between ${MIN_AGE} and ${MAX_AGE}.`;
  if (min > max) return "The minimum age cannot exceed the maximum age.";
  return null;
}

export function toBirthDate(year: string, month: string, day: string): string | null {
  const y = Number(year);
  const m = Number(month);
  const d = Number(day);
  if (!Number.isInteger(y) || !Number.isInteger(m) || !Number.isInteger(d)) return null;
  if (year.trim().length !== 4) return null;
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) return null;
  return `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

export function ageFromBirthDate(birthDate: string, now: Date = new Date()): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(birthDate);
  if (!match) return null;
  const [y, m, d] = [Number(match[1]), Number(match[2]), Number(match[3])];
  let age = now.getUTCFullYear() - y;
  const hadBirthday = now.getUTCMonth() + 1 > m || (now.getUTCMonth() + 1 === m && now.getUTCDate() >= d);
  if (!hadBirthday) age -= 1;
  return age;
}

export function validateBirthDate(birthDate: string | null, now: Date = new Date()): string | null {
  if (!birthDate) return "Enter a valid date of birth.";
  const age = ageFromBirthDate(birthDate, now);
  if (age === null || age < 0) return "Enter a valid date of birth.";
  if (age < MIN_AGE) return `You must be at least ${MIN_AGE} to use Lumen.`;
  if (age > MAX_AGE) return "Enter a valid date of birth.";
  return null;
}
