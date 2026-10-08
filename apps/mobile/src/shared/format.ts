const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** "14:05" in the user's locale. */
export function formatClock(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
}

/** Compact relative time for lists: "à l'instant", "5 min", "3 h", "hier", "12 mars". */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const diff = now.getTime() - date.getTime();
  if (diff < MINUTE) return "à l'instant";
  if (diff < HOUR) return `${Math.floor(diff / MINUTE)} min`;
  if (diff < DAY) return `${Math.floor(diff / HOUR)} h`;
  if (diff < 2 * DAY) return "hier";
  return date.toLocaleDateString("fr-FR", { day: "numeric", month: "short" });
}

/** Day separators in a conversation: "Aujourd'hui", "Hier" or "12 mars 2026". */
export function formatDay(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(date)) / DAY);
  if (days === 0) return "Aujourd'hui";
  if (days === 1) return "Hier";
  return date.toLocaleDateString("fr-FR", { day: "numeric", month: "long", year: "numeric" });
}

export function sameDay(a: string, b: string): boolean {
  const x = new Date(a);
  const y = new Date(b);
  return x.getFullYear() === y.getFullYear() && x.getMonth() === y.getMonth() && x.getDate() === y.getDate();
}

export function formatDistance(km: number | null): string | null {
  if (km === null) return null;
  return km <= 5 ? "À moins de 5 km" : `À ${km} km`;
}

export const GENDER_LABELS = { man: "Homme", woman: "Femme", non_binary: "Non-binaire" } as const;
export const LOOKING_FOR_LABELS = {
  relationship: "Une relation",
  casual: "Quelque chose de léger",
  friends: "Des amis",
  open: "Ouvert(e) à tout"
} as const;
