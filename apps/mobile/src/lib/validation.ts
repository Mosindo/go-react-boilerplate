const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateEmail(value: string): string | null {
  const email = value.trim();
  if (!email) return "Entrez votre adresse e-mail.";
  if (!EMAIL.test(email) || email.length > 254) return "Adresse e-mail invalide.";
  return null;
}

/** The API hashes with bcrypt, which only considers the first 72 bytes. */
export function validateNewPassword(value: string): string | null {
  if (value.length < 8) return "Au moins 8 caractères.";
  if (new TextEncoder().encode(value).length > 72) return "72 caractères maximum.";
  return null;
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
