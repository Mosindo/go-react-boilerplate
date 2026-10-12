const timeFormat = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormat = new Intl.DateTimeFormat("fr-FR", { weekday: "long", day: "numeric", month: "long" });

export function formatTime(iso: string): string {
  return timeFormat.format(new Date(iso));
}

export function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (sameDay(date, now)) {
    return "Aujourd'hui";
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) {
    return "Hier";
  }
  return dayFormat.format(date);
}

/** Short relative label used in conversation lists. */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (sameDay(date, now)) {
    return formatTime(iso);
  }
  const diffDays = Math.floor((now.getTime() - date.getTime()) / 86_400_000);
  if (diffDays < 7) {
    return new Intl.DateTimeFormat("fr-FR", { weekday: "short" }).format(date);
  }
  return new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" }).format(date);
}

/** Distances are already bucketed by the API; never display a falsely precise value. */
export function formatDistance(km: number | null): string | null {
  if (km === null) {
    return null;
  }
  if (km >= 1000) {
    return "Très loin";
  }
  return km <= 1 ? "À moins de 1 km" : `À moins de ${km} km`;
}

export function isSameMinuteGroup(a: string, b: string): boolean {
  return Math.abs(new Date(a).getTime() - new Date(b).getTime()) < 5 * 60_000;
}
