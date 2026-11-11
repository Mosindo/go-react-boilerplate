/** Mirrors the server password policy so users get instant feedback. */
export function passwordProblem(password: string): string | null {
  if (password.length < 8) {
    return "8 caractères minimum.";
  }
  if (password.length > 72) {
    return "72 caractères maximum.";
  }
  if (!/\p{L}/u.test(password) || !/\d/.test(password)) {
    return "Ajoute au moins une lettre et un chiffre.";
  }
  return null;
}

export function looksLikeEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim());
}
