const pad = (n: number) => String(n).padStart(2, "0");

const MONTHS = ["janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."];

/** "14:05" in the device time zone. */
export function formatClock(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

/** Compact relative time for lists: "À l'instant", "14:05", "Hier", "12 mars". */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const diffMin = Math.floor((now.getTime() - d.getTime()) / 60_000);
  if (diffMin < 1) return "À l'instant";
  if (diffMin < 60) return `il y a ${diffMin} min`;
  const dayDiff = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (dayDiff === 0) return formatClock(iso);
  if (dayDiff === 1) return "Hier";
  if (dayDiff < 7) return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][d.getDay()] ?? "";
  const month = MONTHS[d.getMonth()] ?? "";
  return d.getFullYear() === now.getFullYear() ? `${d.getDate()} ${month}` : `${d.getDate()} ${month} ${d.getFullYear()}`;
}

/** "à 3 km" — the server already coarsens the value, we only word it. */
export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  return km <= 1 ? "à moins de 1 km" : `à ${km} km`;
}
