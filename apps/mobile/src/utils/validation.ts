export function validateEmail(value: string): string | null {
  const email = value.trim();
  if (!email) {
    return "Renseignez votre e-mail.";
  }
  if (!/^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/.test(email) || email.length > 254) {
    return "Cette adresse e-mail n’est pas valide.";
  }
  return null;
}

/** Mirrors the server policy so members get instant feedback. */
export function validatePassword(value: string): string | null {
  if (value.length < 8) {
    return "8 caractères minimum.";
  }
  if (new TextEncoder().encode(value).length > 72) {
    return "72 caractères maximum.";
  }
  if (!/\p{L}/u.test(value) || !/\d/.test(value)) {
    return "Ajoutez au moins une lettre et un chiffre.";
  }
  return null;
}

export function validateFirstName(value: string): string | null {
  const name = value.trim().replace(/\s+/g, " ");
  if (!name) {
    return "Indiquez votre prénom.";
  }
  if (name.length > 30 || !/^\p{L}[\p{L}\p{M}'’ -]*$/u.test(name)) {
    return "Lettres, espaces, apostrophes et tirets uniquement (30 max).";
  }
  return null;
}
