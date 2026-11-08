export const MIN_AGE = 18;
export const MAX_AGE = 99;

export function ageFromIso(iso: string, now: Date = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getUTCFullYear() - y;
  const month = now.getUTCMonth() + 1;
  if (month < m || (month === m && now.getUTCDate() < d)) age -= 1;
  return age;
}

export type BirthDateResult = { iso: string; error: null } | { iso: null; error: string };

/** Validates a day/month/year triple typed by the user and returns an ISO date (YYYY-MM-DD). */
export function parseBirthDate(
  day: string,
  month: string,
  year: string,
  now: Date = new Date()
): BirthDateResult {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return { iso: null, error: "Entre ta date de naissance au format JJ / MM / AAAA." };
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return { iso: null, error: "Cette date n'existe pas." };
  }
  const iso = `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  const age = ageFromIso(iso, now);
  if (age < MIN_AGE) return { iso: null, error: "Tu dois avoir au moins 18 ans pour utiliser Ember." };
  if (age > MAX_AGE) return { iso: null, error: "Cette date de naissance n'est pas valide." };
  return { iso, error: null };
}
