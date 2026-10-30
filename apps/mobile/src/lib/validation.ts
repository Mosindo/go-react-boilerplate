import { ageFromIso, parseBirthDate } from "./dates";

export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const PASSWORD_MIN = 10;
export const PASSWORD_MAX = 72;

export function validateEmail(email: string): string | null {
  const value = email.trim();
  if (!value) {
    return "Saisissez votre adresse e-mail.";
  }
  if (value.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    return "Cette adresse e-mail n'est pas valide.";
  }
  return null;
}

export function validateNewPassword(password: string): string | null {
  if (password.length < PASSWORD_MIN) {
    return `Le mot de passe doit contenir au moins ${PASSWORD_MIN} caractères.`;
  }
  // The server hashes with bcrypt, which only reads the first 72 bytes.
  if (new TextEncoder().encode(password).length > PASSWORD_MAX) {
    return `Le mot de passe ne peut pas dépasser ${PASSWORD_MAX} caractères.`;
  }
  return null;
}

export function validateDisplayName(name: string): string | null {
  const value = name.trim();
  if (!value) {
    return "Indiquez votre prénom.";
  }
  if (value.length > 50) {
    return "Votre prénom est trop long.";
  }
  return null;
}

export function validateBirthDate(input: string, now: Date = new Date()): { iso: string | null; error: string | null } {
  const iso = parseBirthDate(input);
  if (!iso) {
    return { iso: null, error: "Entrez une date valide au format JJ/MM/AAAA." };
  }
  const age = ageFromIso(iso, now);
  if (age < MIN_AGE) {
    return { iso: null, error: "Alba est réservé aux personnes de 18 ans et plus." };
  }
  if (age > MAX_AGE) {
    return { iso: null, error: "Cette date de naissance n'est pas valide." };
  }
  return { iso, error: null };
}
