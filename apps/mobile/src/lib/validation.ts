const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  return EMAIL.test(normalizeEmail(value)) ? null : "Adresse e-mail invalide.";
}

export function validatePassword(value: string): string | null {
  if (value.length < 8) return "8 caractères minimum.";
  if (new TextEncoder().encode(value).length > 72) return "72 caractères maximum.";
  return null;
}
