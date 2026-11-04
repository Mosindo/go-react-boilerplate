export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const MAX_MESSAGE_LENGTH = 2000;
export const MAX_BIO_LENGTH = 500;

export type Preferences = {
  interestedIn: string[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export function validatePreferences(p: Preferences): string | null {
  if (p.interestedIn.length === 0) return "Choisissez au moins une option";
  if (p.minAge < MIN_AGE || p.maxAge > MAX_AGE) return `L'âge doit être entre ${MIN_AGE} et ${MAX_AGE} ans`;
  if (p.minAge > p.maxAge) return "L'âge minimum dépasse l'âge maximum";
  if (p.maxDistanceKm < 1 || p.maxDistanceKm > 500) return "La distance doit être entre 1 et 500 km";
  return null;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

export function validateEmail(email: string): string | null {
  const v = email.trim();
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) || v.length > 254) return "Adresse e-mail invalide";
  return null;
}

export function validatePassword(password: string): string | null {
  if (password.length < 8) return "8 caractères minimum";
  if (password.length > 72) return "72 caractères maximum";
  if (!/\p{L}/u.test(password) || !/\d/.test(password)) return "Au moins une lettre et un chiffre";
  return null;
}

export type CleanedMessage = { ok: true; body: string } | { ok: false; error: string };

export function cleanMessage(raw: string): CleanedMessage {
  // eslint-disable-next-line no-control-regex
  const body = raw.replace(/[\u0000-\u0008\u000B-\u001F\u007F]/g, "").trim();
  if (body.length === 0) return { ok: false, error: "Message vide" };
  if ([...body].length > MAX_MESSAGE_LENGTH) return { ok: false, error: "Message trop long (2000 caractères max)" };
  return { ok: true, body };
}

export function validateFirstName(name: string): string | null {
  const n = name.trim().replace(/\s+/g, " ");
  if (n.length < 1) return "Prénom requis";
  if ([...n].length > 40) return "40 caractères maximum";
  return null;
}
