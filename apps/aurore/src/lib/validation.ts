const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(email: string): string {
  return email.trim().toLowerCase();
}

export function validateEmail(email: string): string | null {
  const e = normalizeEmail(email);
  if (!e) return "Entrez votre email.";
  if (e.length > 254 || !EMAIL_RE.test(e)) return "Cet email ne semble pas valide.";
  return null;
}

export function validatePassword(password: string): string | null {
  if (password.length < 8) return "Le mot de passe doit contenir au moins 8 caractères.";
  // bcrypt only uses the first 72 bytes; the API refuses longer passwords.
  if (new TextEncoder().encode(password).length > 72) return "Le mot de passe est trop long (72 caractères maximum).";
  return null;
}

export function validateFirstName(name: string): string | null {
  const n = name.trim();
  if (!n) return "Entrez votre prénom.";
  if ([...n].length > 40) return "Le prénom est trop long (40 caractères maximum).";
  return null;
}

export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const MAX_INTERESTS = 10;
export const MAX_BIO = 500;

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
