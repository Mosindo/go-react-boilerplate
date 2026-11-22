/** Client-side checks mirror the API rules for instant feedback; the server remains the authority. */

export function normalizeEmail(email: string): string {
  return email.trim().toLowerCase();
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizeEmail(email));
}

export function passwordProblem(password: string): string | null {
  if (password.length < 8) {
    return "8 caractères minimum.";
  }
  if (password.length > 128) {
    return "128 caractères maximum.";
  }
  if (!/\p{L}/u.test(password) || !/\d/.test(password)) {
    return "Ajoutez au moins une lettre et un chiffre.";
  }
  return null;
}

export function firstNameProblem(name: string): string | null {
  const trimmed = name.trim();
  if (!trimmed) {
    return "Indiquez votre prénom.";
  }
  if (trimmed.length > 40) {
    return "40 caractères maximum.";
  }
  if (!/^[\p{L}\p{M}' ’-]+$/u.test(trimmed)) {
    return "Uniquement des lettres, espaces, tirets ou apostrophes.";
  }
  return null;
}

export function ageAt(birthdate: Date, now = new Date()): number {
  let age = now.getFullYear() - birthdate.getFullYear();
  const m = now.getMonth() - birthdate.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < birthdate.getDate())) {
    age--;
  }
  return age;
}

/** Parses day/month/year fields into an ISO date, or explains the problem. */
export function parseBirthdate(day: string, month: string, year: string, now = new Date()): { iso?: string; error?: string } {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { error: "Date incomplète (JJ / MM / AAAA)." };
  }
  const date = new Date(y, m - 1, d);
  if (date.getFullYear() !== y || date.getMonth() !== m - 1 || date.getDate() !== d) {
    return { error: "Cette date n'existe pas." };
  }
  const age = ageAt(date, now);
  if (date > now || age > 100) {
    return { error: "Date de naissance invalide." };
  }
  if (age < 18) {
    return { error: "Lueur est réservé aux personnes majeures." };
  }
  const iso = `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return { iso };
}
