export const MIN_AGE = 18;

export type BirthDateParts = { day: string; month: string; year: string };

/** Returns an ISO date (YYYY-MM-DD) for a valid calendar date, otherwise null. */
export function toIsoDate({ day, month, year }: BirthDateParts): string | null {
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return null;
  }
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return null;
  }
  return `${year}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

export function fromIsoDate(iso: string): BirthDateParts {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day: day.replace(/^0/, ""), month: month.replace(/^0/, ""), year };
}

/** Whole years between an ISO birth date and `now`. */
export function ageFromIso(iso: string, now: Date = new Date()): number {
  const [y = 0, m = 1, d = 1] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  const beforeBirthday = now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d);
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

export function validateBirthDate(
  parts: BirthDateParts,
  now: Date = new Date()
): { iso: string } | { error: string } {
  const iso = toIsoDate(parts);
  if (!iso) {
    return { error: "Entrez une date de naissance valide (JJ / MM / AAAA)." };
  }
  const age = ageFromIso(iso, now);
  if (age < MIN_AGE) {
    return { error: `Vous devez avoir au moins ${MIN_AGE} ans pour utiliser Sillage.` };
  }
  if (age > 110) {
    return { error: "Entrez une date de naissance valide." };
  }
  return { iso };
}
