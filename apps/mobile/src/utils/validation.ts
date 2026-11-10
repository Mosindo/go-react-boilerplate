export const MIN_AGE = 18;

export function ageFrom(birthdate: Date, now: Date = new Date()): number {
  let age = now.getFullYear() - birthdate.getFullYear();
  const m = now.getMonth() - birthdate.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < birthdate.getDate())) age -= 1;
  return age;
}

/** Validates day/month/year inputs and returns an ISO date (YYYY-MM-DD). */
export function parseBirthdate(day: string, month: string, year: string): { iso?: string; error?: string } {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { error: "Saisissez une date complète (JJ / MM / AAAA)." };
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return { error: "Cette date n'existe pas." };
  }
  const age = ageFrom(new Date(y, m - 1, d));
  if (age < MIN_AGE) return { error: "Lueur est réservée aux personnes de 18 ans et plus." };
  if (age > 110) return { error: "Vérifiez l'année de naissance." };
  return { iso: `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}` };
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
}

export function passwordIssue(password: string): string | null {
  if (password.length < 8) return "8 caractères minimum.";
  if (!/[0-9]/.test(password) || !/[^0-9]/.test(password)) return "Au moins une lettre et un chiffre.";
  return null;
}

export function isValidFirstName(name: string): boolean {
  const trimmed = name.trim();
  return trimmed.length >= 1 && trimmed.length <= 30 && /^[\p{L} '’.-]+$/u.test(trimmed) && /\p{L}/u.test(trimmed);
}
