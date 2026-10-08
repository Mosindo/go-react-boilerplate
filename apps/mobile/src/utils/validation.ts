// Client-side checks mirror the server rules to give instant feedback; the server remains the authority.

const EMAIL_PATTERN = /^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/;

export function validateEmail(value: string): string | null {
  const email = value.trim();
  if (!email) return "Saisissez votre adresse e-mail.";
  if (email.length > 254 || !EMAIL_PATTERN.test(email)) return "Adresse e-mail invalide.";
  return null;
}

export function validatePassword(value: string): string | null {
  if (value.length < 8) return "Au moins 8 caractères.";
  if (new TextEncoder().encode(value).length > 72) return "72 caractères maximum.";
  if (!/\p{L}/u.test(value) || !/[^\p{L}]/u.test(value)) return "Mélangez lettres et chiffres ou symboles.";
  return null;
}

export function validateFirstName(value: string): string | null {
  const name = value.trim().replace(/\s+/g, " ");
  if (!name) return "Saisissez votre prénom.";
  if ([...name].length > 40) return "40 caractères maximum.";
  if (!/^[\p{L}\p{M}' ’-]+$/u.test(name)) return "Lettres, espaces, tirets et apostrophes uniquement.";
  return null;
}

export type BirthdateParts = { day: string; month: string; year: string };

/** Builds an ISO date (YYYY-MM-DD) from the three inputs, or an error message. */
export function parseBirthdate(parts: BirthdateParts, now: Date = new Date()): { iso: string; error: null } | { iso: null; error: string } {
  const day = Number(parts.day);
  const month = Number(parts.month);
  const year = Number(parts.year);
  if (!parts.day || !parts.month || !parts.year || [day, month, year].some((n) => !Number.isInteger(n))) {
    return { iso: null, error: "Saisissez votre date de naissance (JJ / MM / AAAA)." };
  }
  const date = new Date(Date.UTC(year, month - 1, day));
  if (year < 1900 || date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return { iso: null, error: "Cette date n'existe pas." };
  }
  const age = ageOn(date, now);
  if (age < 18) return { iso: null, error: "Vous devez avoir au moins 18 ans." };
  if (age > 99) return { iso: null, error: "Date de naissance invalide." };
  const pad = (n: number, size: number) => String(n).padStart(size, "0");
  return { iso: `${pad(year, 4)}-${pad(month, 2)}-${pad(day, 2)}`, error: null };
}

/** Full years elapsed; `birth` is read in UTC to match YYYY-MM-DD strings. */
export function ageOn(birth: Date, now: Date): number {
  let age = now.getFullYear() - birth.getUTCFullYear();
  const monthDiff = now.getMonth() - birth.getUTCMonth();
  if (monthDiff < 0 || (monthDiff === 0 && now.getDate() < birth.getUTCDate())) age -= 1;
  return age;
}

export function splitBirthdate(iso: string): BirthdateParts {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day, month, year };
}

/** Keeps only digits and truncates; used by the date inputs. */
export function digitsOnly(value: string, max: number): string {
  return value.replace(/\D/g, "").slice(0, max);
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
