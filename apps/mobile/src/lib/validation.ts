export function isValidEmail(value: string): boolean {
  const email = value.trim();
  return email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

export function passwordProblem(value: string): string | null {
  if (value.length < 8) {
    return "Au moins 8 caractères.";
  }
  if (value.length > 128) {
    return "128 caractères maximum.";
  }
  return null;
}

export function describeError(error: unknown, fallback = "Une erreur est survenue."): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
