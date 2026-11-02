// Pure formatting helpers (no imports) — unit-tested with `node --test`.

export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  return `À moins de ${km} km`;
}

function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

function pad(n: number): string {
  return String(n).padStart(2, "0");
}

/** "14:05" today, "Hier", weekday within a week, otherwise "03/11". */
export function formatActivity(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  if (sameDay(date, now)) return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) return "Hier";
  const days = Math.floor((now.getTime() - date.getTime()) / 86_400_000);
  if (days < 7) {
    return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][date.getDay()] ?? "";
  }
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}

export function formatClock(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function pluralize(count: number, one: string, many: string): string {
  return count > 1 ? many : one;
}
