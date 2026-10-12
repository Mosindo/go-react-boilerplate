export const MIN_AGE = 18;
export const MAX_AGE = 99;

export type DateParts = { day: string; month: string; year: string };

/** Age in whole years on `today` for an ISO birth date; null when the date is invalid. */
export function ageOn(isoDate: string, today: Date = new Date()): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(isoDate);
  if (!match) {
    return null;
  }
  const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])];
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return null;
  }
  let age = today.getUTCFullYear() - year;
  const before = today.getUTCMonth() + 1 < month || (today.getUTCMonth() + 1 === month && today.getUTCDate() < day);
  if (before) {
    age -= 1;
  }
  return age;
}

/** Converts DD / MM / YYYY inputs to an ISO date, or an error message in French. */
export function parseBirthDate(parts: DateParts, today: Date = new Date()): { iso: string; error: null } | { iso: null; error: string } {
  const { day, month, year } = parts;
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return { iso: null, error: "Saisissez une date valide (JJ / MM / AAAA)." };
  }
  const iso = `${year}-${month.padStart(2, "0")}-${day.padStart(2, "0")}`;
  const age = ageOn(iso, today);
  if (age === null) {
    return { iso: null, error: "Cette date n'existe pas." };
  }
  if (age < MIN_AGE) {
    return { iso: null, error: "Vous devez avoir au moins 18 ans pour utiliser l'application." };
  }
  if (age > MAX_AGE) {
    return { iso: null, error: "Cette date de naissance n'est pas valide." };
  }
  return { iso, error: null };
}

export function splitIsoDate(iso: string): DateParts {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day: String(Number(day) || ""), month: String(Number(month) || ""), year };
}

export function validateEmail(value: string): string | null {
  const email = value.trim();
  if (!email) {
    return "Saisissez votre adresse e-mail.";
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || email.length > 254) {
    return "Cette adresse e-mail n'est pas valide.";
  }
  return null;
}

export function validatePassword(value: string): string | null {
  if (value.length < 8) {
    return "Le mot de passe doit contenir au moins 8 caractères.";
  }
  if (new TextEncoder().encode(value).length > 72) {
    return "Le mot de passe est trop long (72 octets maximum).";
  }
  return null;
}

export function validateFirstName(value: string): string | null {
  const name = value.trim();
  if (!name) {
    return "Votre prénom est requis.";
  }
  if (name.length > 40) {
    return "Le prénom est trop long.";
  }
  if (!/^[\p{L}\p{M}][\p{L}\p{M} '’-]*$/u.test(name)) {
    return "Le prénom ne peut contenir que des lettres.";
  }
  return null;
}
