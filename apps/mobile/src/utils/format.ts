const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { weekday: "long", day: "numeric", month: "long" });
const shortDayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

export function formatTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

/** "Aujourd'hui", "Hier" or a long date: used for chat day separators. */
export function formatDayLabel(iso: string, now = new Date()): string {
  const diffDays = Math.round((startOfDay(now) - startOfDay(new Date(iso))) / 86_400_000);
  if (diffDays === 0) return "Aujourd'hui";
  if (diffDays === 1) return "Hier";
  const label = dayFormatter.format(new Date(iso));
  return label.charAt(0).toUpperCase() + label.slice(1);
}

/** Compact relative time for lists: "à l'instant", "5 min", "14:02", "hier", "3 mars". */
export function formatRelative(iso: string, now = new Date()): string {
  const date = new Date(iso);
  const seconds = Math.max(0, Math.round((now.getTime() - date.getTime()) / 1000));
  if (seconds < 60) return "à l'instant";
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min`;
  const diffDays = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (diffDays === 0) return timeFormatter.format(date);
  if (diffDays === 1) return "hier";
  return shortDayFormatter.format(date);
}

export function formatDistance(km: number | null): string | null {
  if (km === null) return null;
  return km <= 1 ? "à moins d'1 km" : `à ${km} km`;
}

export const genderLabels: Record<string, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const genderPluralLabels: Record<string, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export function sameDay(a: string, b: string): boolean {
  return startOfDay(new Date(a)) === startOfDay(new Date(b));
}
