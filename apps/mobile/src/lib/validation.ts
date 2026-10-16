// Client-side validation mirrors the server rules for fast feedback.
// The server stays the source of truth: nothing here is a security boundary.

export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const BIO_MAX = 500;
export const NAME_MAX = 40;
export const MESSAGE_MAX = 2000;
export const PASSWORD_MIN = 8;
export const PASSWORD_MAX = 72;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) {
    return "Saisissez votre adresse e-mail.";
  }
  if (email.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return "Cette adresse e-mail n'est pas valide.";
  }
  return null;
}

// bcrypt only looks at the first 72 bytes, so the limit is on bytes, not characters.
export function utf8Length(value: string): number {
  let bytes = 0;
  for (const char of value) {
    const code = char.codePointAt(0) ?? 0;
    bytes += code < 0x80 ? 1 : code < 0x800 ? 2 : code < 0x10000 ? 3 : 4;
  }
  return bytes;
}

export function validatePassword(value: string): string | null {
  if (utf8Length(value) < PASSWORD_MIN) {
    return `Le mot de passe doit contenir au moins ${PASSWORD_MIN} caractères.`;
  }
  if (utf8Length(value) > PASSWORD_MAX) {
    return `Le mot de passe est trop long (${PASSWORD_MAX} octets maximum).`;
  }
  return null;
}

export type DateParts = { day: string; month: string; year: string };

/** Builds an ISO date (YYYY-MM-DD) from the three inputs, or null if it is not a real calendar date. */
export function toIsoDate({ day, month, year }: DateParts): string | null {
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return null;
  }
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return null;
  }
  return `${year}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

export function ageFromIso(iso: string, today: Date = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = today.getUTCFullYear() - y;
  const monthDiff = today.getUTCMonth() + 1 - m;
  if (monthDiff < 0 || (monthDiff === 0 && today.getUTCDate() < d)) {
    age -= 1;
  }
  return age;
}

export function validateBirthDate(parts: DateParts, today: Date = new Date()): string | null {
  const iso = toIsoDate(parts);
  if (!iso) {
    return "Saisissez une date de naissance valide (JJ / MM / AAAA).";
  }
  const age = ageFromIso(iso, today);
  if (age < MIN_AGE) {
    return `Vous devez avoir au moins ${MIN_AGE} ans pour utiliser l'application.`;
  }
  if (age > 120) {
    return "Cette date de naissance n'est pas valide.";
  }
  return null;
}

export function isoToParts(iso: string): DateParts {
  const [year, month, day] = iso.split("-");
  return { day: day ?? "", month: month ?? "", year: year ?? "" };
}

export function validateFirstName(value: string): string | null {
  const name = value.trim();
  if (!name) {
    return "Saisissez votre prénom.";
  }
  if ([...name].length > NAME_MAX) {
    return `Votre prénom ne peut pas dépasser ${NAME_MAX} caractères.`;
  }
  return null;
}

export function clampAgeRange(min: number, max: number): { min: number; max: number } {
  const lo = Math.min(Math.max(Math.round(min), MIN_AGE), MAX_AGE);
  const hi = Math.min(Math.max(Math.round(max), MIN_AGE), MAX_AGE);
  return lo <= hi ? { min: lo, max: hi } : { min: hi, max: lo };
}
