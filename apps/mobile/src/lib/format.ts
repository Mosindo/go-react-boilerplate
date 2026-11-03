/** Returns the age in years on `today`, or null for an invalid YYYY-MM-DD string. */
export function ageFromBirthDate(value: string, today: Date = new Date()): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim());
  if (!match) {
    return null;
  }
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
    return null;
  }
  let age = today.getUTCFullYear() - year;
  const beforeBirthday = today.getUTCMonth() + 1 < month || (today.getUTCMonth() + 1 === month && today.getUTCDate() < day);
  if (beforeBirthday) {
    age -= 1;
  }
  return age;
}

/** Progressive "AAAAMMJJ" -> "AAAA-MM-JJ" formatter for the birth date field. */
export function formatBirthDateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 4) {
    return digits;
  }
  if (digits.length <= 6) {
    return `${digits.slice(0, 4)}-${digits.slice(4)}`;
  }
  return `${digits.slice(0, 4)}-${digits.slice(4, 6)}-${digits.slice(6)}`;
}

export function validateBirthDate(value: string): string | null {
  const age = ageFromBirthDate(value);
  if (age === null) {
    return "Saisissez une date valide (AAAA-MM-JJ).";
  }
  if (age < 18) {
    return "Vous devez avoir au moins 18 ans.";
  }
  if (age > 120) {
    return "Cette date de naissance n'est pas valide.";
  }
  return null;
}

export function formatTime(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
}

/** Short relative label: today -> hour, this week -> weekday, otherwise date. */
export function formatActivity(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const dayMs = 24 * 60 * 60 * 1000;
  const diff = now.getTime() - date.getTime();
  if (date.toDateString() === now.toDateString()) {
    return formatTime(iso);
  }
  if (diff < 6 * dayMs) {
    return date.toLocaleDateString("fr-FR", { weekday: "short" });
  }
  return date.toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" });
}

export function distanceLabel(distanceKm: number | null): string | null {
  return distanceKm === null ? null : `À moins de ${distanceKm} km`;
}

export const DISTANCE_OPTIONS = [10, 25, 50, 100, 250, 500] as const;

export function distanceOptionLabel(km: number): string {
  return km >= 500 ? "Partout" : `${km} km`;
}
