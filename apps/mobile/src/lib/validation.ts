const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  return EMAIL_PATTERN.test(email) && email.length <= 254 ? null : "Adresse e-mail invalide";
}

/** Doit rester coherent avec la politique serveur: 8-72 caracteres, lettres et chiffres. */
export function validatePassword(value: string): string | null {
  if (value.length < 8) {
    return "8 caractères minimum";
  }
  if (value.length > 72) {
    return "72 caractères maximum";
  }
  if (!/[A-Za-zÀ-￿]/.test(value) || !/\d/.test(value)) {
    return "Utilisez des lettres et des chiffres";
  }
  return null;
}
