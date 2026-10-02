export const MIN_AGE = 18;
export const MAX_AGE = 99;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function isValidEmail(value: string): boolean {
  return EMAIL_RE.test(value.trim());
}

export function passwordError(value: string): string | null {
  if (value.length < 8) return "8 caractères minimum.";
  if (new TextEncoder().encode(value).length > 72) return "72 caractères maximum.";
  return null;
}

/** Completed years between a birth date and `now`. */
export function ageOn(birth: Date, now: Date = new Date()): number {
  let age = now.getFullYear() - birth.getFullYear();
  const m = now.getMonth() - birth.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < birth.getDate())) age--;
  return age;
}

/** Parses "JJ/MM/AAAA" into an ISO date (YYYY-MM-DD), or null if it is not a real date. */
export function parseBirthDate(input: string): string | null {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) return null;
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (
    date.getUTCFullYear() !== year ||
    date.getUTCMonth() !== month - 1 ||
    date.getUTCDate() !== day
  ) {
    return null;
  }
  return `${match[3]}-${match[2]}-${match[1]}`;
}

export function birthDateError(input: string, now: Date = new Date()): string | null {
  const iso = parseBirthDate(input);
  if (!iso) return "Date invalide (JJ/MM/AAAA).";
  const age = ageOn(new Date(`${iso}T00:00:00`), now);
  if (age < MIN_AGE) return "Alba est réservé aux personnes majeures (18 ans et plus).";
  if (age > MAX_AGE) return "Date invalide.";
  return null;
}

/** ISO date -> "JJ/MM/AAAA". */
export function isoToBirthInput(iso: string): string {
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : "";
}

/** Inserts slashes while typing: "12031990" -> "12/03/1990". */
export function maskBirthInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  const parts = [digits.slice(0, 2), digits.slice(2, 4), digits.slice(4, 8)].filter(Boolean);
  return parts.join("/");
}
