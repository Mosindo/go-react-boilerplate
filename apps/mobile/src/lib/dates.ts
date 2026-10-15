// Pure date helpers (no React Native imports) so they can be unit tested.

export const MIN_AGE = 18;
export const MAX_AGE = 99;

export type BirthDateParts = { day: string; month: string; year: string };

/** Builds an ISO date (YYYY-MM-DD) from the three input fields, or null when it is not a real date. */
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
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) {
    age -= 1;
  }
  return age;
}

export type BirthDateCheck = "ok" | "invalid" | "underage" | "too_old";

export function checkBirthDate(parts: BirthDateParts, now: Date = new Date()): { status: BirthDateCheck; iso: string | null } {
  const iso = toIsoDate(parts);
  if (!iso) {
    return { status: "invalid", iso: null };
  }
  const age = ageFromIso(iso, now);
  if (age < 0) {
    return { status: "invalid", iso };
  }
  if (age < MIN_AGE) {
    return { status: "underage", iso };
  }
  if (age > MAX_AGE) {
    return { status: "too_old", iso };
  }
  return { status: "ok", iso };
}

const timeFormat = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormat = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });
const weekdayFormat = new Intl.DateTimeFormat("fr-FR", { weekday: "short" });

/** Clock time for chat bubbles, e.g. "14:32". */
export function formatClock(iso: string): string {
  return timeFormat.format(new Date(iso));
}

/** Compact relative label for lists: "14:32", "hier", "mar.", "12 mars". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) {
    return timeFormat.format(date);
  }
  if (days === 1) {
    return "hier";
  }
  if (days < 7) {
    return weekdayFormat.format(date);
  }
  return dayFormat.format(date);
}

/** Day separator label inside a conversation. */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const label = formatListTime(iso, now);
  return /^\d{2}:\d{2}$/.test(label) ? "Aujourd'hui" : label === "hier" ? "Hier" : label;
}

export function isSameDay(a: string, b: string): boolean {
  const da = new Date(a);
  const db = new Date(b);
  return da.getFullYear() === db.getFullYear() && da.getMonth() === db.getMonth() && da.getDate() === db.getDate();
}
