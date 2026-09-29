const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const WEEKDAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

function pad2(value: number): string {
  return value < 10 ? `0${value}` : String(value);
}

function parse(iso: string): Date | null {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? null : date;
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

/** Whole calendar days between two dates (local time), `later - earlier`. */
export function calendarDaysBetween(earlier: Date, later: Date): number {
  return Math.round((startOfDay(later) - startOfDay(earlier)) / DAY);
}

/** Short relative time for lists: "now", "5m", "3h", "2d", "12 Mar". */
export function formatRelativeTime(iso: string, now: Date = new Date()): string {
  const date = parse(iso);
  if (!date) return "";
  const diff = now.getTime() - date.getTime();
  if (diff < MINUTE) return "now";
  if (diff < HOUR) return `${Math.floor(diff / MINUTE)}m`;
  if (diff < DAY) return `${Math.floor(diff / HOUR)}h`;
  if (diff < 7 * DAY) return `${Math.max(1, Math.floor(diff / DAY))}d`;
  const sameYear = date.getFullYear() === now.getFullYear();
  const label = `${date.getDate()} ${MONTHS[date.getMonth()]}`;
  return sameYear ? label : `${label} ${date.getFullYear()}`;
}

/** 24h clock label "HH:mm" in local time. */
export function formatClock(iso: string): string {
  const date = parse(iso);
  if (!date) return "";
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}`;
}

/** Label for a date separator in a thread: "Today", "Yesterday", "Monday", "12 Mar". */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = parse(iso);
  if (!date) return "";
  const days = calendarDaysBetween(date, now);
  if (days <= 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 7) return WEEKDAYS[date.getDay()];
  const label = `${date.getDate()} ${MONTHS[date.getMonth()]}`;
  return date.getFullYear() === now.getFullYear() ? label : `${label} ${date.getFullYear()}`;
}

/** Stable key identifying the local calendar day of a timestamp. */
export function dayKey(iso: string): string {
  const date = parse(iso);
  if (!date) return "unknown";
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`;
}

/** "about 5 km away", or null when the distance is hidden or unknown. */
export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined || !Number.isFinite(km) || km < 0) return null;
  const rounded = Math.max(1, Math.round(km));
  return `about ${rounded} km away`;
}

/** "Alice, 29" or "Alice" when the age is hidden. */
export function formatNameAge(name: string, age: number | null | undefined): string {
  return age === null || age === undefined ? name : `${name}, ${age}`;
}

/** "Location line": city and distance joined, skipping missing parts. */
export function formatPlace(city: string, km: number | null | undefined): string {
  const parts: string[] = [];
  if (city.trim()) parts.push(city.trim());
  const distance = formatDistance(km);
  if (distance) parts.push(distance);
  return parts.join(" · ");
}

/** Turn an interest slug ("hiking_trips") into a display label ("Hiking trips"). */
export function humanizeSlug(slug: string): string {
  const text = slug.replace(/[-_]+/g, " ").trim();
  if (!text) return "";
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/** One-line preview of a message body. */
export function previewText(body: string, maxLength = 80): string {
  const single = body.replace(/\s+/g, " ").trim();
  return single.length > maxLength ? `${single.slice(0, maxLength - 1).trimEnd()}…` : single;
}
