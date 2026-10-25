const CLOCK_FORMAT = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const DAY_MONTH_FORMAT = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "long" });
const DAY_MONTH_YEAR_FORMAT = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "long", year: "numeric" });
const SHORT_DATE_FORMAT = new Intl.DateTimeFormat("fr-FR", { day: "2-digit", month: "2-digit" });

/** Local calendar day, used to group messages under day separators. */
export function dayKey(iso: string): string {
  const date = new Date(iso);
  return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;
}

export function formatClock(iso: string): string {
  return CLOCK_FORMAT.format(new Date(iso));
}

function dayDifference(iso: string, now: Date): number {
  const date = new Date(iso);
  const startOfDay = (value: Date) => new Date(value.getFullYear(), value.getMonth(), value.getDate()).getTime();
  return Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
}

export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const difference = dayDifference(iso, now);
  if (difference === 0) return "Aujourd'hui";
  if (difference === 1) return "Hier";
  const date = new Date(iso);
  return date.getFullYear() === now.getFullYear() ? DAY_MONTH_FORMAT.format(date) : DAY_MONTH_YEAR_FORMAT.format(date);
}

/** Compact timestamp for conversation and notification lists. */
export function formatListTimestamp(iso: string, now: Date = new Date()): string {
  const difference = dayDifference(iso, now);
  if (difference === 0) return formatClock(iso);
  if (difference === 1) return "Hier";
  return SHORT_DATE_FORMAT.format(new Date(iso));
}

export function formatDistance(distanceKm: number | null): string | null {
  return distanceKm === null ? null : `à ${distanceKm} km`;
}
