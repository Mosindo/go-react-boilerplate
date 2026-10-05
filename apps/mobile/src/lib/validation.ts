const EMAIL_PATTERN = /^[^\s@<>,;"]+@[^\s@<>,;"]+\.[^\s@<>,;"]+$/;

export const PASSWORD_MIN = 8;
export const PASSWORD_MAX_BYTES = 72; // bcrypt ignores anything beyond 72 bytes

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidEmail(value: string): boolean {
  const email = normalizeEmail(value);
  return email.length <= 254 && EMAIL_PATTERN.test(email) && !email.endsWith(".");
}

/** Returns a French message describing what is wrong with the password, or null when it is acceptable. */
export function passwordProblem(password: string): string | null {
  if (password.length < PASSWORD_MIN) return `Le mot de passe doit contenir au moins ${PASSWORD_MIN} caractères.`;
  if (new TextEncoder().encode(password).length > PASSWORD_MAX_BYTES) return "Le mot de passe est trop long (72 octets maximum).";
  return null;
}

/** First names accept letters, spaces, hyphens, apostrophes and dots — the same rule as the API. */
export function isValidFirstName(value: string): boolean {
  const name = value.trim();
  return name.length >= 1 && name.length <= 40 && /^[\p{L} '’.-]+$/u.test(name);
}
