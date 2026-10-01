/** Parses a "JJ", "MM", "AAAA" triple into an ISO date, or null when it is not a real calendar date. */
export function toIsoDate(day: string, month: string, year: string): string | null {
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) return null;
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) return null;
  return `${year}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

export function ageOn(isoDate: string, now: Date = new Date()): number {
  const [y, m, d] = isoDate.split("-").map(Number) as [number, number, number];
  let age = now.getUTCFullYear() - y;
  if (now.getUTCMonth() + 1 < m || (now.getUTCMonth() + 1 === m && now.getUTCDate() < d)) age -= 1;
  return age;
}

export function splitIsoDate(iso: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day: day.replace(/^0/, ""), month: month.replace(/^0/, ""), year };
}

/** "14:32" for today, "hier", weekday for this week, otherwise "12/03". */
export function formatMessageTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  const diffDays = Math.floor((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (diffDays === 1) return "hier";
  if (diffDays > 1 && diffDays < 7) return date.toLocaleDateString("fr-FR", { weekday: "short" });
  return date.toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" });
}

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}
