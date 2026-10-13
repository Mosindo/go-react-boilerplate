import type { Gender } from "../api/types";

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

export const MIN_AGE = 18;
export const MAX_AGE = 99;

/** "à 10 km" style label for the coarse distance sent by the API. */
export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) {
    return null;
  }
  return km <= 5 ? "À moins de 5 km" : `À ${km} km`;
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

/** Parses a user-typed "JJ/MM/AAAA" date into "AAAA-MM-JJ", or null when it is not a real date. */
export function parseBirthDateInput(input: string): string | null {
  const match = /^(\d{1,2})[/.\-\s](\d{1,2})[/.\-\s](\d{4})$/.exec(input.trim());
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
  return `${year}-${pad(month)}-${pad(day)}`;
}

export function formatBirthDate(iso: string): string {
  const [year, month, day] = iso.split("-");
  return year && month && day ? `${day}/${month}/${year}` : iso;
}

export function ageOn(isoBirthDate: string, today: Date): number {
  const [y, m, d] = isoBirthDate.split("-").map(Number);
  let age = today.getFullYear() - y;
  if (today.getMonth() + 1 < m || (today.getMonth() + 1 === m && today.getDate() < d)) {
    age -= 1;
  }
  return age;
}

export function isAdult(isoBirthDate: string, today: Date = new Date()): boolean {
  return ageOn(isoBirthDate, today) >= MIN_AGE;
}

export function formatClock(iso: string): string {
  const d = new Date(iso);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** Compact relative time for lists: "14:05", "Hier", "lun.", "12/03". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (days <= 0) {
    return formatClock(iso);
  }
  if (days === 1) {
    return "Hier";
  }
  if (days < 7) {
    return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][d.getDay()] ?? "";
  }
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}`;
}

/** Day separator label inside a conversation. */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (days <= 0) {
    return "Aujourd'hui";
  }
  if (days === 1) {
    return "Hier";
  }
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()}`;
}

export function sameDay(a: string, b: string): boolean {
  const x = new Date(a);
  const y = new Date(b);
  return x.getFullYear() === y.getFullYear() && x.getMonth() === y.getMonth() && x.getDate() === y.getDate();
}
