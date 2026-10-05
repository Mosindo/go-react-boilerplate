export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

export const MIN_AGE = 18;
export const MAX_AGE = 120;

/** Age in full years on `now` for a "YYYY-MM-DD" birth date. */
export function ageOn(iso: string, now: Date): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) age--;
  return age;
}

/**
 * Validates the three birth-date fields. The server enforces the same 18+ rule; checking here only gives
 * immediate feedback and never replaces the server check.
 */
export function parseBirthDate(day: string, month: string, year: string, now: Date = new Date()): BirthDateResult {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!/^\d{1,2}$/.test(day.trim()) || !/^\d{1,2}$/.test(month.trim()) || !/^\d{4}$/.test(year.trim())) {
    return { ok: false, error: "Entrez une date valide (jj / mm / aaaa)." };
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (m < 1 || m > 12 || d < 1 || date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return { ok: false, error: "Cette date n'existe pas." };
  }
  const iso = `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  const age = ageOn(iso, now);
  if (age < MIN_AGE) return { ok: false, error: "Vous devez avoir au moins 18 ans pour utiliser l'application." };
  if (age > MAX_AGE) return { ok: false, error: "Cette date n'est pas plausible." };
  return { ok: true, iso, age };
}

export function splitIsoDate(iso: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day: String(Number(day) || ""), month: String(Number(month) || ""), year };
}

const timeFormat = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormat = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "long" });
const weekdayFormat = new Intl.DateTimeFormat("fr-FR", { weekday: "long" });

export function formatTime(iso: string): string {
  return timeFormat.format(new Date(iso));
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

/** "14:05" today, "Hier", weekday within a week, otherwise "3 mars". */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) return formatTime(iso);
  if (days === 1) return "Hier";
  if (days < 7) return weekdayFormat.format(date);
  return dayFormat.format(date);
}

/** Label for a chat day separator. */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) return "Aujourd'hui";
  if (days === 1) return "Hier";
  return dayFormat.format(date);
}

export function isSameDay(a: string, b: string): boolean {
  return startOfDay(new Date(a)) === startOfDay(new Date(b));
}
