const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export const PASSWORD_MIN_LENGTH = 8;
export const PASSWORD_MAX_BYTES = 72;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) {
    return "Entrez votre adresse e-mail.";
  }
  if (email.length > 254 || !EMAIL_PATTERN.test(email)) {
    return "Cette adresse e-mail n'est pas valide.";
  }
  return null;
}

function utf8Length(value: string): number {
  let bytes = 0;
  for (const char of value) {
    const code = char.codePointAt(0) ?? 0;
    bytes += code < 0x80 ? 1 : code < 0x800 ? 2 : code < 0x10000 ? 3 : 4;
  }
  return bytes;
}

export function validatePassword(value: string): string | null {
  if (Array.from(value).length < PASSWORD_MIN_LENGTH) {
    return `Le mot de passe doit contenir au moins ${PASSWORD_MIN_LENGTH} caractères.`;
  }
  if (utf8Length(value) > PASSWORD_MAX_BYTES) {
    return "Le mot de passe est trop long.";
  }
  if (!value.trim()) {
    return "Le mot de passe ne peut pas être vide.";
  }
  return null;
}
