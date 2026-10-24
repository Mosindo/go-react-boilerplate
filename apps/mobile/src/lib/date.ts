export const MIN_AGE = 18;

export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

/** Full years elapsed between `birth` and `now` (UTC-safe: only calendar fields are compared). */
export function ageAt(birth: { year: number; month: number; day: number }, now: Date): number {
  let age = now.getFullYear() - birth.year;
  const m = now.getMonth() + 1;
  if (m < birth.month || (m === birth.month && now.getDate() < birth.day)) age -= 1;
  return age;
}

/** Validates day/month/year text fields and returns the ISO date the API expects. */
export function parseBirthDate(
  day: string,
  month: string,
  year: string,
  now: Date = new Date(),
): BirthDateResult {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (![day, month, year].every((v) => /^\d+$/.test(v.trim())) || year.trim().length !== 4) {
    return { ok: false, error: "Entrez une date valide (jj / mm / aaaa)." };
  }
  const candidate = new Date(Date.UTC(y, m - 1, d));
  if (
    candidate.getUTCFullYear() !== y ||
    candidate.getUTCMonth() !== m - 1 ||
    candidate.getUTCDate() !== d
  ) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const age = ageAt({ year: y, month: m, day: d }, now);
  if (age < MIN_AGE)
    return { ok: false, error: "Alba est réservé aux personnes majeures (18 ans et plus)." };
  if (age > 120) return { ok: false, error: "Entrez une date valide." };
  const iso = `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return { ok: true, iso, age };
}

export function splitIsoDate(iso: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day, month, year };
}

const pad = (n: number) => String(n).padStart(2, "0");

/** Short, locale-independent timestamp: "14:05" today, "hier", otherwise "12/05". */
export function formatShortTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const diffDays = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (diffDays <= 0) return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
  if (diffDays === 1) return "hier";
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}
