export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function ageFromBirthDate(birthDate: string, now: Date = new Date()): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(birthDate.trim());
  if (!m) return null;
  const [y, mo, d] = [Number(m[1]), Number(m[2]), Number(m[3])];
  const date = new Date(Date.UTC(y, mo - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== mo - 1 || date.getUTCDate() !== d) return null;
  let age = now.getUTCFullYear() - y;
  if (now.getUTCMonth() + 1 < mo || (now.getUTCMonth() + 1 === mo && now.getUTCDate() < d)) age -= 1;
  return age;
}

export function validateBirthDate(birthDate: string, now: Date = new Date()): string | null {
  const age = ageFromBirthDate(birthDate, now);
  if (age === null) return "Format attendu : AAAA-MM-JJ";
  if (age < MIN_AGE) return "Aube est réservé aux personnes de 18 ans et plus.";
  if (age > MAX_AGE) return "Date de naissance invalide.";
  return null;
}

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
export function validateEmail(email: string): string | null {
  return EMAIL.test(email.trim()) ? null : "Adresse e-mail invalide.";
}

export function validatePassword(password: string): string | null {
  if (password.length < 8) return "8 caractères minimum.";
  if (new TextEncoder().encode(password).length > 72) return "72 caractères maximum.";
  return null;
}
