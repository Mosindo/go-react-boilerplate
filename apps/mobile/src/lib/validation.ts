const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function isValidEmail(value: string): boolean {
  const trimmed = value.trim();
  return trimmed.length <= 254 && EMAIL.test(trimmed);
}

export function isValidPassword(value: string): boolean {
  return value.length >= 8 && value.length <= 72;
}

export function isValidFirstName(value: string): boolean {
  const trimmed = value.trim().replace(/\s+/g, " ");
  return trimmed.length >= 1 && trimmed.length <= 40 && /^\p{L}[\p{L}\p{M}' .-]*$/u.test(trimmed);
}
