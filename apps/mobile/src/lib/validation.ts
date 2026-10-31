export const MIN_AGE = 18;
export const MAX_AGE = 120;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(email: string): string {
  return email.trim().toLowerCase();
}

export function validateEmail(email: string): string | null {
  const value = normalizeEmail(email);
  if (!value) return "Saisissez votre e-mail.";
  if (value.length > 254 || !EMAIL_RE.test(value)) return "Cette adresse e-mail n'est pas valide.";
  return null;
}

/** bcrypt only uses the first 72 bytes, so the API caps passwords at 72. */
export function validatePassword(password: string): string | null {
  if (password.length < 8) return "Le mot de passe doit contenir au moins 8 caractères.";
  if (new TextEncoder().encode(password).length > 72) return "Le mot de passe est trop long (72 octets maximum).";
  return null;
}

export function ageFromBirthDate(birth: Date, now: Date = new Date()): number {
  let age = now.getFullYear() - birth.getFullYear();
  const beforeBirthday = now.getMonth() < birth.getMonth() || (now.getMonth() === birth.getMonth() && now.getDate() < birth.getDate());
  if (beforeBirthday) age -= 1;
  return age;
}

export type BirthDateResult = { ok: true; iso: string } | { ok: false; error: string };

/** Validates a day/month/year triple and the 18+ rule. The server re-checks it. */
export function parseBirthDate(day: string, month: string, year: string, now: Date = new Date()): BirthDateResult {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { ok: false, error: "Saisissez une date de naissance valide (JJ / MM / AAAA)." };
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const local = new Date(y, m - 1, d);
  const age = ageFromBirthDate(local, now);
  if (local > now || age > MAX_AGE) return { ok: false, error: "Cette date de naissance n'est pas valide." };
  if (age < MIN_AGE) return { ok: false, error: "Alma est réservée aux personnes majeures (18 ans et plus)." };
  const iso = `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return { ok: true, iso };
}

export function describeApiError(status: number | undefined, fallback: string): string {
  switch (status) {
    case 401:
      return "Identifiants incorrects.";
    case 403:
      return "Action non autorisée.";
    case 409:
      return fallback;
    case 429:
      return "Trop de tentatives. Patientez un instant.";
    default:
      return fallback;
  }
}
