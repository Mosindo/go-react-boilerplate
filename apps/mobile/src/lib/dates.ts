const MIN_AGE = 18;
const MAX_AGE = 99;

export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

export function ageOn(birthIso: string, now: Date): number {
  const [year, month, day] = birthIso.split("-").map(Number);
  let age = now.getFullYear() - year;
  const beforeBirthday = now.getMonth() + 1 < month || (now.getMonth() + 1 === month && now.getDate() < day);
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

/** Validates the three birth-date fields typed by the user and enforces the 18+ rule. */
export function parseBirthDate(day: string, month: string, year: string, now: Date = new Date()): BirthDateResult {
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return { ok: false, error: "Enter your birth date as day, month and a 4-digit year." };
  }
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  const probe = new Date(Date.UTC(y, m - 1, d));
  if (probe.getUTCFullYear() !== y || probe.getUTCMonth() !== m - 1 || probe.getUTCDate() !== d) {
    return { ok: false, error: "That date does not exist." };
  }
  const iso = `${year}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  const age = ageOn(iso, now);
  if (age < MIN_AGE) {
    return { ok: false, error: "You must be at least 18 years old to use this app." };
  }
  if (age > MAX_AGE) {
    return { ok: false, error: "Please check your birth year." };
  }
  return { ok: true, iso, age };
}

export function splitIsoDate(iso: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day: day.replace(/^0/, ""), month: month.replace(/^0/, ""), year };
}

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Compact timestamp for lists: "now", "5m", "3h", "Tue", "12 Mar". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const then = new Date(iso);
  const diff = now.getTime() - then.getTime();
  if (diff < MINUTE) {
    return "now";
  }
  if (diff < HOUR) {
    return `${Math.floor(diff / MINUTE)}m`;
  }
  if (diff < DAY) {
    return `${Math.floor(diff / HOUR)}h`;
  }
  if (diff < 7 * DAY) {
    return then.toLocaleDateString(undefined, { weekday: "short" });
  }
  return then.toLocaleDateString(undefined, { day: "numeric", month: "short" });
}

/** Clock time shown under chat bubbles. */
export function formatClock(iso: string): string {
  return new Date(iso).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}
