// Client-side checks give instant feedback; the server remains the authority.

export const PASSWORD_MIN = 8;
export const PASSWORD_MAX_BYTES = 72;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function isValidEmail(value: string): boolean {
  const email = normalizeEmail(value);
  return email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email);
}

function utf8Length(value: string): number {
  let bytes = 0;
  for (const char of value) {
    const code = char.codePointAt(0) ?? 0;
    bytes += code < 0x80 ? 1 : code < 0x800 ? 2 : code < 0x10000 ? 3 : 4;
  }
  return bytes;
}

export type PasswordIssue = "short" | "long" | null;

export function checkPassword(value: string): PasswordIssue {
  if (Array.from(value).length < PASSWORD_MIN) {
    return "short";
  }
  if (utf8Length(value) > PASSWORD_MAX_BYTES) {
    return "long";
  }
  return null;
}

export function cleanName(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

export function isValidFirstName(value: string): boolean {
  const length = Array.from(cleanName(value)).length;
  return length >= 1 && length <= 40;
}
