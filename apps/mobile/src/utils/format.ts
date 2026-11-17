// Pure formatting helpers (no React Native imports) — covered by node unit tests.

export type GenderValue = "woman" | "man" | "nonbinary";

export const GENDER_LABELS: Record<GenderValue, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const INTERESTED_LABELS: Record<GenderValue, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export function distanceLabel(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  if (km <= 1) return "À moins de 1 km";
  return `À ${km} km`;
}

/** Parses "JJ/MM/AAAA" into an ISO date string, or returns null when invalid. */
export function parseFrenchDate(input: string): string | null {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) return null;
  const [, dd, mm, yyyy] = match;
  const day = Number(dd);
  const month = Number(mm);
  const year = Number(yyyy);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) return null;
  return `${yyyy}-${mm}-${dd}`;
}

export function isoToFrenchDate(iso: string | null | undefined): string {
  if (!iso) return "";
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : "";
}

/** Inserts slashes while typing a birthdate: "1203199" -> "12/03/199". */
export function maskFrenchDate(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) return digits;
  if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

export function ageFromIso(iso: string, now: Date = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) age -= 1;
  return age;
}

function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

const pad = (n: number) => String(n).padStart(2, "0");

/** Short timestamp for lists: "14:05", "Hier", "lun.", "12/03". */
export function shortTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (sameDay(date, now)) return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) return "Hier";
  const diffDays = (now.getTime() - date.getTime()) / 86_400_000;
  if (diffDays < 7) return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][date.getDay()];
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}

export function clockTime(iso: string): string {
  const date = new Date(iso);
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** Day separator label in the chat: "Aujourd'hui", "Hier", "12/03/2026". */
export function dayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (sameDay(date, now)) return "Aujourd'hui";
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) return "Hier";
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}/${date.getFullYear()}`;
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/.test(email.trim());
}

/** Mirrors the server password policy so users get instant feedback. */
export function passwordProblem(password: string): string | null {
  if (password.length < 8) return "8 caractères minimum.";
  if (password.length > 72) return "72 caractères maximum.";
  if (!/\p{L}/u.test(password) || !/\d/.test(password)) return "Au moins une lettre et un chiffre.";
  return null;
}
