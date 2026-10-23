import { t } from "../i18n/fr";

const pad = (value: number) => String(value).padStart(2, "0");

export function toIsoDate(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

/** Builds "YYYY-MM-DD" from user inputs, or null when it is not a real past calendar date. */
export function parseBirthDate(
  day: string,
  month: string,
  year: string,
  now: Date = new Date()
): string | null {
  if (!/^\d{1,2}$/.test(day.trim()) || !/^\d{1,2}$/.test(month.trim())) return null;
  if (!/^\d{4}$/.test(year.trim())) return null;
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (y < 1900 || m < 1 || m > 12 || d < 1) return null;
  const check = new Date(Date.UTC(y, m - 1, d));
  if (check.getUTCFullYear() !== y || check.getUTCMonth() !== m - 1 || check.getUTCDate() !== d) {
    return null;
  }
  const iso = `${y}-${pad(m)}-${pad(d)}`;
  return iso > toIsoDate(now) ? null : iso;
}

/** Full years elapsed between an ISO birth date and `now`. */
export function calculateAge(birthDate: string, now: Date = new Date()): number {
  const [y, m, d] = birthDate.split("-").map(Number);
  let age = now.getFullYear() - y;
  const monthDelta = now.getMonth() + 1 - m;
  if (monthDelta < 0 || (monthDelta === 0 && now.getDate() < d)) age -= 1;
  return age;
}

export function isAdult(birthDate: string, now: Date = new Date()): boolean {
  return calculateAge(birthDate, now) >= 18;
}

export function splitBirthDate(birthDate: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = birthDate.split("-");
  return { day, month, year };
}

const MONTHS_SHORT = [
  "janv.",
  "févr.",
  "mars",
  "avr.",
  "mai",
  "juin",
  "juil.",
  "août",
  "sept.",
  "oct.",
  "nov.",
  "déc."
];
const WEEKDAYS_SHORT = ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."];

export function formatClock(date: Date): string {
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

function dayDifference(date: Date, now: Date): number {
  return Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
}

function shortDate(date: Date, now: Date): string {
  const base = `${date.getDate()} ${MONTHS_SHORT[date.getMonth()]}`;
  return date.getFullYear() === now.getFullYear() ? base : `${base} ${date.getFullYear()}`;
}

/** Chat bubble time: "14:32". */
export function formatMessageTime(iso: string): string {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? "" : formatClock(date);
}

/** Compact list time: clock today, "Hier", weekday within a week, then "12 mars" / "12 mars 2024". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const diff = dayDifference(date, now);
  if (diff <= 0) return formatClock(date);
  if (diff === 1) return t.time.yesterday;
  if (diff < 7) return WEEKDAYS_SHORT[date.getDay()];
  return shortDate(date, now);
}

/** Day separator label in chats. */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diff = dayDifference(date, now);
  if (diff <= 0) return t.chat.today;
  if (diff === 1) return t.chat.yesterday;
  return shortDate(date, now);
}

/** True when two ISO timestamps fall on different calendar days. */
export function isDifferentDay(a: string, b: string): boolean {
  return startOfDay(new Date(a)) !== startOfDay(new Date(b));
}
