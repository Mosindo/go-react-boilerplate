/** Utilitaires de date purs (sans dependance React Native), testes avec vitest. */

/** Convertit "JJ/MM/AAAA" en "AAAA-MM-JJ". Retourne null si la date n'existe pas. */
export function parseBirthInput(input: string): string | null {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) {
    return null;
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return null;
  }
  return `${match[3]}-${match[2]}-${match[1]}`;
}

/** "AAAA-MM-JJ" -> "JJ/MM/AAAA". */
export function formatBirthInput(iso: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
  return match ? `${match[3]}/${match[2]}/${match[1]}` : "";
}

/** Masque de saisie JJ/MM/AAAA: ne garde que les chiffres et insere les "/". */
export function maskBirthInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

/** Age complet a la date `now` pour une date ISO "AAAA-MM-JJ". */
export function ageFromIso(iso: string, now: Date = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  const month = now.getMonth() + 1;
  if (month < m || (month === m && now.getDate() < d)) {
    age -= 1;
  }
  return age;
}

export const MIN_AGE = 18;
export const MAX_AGE = 99;

/** Pour un champ date de naissance: renvoie un message d'erreur ou null. */
export function validateBirthInput(input: string, now: Date = new Date()): string | null {
  const iso = parseBirthInput(input);
  if (!iso) {
    return "Date invalide (JJ/MM/AAAA)";
  }
  const age = ageFromIso(iso, now);
  if (age < MIN_AGE) {
    return "Vous devez avoir au moins 18 ans";
  }
  if (age > MAX_AGE) {
    return "Date de naissance invalide";
  }
  return null;
}
