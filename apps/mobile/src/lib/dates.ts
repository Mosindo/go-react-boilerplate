const pad = (n: number): string => String(n).padStart(2, "0");

/** Formats "JJ/MM/AAAA" input while typing, keeping digits only. */
export function formatBirthInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  const parts = [digits.slice(0, 2), digits.slice(2, 4), digits.slice(4, 8)].filter(Boolean);
  return parts.join("/");
}

/** Converts "JJ/MM/AAAA" to ISO "AAAA-MM-JJ", or null if it is not a real calendar date. */
export function parseBirthInput(input: string): string | null {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input);
  if (!match) return null;
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return null;
  }
  return `${year}-${pad(month)}-${pad(day)}`;
}

export function toBirthInput(iso: string): string {
  const [year = "", month = "", day = ""] = iso.split("-");
  return `${day}/${month}/${year}`;
}

export function ageFromIso(iso: string, now: Date = new Date()): number {
  const [year = 0, month = 1, day = 1] = iso.split("-").map(Number);
  let age = now.getFullYear() - year;
  if (now.getMonth() + 1 < month || (now.getMonth() + 1 === month && now.getDate() < day)) age -= 1;
  return age;
}

export function formatTime(iso: string): string {
  const d = new Date(iso);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** Short relative label for conversation lists: time today, "Hier", else date. */
export function formatListDate(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  const startOfDay = (x: Date): number => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const diffDays = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (diffDays <= 0) return formatTime(iso);
  if (diffDays === 1) return "Hier";
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}`;
}

export function formatDayHeader(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  const label = formatListDate(iso, now);
  if (label.includes(":")) return "Aujourd'hui";
  if (label === "Hier") return label;
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()}`;
}
