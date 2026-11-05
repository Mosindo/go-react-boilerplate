export type ParsedBirthDate = { iso: string; age: number } | null;

/** Parses "JJ/MM/AAAA" (what users type) into ISO "AAAA-MM-JJ" plus the age on `now`. */
export function parseBirthDate(input: string, now: Date = new Date()): ParsedBirthDate {
  const m = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!m) return null;
  const day = Number(m[1]);
  const month = Number(m[2]);
  const year = Number(m[3]);
  const d = new Date(Date.UTC(year, month - 1, day));
  if (d.getUTCFullYear() !== year || d.getUTCMonth() !== month - 1 || d.getUTCDate() !== day) {
    return null;
  }
  let age = now.getUTCFullYear() - year;
  const beforeBirthday =
    now.getUTCMonth() + 1 < month || (now.getUTCMonth() + 1 === month && now.getUTCDate() < day);
  if (beforeBirthday) age -= 1;
  const pad = (n: number, w = 2) => String(n).padStart(w, "0");
  return { iso: `${pad(year, 4)}-${pad(month)}-${pad(day)}`, age };
}

/** Formats ISO "AAAA-MM-JJ" back to "JJ/MM/AAAA". */
export function isoToDisplay(iso: string): string {
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : "";
}

/** Short, human timestamp: "14:05", "hier", "mar.", "12 mars". */
export function formatTimestamp(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) {
    return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  }
  if (days === 1) return "hier";
  if (days < 7) return date.toLocaleDateString("fr-FR", { weekday: "short" });
  return date.toLocaleDateString("fr-FR", { day: "numeric", month: "short" });
}
