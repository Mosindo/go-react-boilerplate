/** "~5 km away" — the API already rounds distances up to 5 km steps. */
export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  return km <= 5 ? "Within 5 km" : `About ${km} km away`;
}

export function formatClock(iso: string, locale?: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString(locale, { hour: "numeric", minute: "2-digit" });
}

/** Short relative stamp for lists: "now", "5m", "3h", "Yesterday", or a date. */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diffMs = now.getTime() - date.getTime();
  if (Number.isNaN(diffMs)) return "";
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "now";
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  if (days === 1) return "Yesterday";
  if (days < 7) return `${days}d`;
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const diffDays = Math.round((startOf(now) - startOf(date)) / 86_400_000);
  if (diffDays === 0) return "Today";
  if (diffDays === 1) return "Yesterday";
  return date.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" });
}

export function initialOf(name: string): string {
  return name.trim().charAt(0).toUpperCase() || "?";
}

let counter = 0;
/** Client-side id for optimistic entries. */
export function localId(prefix = "local"): string {
  counter += 1;
  return `${prefix}-${Date.now().toString(36)}-${counter}`;
}
