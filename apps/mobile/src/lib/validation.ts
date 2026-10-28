export const MIN_AGE = 18;
export const MAX_BIO = 500;
export const MAX_FIRST_NAME = 40;
export const MAX_INTERESTS = 10;
export const MAX_PHOTOS = 6;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;
const DATE_RE = /^(\d{4})-(\d{2})-(\d{2})$/;

export function validateEmail(value: string): string | null {
  const email = value.trim();
  if (!email) return "Enter your email address.";
  if (email.length > 254 || !EMAIL_RE.test(email)) return "Enter a valid email address.";
  return null;
}

export function validatePassword(value: string): string | null {
  if (value.length < 8) return "Use at least 8 characters.";
  if (value.length > 128) return "Use at most 128 characters.";
  return null;
}

export function validateFirstName(value: string): string | null {
  const name = value.trim();
  if (!name) return "Enter your first name.";
  if (name.length > MAX_FIRST_NAME) return `At most ${MAX_FIRST_NAME} characters.`;
  return null;
}

export function validateBio(value: string): string | null {
  return value.length > MAX_BIO ? `At most ${MAX_BIO} characters.` : null;
}

export function parseBirthDate(value: string): Date | null {
  const match = DATE_RE.exec(value.trim());
  if (!match) return null;
  const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])];
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return null;
  }
  return date;
}

export function computeAge(birthDate: Date, now: Date = new Date()): number {
  let age = now.getUTCFullYear() - birthDate.getUTCFullYear();
  const beforeBirthday =
    now.getUTCMonth() < birthDate.getUTCMonth() ||
    (now.getUTCMonth() === birthDate.getUTCMonth() && now.getUTCDate() < birthDate.getUTCDate());
  if (beforeBirthday) age -= 1;
  return age;
}

export function validateBirthDate(value: string, now: Date = new Date()): string | null {
  const date = parseBirthDate(value);
  if (!date) return "Use the format YYYY-MM-DD.";
  const age = computeAge(date, now);
  if (age < MIN_AGE) return `You must be at least ${MIN_AGE} years old.`;
  if (age > 120) return "Enter a valid birth date.";
  return null;
}

export function validateAgeRange(min: number, max: number): string | null {
  if (min < MIN_AGE || max > 99) return `Ages must be between ${MIN_AGE} and 99.`;
  if (min > max) return "Minimum age cannot exceed maximum age.";
  return null;
}
