const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) {
    return "Saisissez votre adresse e-mail.";
  }
  if (email.length > 254 || !EMAIL_PATTERN.test(email)) {
    return "Cette adresse e-mail n'est pas valide.";
  }
  return null;
}

/** Mirrors the server rule: 8 to 72 characters with at least one letter and one digit. */
export function validatePassword(value: string): string | null {
  if (value.length < 8) {
    return "Le mot de passe doit contenir au moins 8 caractères.";
  }
  if (value.length > 72) {
    return "Le mot de passe est trop long (72 caractères maximum).";
  }
  if (!/[A-Za-zÀ-ɏ]/.test(value) || !/\d/.test(value)) {
    return "Utilisez au moins une lettre et un chiffre.";
  }
  return null;
}

export function validateFirstName(value: string): string | null {
  const name = value.trim();
  if (!name) {
    return "Indiquez votre prénom.";
  }
  if (name.length > 40) {
    return "Le prénom est trop long.";
  }
  if (!/^[\p{L}][\p{L} '’.-]*$/u.test(name)) {
    return "Le prénom ne peut contenir que des lettres.";
  }
  return null;
}
