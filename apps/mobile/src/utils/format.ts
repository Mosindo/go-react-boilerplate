const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });

export function formatTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

/** "à l'instant", "14:05", "hier", "3 mars". */
export function formatRelative(iso: string, now = new Date()): string {
  const date = new Date(iso);
  const diffMs = now.getTime() - date.getTime();
  if (diffMs < 60_000) {
    return "à l'instant";
  }
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) {
    return timeFormatter.format(date);
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) {
    return "hier";
  }
  return dayFormatter.format(date);
}

export function formatDistance(km: number | null): string | null {
  if (km === null) {
    return null;
  }
  return km <= 1 ? "à moins d'1 km" : `à ${km} km`;
}

/** Builds an ISO date from day/month/year inputs; null when invalid. */
export function toIsoDate(day: string, month: string, year: string): string | null {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || y < 1900) {
    return null;
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return null;
  }
  return `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

export function ageFromIso(iso: string, today = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = today.getFullYear() - y;
  if (today.getMonth() + 1 < m || (today.getMonth() + 1 === m && today.getDate() < d)) {
    age -= 1;
  }
  return age;
}
