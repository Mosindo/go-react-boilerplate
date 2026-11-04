export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

export function ageOn(year: number, month: number, day: number, today: Date): number {
  let age = today.getUTCFullYear() - year;
  const m = today.getUTCMonth() + 1;
  if (m < month || (m === month && today.getUTCDate() < day)) age -= 1;
  return age;
}

/** Parses a DD/MM/YYYY string, checks it is a real date and that the person is 18+. */
export function parseBirthDate(input: string, today: Date = new Date()): BirthDateResult {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) return { ok: false, error: "Format attendu : JJ/MM/AAAA" };
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const d = new Date(Date.UTC(year, month - 1, day));
  if (d.getUTCFullYear() !== year || d.getUTCMonth() !== month - 1 || d.getUTCDate() !== day) {
    return { ok: false, error: "Cette date n'existe pas" };
  }
  const age = ageOn(year, month, day, today);
  if (age < 0 || age > 120) return { ok: false, error: "Date de naissance invalide" };
  if (age < 18) return { ok: false, error: "Vous devez avoir au moins 18 ans" };
  const iso = `${match[3]}-${match[2]}-${match[1]}`;
  return { ok: true, iso, age };
}

/** Converts an ISO date (YYYY-MM-DD) to DD/MM/YYYY for display. */
export function isoToFrench(iso: string): string {
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : iso;
}

/** Auto-inserts slashes while the user types digits. */
export function maskBirthDate(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) return digits;
  if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}
