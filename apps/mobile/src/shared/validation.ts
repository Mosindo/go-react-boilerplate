import {
  APP_NAME,
  MAX_AGE,
  MAX_BIO_LENGTH,
  MAX_FIRST_NAME_LENGTH,
  MAX_PASSWORD_BYTES,
  MIN_AGE,
  MIN_PASSWORD_LENGTH
} from "../config";

export type FieldResult = string | null;

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): FieldResult {
  const email = normalizeEmail(value);
  if (!email) return "Saisissez votre email.";
  return EMAIL_PATTERN.test(email) ? null : "Cet email ne semble pas valide.";
}

/** The API enforces bcrypt's 72-byte limit, so length is measured in UTF-8 bytes. */
export function utf8ByteLength(value: string): number {
  let bytes = 0;
  for (const char of value) {
    const code = char.codePointAt(0) ?? 0;
    bytes += code < 0x80 ? 1 : code < 0x800 ? 2 : code < 0x10000 ? 3 : 4;
  }
  return bytes;
}

export function validatePassword(value: string): FieldResult {
  if (value.length < MIN_PASSWORD_LENGTH) return `Au moins ${MIN_PASSWORD_LENGTH} caractères.`;
  if (utf8ByteLength(value) > MAX_PASSWORD_BYTES)
    return `${MAX_PASSWORD_BYTES} octets maximum (évitez les très longs mots de passe).`;
  return null;
}

export function validatePasswordConfirmation(password: string, confirmation: string): FieldResult {
  return password === confirmation ? null : "Les mots de passe ne correspondent pas.";
}

export function validateResetCode(value: string): FieldResult {
  return value.trim().length >= 4 ? null : "Saisissez le code reçu par email.";
}

export function validateFirstName(value: string): FieldResult {
  const name = value.trim();
  if (!name) return "Saisissez votre prénom.";
  return name.length > MAX_FIRST_NAME_LENGTH ? `${MAX_FIRST_NAME_LENGTH} caractères maximum.` : null;
}

export function validateBio(value: string): FieldResult {
  return value.length > MAX_BIO_LENGTH ? `${MAX_BIO_LENGTH} caractères maximum.` : null;
}

export function calculateAge(birthYear: number, birthMonth: number, birthDay: number, today: Date): number {
  let age = today.getFullYear() - birthYear;
  const hadBirthday =
    today.getMonth() + 1 > birthMonth || (today.getMonth() + 1 === birthMonth && today.getDate() >= birthDay);
  if (!hadBirthday) {
    age -= 1;
  }
  return age;
}

export type BirthDateResult = { ok: true; isoDate: string; age: number } | { ok: false; error: string };

/** Validates a day/month/year triple typed by the user; the person must be 18 to 99 years old. */
export function parseBirthDate(day: string, month: string, year: string, today: Date = new Date()): BirthDateResult {
  const dayNumber = Number(day);
  const monthNumber = Number(month);
  const yearNumber = Number(year);
  const filled = day.trim() !== "" && month.trim() !== "" && year.trim().length === 4;
  if (!filled || ![dayNumber, monthNumber, yearNumber].every(Number.isInteger)) {
    return { ok: false, error: "Saisissez une date de naissance complète (JJ / MM / AAAA)." };
  }

  const candidate = new Date(Date.UTC(yearNumber, monthNumber - 1, dayNumber));
  const exists =
    candidate.getUTCFullYear() === yearNumber &&
    candidate.getUTCMonth() === monthNumber - 1 &&
    candidate.getUTCDate() === dayNumber;
  if (!exists) {
    return { ok: false, error: "Cette date n'existe pas." };
  }

  const age = calculateAge(yearNumber, monthNumber, dayNumber, today);
  if (age < 0) {
    return { ok: false, error: "Cette date est dans le futur." };
  }
  if (age < MIN_AGE) {
    return { ok: false, error: `${APP_NAME} est réservé aux personnes de ${MIN_AGE} ans et plus.` };
  }
  if (age > MAX_AGE) {
    return { ok: false, error: `L'âge maximum est de ${MAX_AGE} ans.` };
  }

  const pad = (value: number, size: number) => String(value).padStart(size, "0");
  return { ok: true, isoDate: `${pad(yearNumber, 4)}-${pad(monthNumber, 2)}-${pad(dayNumber, 2)}`, age };
}

/** Splits an ISO date (YYYY-MM-DD) into the three text inputs of the date field. */
export function splitIsoDate(isoDate: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = isoDate.split("-");
  return { day, month, year };
}
