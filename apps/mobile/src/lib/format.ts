const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Short, human relative time: "à l'instant", "5 min", "3 h", "hier", "12 mars". */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diff = now.getTime() - date.getTime();
  if (diff < MINUTE) {
    return "à l'instant";
  }
  if (diff < HOUR) {
    return `${Math.floor(diff / MINUTE)} min`;
  }
  if (diff < DAY) {
    return `${Math.floor(diff / HOUR)} h`;
  }
  if (diff < 2 * DAY) {
    return "hier";
  }
  return date.toLocaleDateString("fr-FR", { day: "numeric", month: "short" });
}

export function formatClock(iso: string): string {
  return new Date(iso).toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
}

/** Day separator label used in chat: "Aujourd'hui", "Hier" or "12 mars". */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(date)) / DAY);
  if (days === 0) {
    return "Aujourd'hui";
  }
  if (days === 1) {
    return "Hier";
  }
  return date.toLocaleDateString("fr-FR", { day: "numeric", month: "long" });
}

export function isSameDay(a: string, b: string): boolean {
  const da = new Date(a);
  const db = new Date(b);
  return (
    da.getFullYear() === db.getFullYear() &&
    da.getMonth() === db.getMonth() &&
    da.getDate() === db.getDate()
  );
}

export function formatDistance(km: number | null): string | null {
  if (km === null) {
    return null;
  }
  return km <= 1 ? "à moins de 1 km" : `à ${km} km`;
}

export const GENDER_LABELS = {
  man: "Homme",
  woman: "Femme",
  non_binary: "Non-binaire"
} as const;
