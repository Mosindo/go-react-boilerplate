/** Parses "JJ/MM/AAAA" into an ISO date (AAAA-MM-JJ), or null when it is not a real date. */
export function parseBirthDate(input: string): string | null {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) {
    return null;
  }
  const day = Number(match[1]);
  const month = Number(match[2]);
  const year = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return null;
  }
  return `${match[3]}-${match[2]}-${match[1]}`;
}

export function ageFromIso(iso: string, now: Date = new Date()): number {
  const [year, month, day] = iso.split("-").map(Number);
  let age = now.getUTCFullYear() - year;
  if (now.getUTCMonth() + 1 < month || (now.getUTCMonth() + 1 === month && now.getUTCDate() < day)) {
    age -= 1;
  }
  return age;
}

/** Inserts the slashes while the user types digits: "12031990" -> "12/03/1990". */
export function formatBirthDateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  const parts = [digits.slice(0, 2), digits.slice(2, 4), digits.slice(4, 8)].filter(Boolean);
  return parts.join("/");
}

export function isoToDisplay(iso: string): string {
  const [year, month, day] = iso.split("-");
  return `${day}/${month}/${year}`;
}
