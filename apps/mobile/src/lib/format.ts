const pad = (n: number) => String(n).padStart(2, "0");

/** Converts "JJ/MM/AAAA" (also accepts "-" or "." separators) to an ISO "AAAA-MM-JJ" date, or null. */
export function parseBirthDate(input: string): string | null {
  const match = /^(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})$/.exec(input.trim());
  if (!match) return null;
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (
    date.getUTCFullYear() !== year ||
    date.getUTCMonth() !== month - 1 ||
    date.getUTCDate() !== day
  ) {
    return null;
  }
  return `${year}-${pad(month)}-${pad(day)}`;
}

/** ISO "AAAA-MM-JJ" to display "JJ/MM/AAAA". */
export function displayBirthDate(iso: string): string {
  const [year, month, day] = iso.split("-");
  return year && month && day ? `${day}/${month}/${year}` : iso;
}

/** Age in whole years for an ISO birth date as of `now`. */
export function ageFromISO(iso: string, now: Date = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  if (!y || !m || !d) return 0;
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) age -= 1;
  return age;
}

export function initials(name: string): string {
  return name.trim().charAt(0).toUpperCase() || "?";
}

/** "14:05" for today, "Hier", else "12/03". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const sameDay = (a: Date, b: Date) =>
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate();
  if (sameDay(date, now)) return formatClock(iso);
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) return "Hier";
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}

export function formatClock(iso: string): string {
  const date = new Date(iso);
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** Label for a day divider inside a conversation. */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const label = formatListTime(iso, now);
  if (label.includes(":")) return "Aujourd'hui";
  return label === "Hier" ? label : `${label}/${new Date(iso).getFullYear()}`;
}

export function dayKey(iso: string): string {
  const d = new Date(iso);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}
