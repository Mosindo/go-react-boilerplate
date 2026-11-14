export const MIN_AGE = 18;

export function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(value.trim());
}

export function passwordProblem(value: string): string | null {
  if (value.length < 8) return "8 caractères minimum.";
  if (new TextEncoder().encode(value).length > 72) return "72 caractères maximum.";
  if (!/\p{L}/u.test(value) || !/\d/.test(value)) return "Au moins une lettre et un chiffre.";
  return null;
}

export function ageOn(birth: Date, now = new Date()): number {
  let age = now.getFullYear() - birth.getFullYear();
  const m = now.getMonth() - birth.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < birth.getDate())) age -= 1;
  return age;
}

/** Parses day/month/year fields into an ISO date, or returns an error. */
export function parseBirthdate(day: string, month: string, year: string, now = new Date()): { iso: string } | { error: string } {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { error: "Date incomplète." };
  }
  const date = new Date(y, m - 1, d);
  if (date.getFullYear() !== y || date.getMonth() !== m - 1 || date.getDate() !== d) {
    return { error: "Cette date n'existe pas." };
  }
  const age = ageOn(date, now);
  if (age < MIN_AGE) return { error: "Lueur est réservée aux personnes majeures." };
  if (age > 99) return { error: "Vérifiez l'année de naissance." };
  const pad = (n: number) => String(n).padStart(2, "0");
  return { iso: `${y}-${pad(m)}-${pad(d)}` };
}

export function isValidFirstName(value: string): boolean {
  const v = value.trim();
  return v.length >= 1 && v.length <= 40 && /^[\p{L}][\p{L} '’-]*$/u.test(v);
}
