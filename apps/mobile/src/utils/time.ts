const pad = (n: number): string => String(n).padStart(2, "0");

/** HH:mm in local time. */
export function formatHour(iso: string): string {
  const d = new Date(iso);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

/** Short label for conversation lists: hour today, "Hier", weekday or dd/mm. */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (days <= 0) return formatHour(iso);
  if (days === 1) return "Hier";
  if (days < 7) return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][d.getDay()] ?? "";
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}`;
}

export function relativeTime(iso: string, now: Date = new Date()): string {
  const seconds = Math.max(0, Math.round((now.getTime() - new Date(iso).getTime()) / 1000));
  if (seconds < 60) return "à l'instant";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `il y a ${minutes} min`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `il y a ${hours} h`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `il y a ${days} j`;
  return formatListTime(iso, now);
}
