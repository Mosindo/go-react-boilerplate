export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizeEmail(value));
}

export function passwordError(value: string): string | null {
  if (value.length < 8) {
    return "Use at least 8 characters.";
  }
  if (value.length > 128) {
    return "Use at most 128 characters.";
  }
  return null;
}
