const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

/** 14:05 */
export function formatClock(iso: string): string {
  const date = new Date(iso);
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** "À l'instant", "5 min", "3 h", "Hier", then dd/mm. Used in lists. */
export function formatRelativeTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diff = now.getTime() - date.getTime();
  if (diff < MINUTE) {
    return "À l'instant";
  }
  if (diff < HOUR) {
    return `${Math.floor(diff / MINUTE)} min`;
  }
  if (diff < DAY && date.getDate() === now.getDate()) {
    return `${Math.floor(diff / HOUR)} h`;
  }
  const yesterday = new Date(now.getTime() - DAY);
  if (date.getDate() === yesterday.getDate() && diff < 2 * DAY) {
    return "Hier";
  }
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}

/** Separator label between message groups: "Aujourd'hui", "Hier", "12/03/2026". */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const sameDay = (a: Date, b: Date) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  if (sameDay(date, now)) {
    return "Aujourd'hui";
  }
  if (sameDay(date, new Date(now.getTime() - DAY))) {
    return "Hier";
  }
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}/${date.getFullYear()}`;
}

/** Distances are already coarse server-side; this only words them. */
export function formatDistance(km: number | null | undefined): string {
  if (km === null || km === undefined) {
    return "";
  }
  return km <= 1 ? "À moins de 1 km" : `À ${km} km`;
}

export function initialOf(name: string): string {
  return name.trim().slice(0, 1).toUpperCase();
}
