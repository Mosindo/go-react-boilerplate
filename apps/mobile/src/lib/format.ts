export function formatDistance(km?: number): string | null {
  if (km === undefined || km === null) return null;
  return km <= 1 ? "À moins de 1 km" : `À ${km} km`;
}

const MINUTE = 60_000;
const DAY = 24 * 60 * MINUTE;

/** Compact timestamp for conversation rows: "14:05", "Hier", "lun.", "12/03". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diff = now.getTime() - date.getTime();
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  const yesterday = new Date(now.getTime() - DAY);
  if (date.toDateString() === yesterday.toDateString()) return "Hier";
  if (diff < 6 * DAY) return date.toLocaleDateString("fr-FR", { weekday: "short" });
  return date.toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" });
}

export function formatMessageTime(iso: string): string {
  return new Date(iso).toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
}

export function formatDayHeader(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (date.toDateString() === now.toDateString()) return "Aujourd'hui";
  if (date.toDateString() === new Date(now.getTime() - DAY).toDateString()) return "Hier";
  return date.toLocaleDateString("fr-FR", { weekday: "long", day: "numeric", month: "long" });
}

export function formatRelative(iso: string, now: Date = new Date()): string {
  const diff = now.getTime() - new Date(iso).getTime();
  if (diff < MINUTE) return "à l'instant";
  if (diff < 60 * MINUTE) return `il y a ${Math.floor(diff / MINUTE)} min`;
  if (diff < DAY) return `il y a ${Math.floor(diff / (60 * MINUTE))} h`;
  return formatListTime(iso, now);
}
