const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });
const weekdayFormatter = new Intl.DateTimeFormat("fr-FR", { weekday: "long" });

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

/** Short timestamp for lists: "14:05", "Hier", "lundi", "3 mars". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) {
    return timeFormatter.format(date);
  }
  if (days === 1) {
    return "Hier";
  }
  if (days < 7) {
    return weekdayFormatter.format(date);
  }
  return dayFormatter.format(date);
}

/** Time shown under a chat bubble. */
export function formatMessageTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

/** Day separator in a conversation: "Aujourd'hui", "Hier", "3 mars". */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) {
    return "Aujourd'hui";
  }
  if (days === 1) {
    return "Hier";
  }
  return dayFormatter.format(date);
}

export function isSameDay(a: string, b: string): boolean {
  return startOfDay(new Date(a)) === startOfDay(new Date(b));
}

/** Approximate distance label; the API already rounds distances. */
export function formatDistance(km: number | null): string | null {
  if (km === null) {
    return null;
  }
  if (km <= 1) {
    return "À moins de 2 km";
  }
  return `À ${km} km`;
}

/** Relative time for notifications: "à l'instant", "il y a 5 min"... */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const seconds = Math.max(0, Math.round((now.getTime() - new Date(iso).getTime()) / 1000));
  if (seconds < 60) {
    return "à l'instant";
  }
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `il y a ${minutes} min`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 24) {
    return `il y a ${hours} h`;
  }
  const days = Math.round(hours / 24);
  if (days < 7) {
    return `il y a ${days} j`;
  }
  return dayFormatter.format(new Date(iso));
}
