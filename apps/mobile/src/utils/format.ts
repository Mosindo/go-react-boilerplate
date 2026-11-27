const DAYS = ["dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"];
const MONTHS = ["janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."];

const pad = (n: number) => String(n).padStart(2, "0");

export function formatTime(iso: string): string {
  const d = new Date(iso);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

export function daysBetween(a: Date, b: Date): number {
  return Math.round((startOfDay(b) - startOfDay(a)) / 86_400_000);
}

/** Short relative label for lists: "14:05", "Hier", "lundi", "12 mars". */
export function formatRelativeShort(iso: string, now = new Date()): string {
  const d = new Date(iso);
  const diff = daysBetween(d, now);
  if (diff <= 0) {
    return formatTime(iso);
  }
  if (diff === 1) {
    return "Hier";
  }
  if (diff < 7) {
    return DAYS[d.getDay()];
  }
  const label = `${d.getDate()} ${MONTHS[d.getMonth()]}`;
  return d.getFullYear() === now.getFullYear() ? label : `${label} ${d.getFullYear()}`;
}

/** Day separator label for conversations. */
export function formatDayLabel(iso: string, now = new Date()): string {
  const d = new Date(iso);
  const diff = daysBetween(d, now);
  if (diff <= 0) {
    return "Aujourd'hui";
  }
  if (diff === 1) {
    return "Hier";
  }
  const label = `${DAYS[d.getDay()]} ${d.getDate()} ${MONTHS[d.getMonth()]}`;
  return d.getFullYear() === now.getFullYear() ? label : `${label} ${d.getFullYear()}`;
}

export function formatDistance(km?: number): string | null {
  if (km === undefined || km === null) {
    return null;
  }
  return km <= 1 ? "À moins de 2 km" : `À ${km} km`;
}

/** Converts DD/MM/YYYY parts to an ISO date, or null when invalid. */
export function toIsoDate(day: string, month: string, year: string): string | null {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || y < 1900 || m < 1 || m > 12 || d < 1) {
    return null;
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCDate() !== d || date.getUTCMonth() !== m - 1) {
    return null;
  }
  return `${y}-${pad(m)}-${pad(d)}`;
}

export function ageFromIso(iso: string, now = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) {
    age--;
  }
  return age;
}

let counter = 0;
/** Locally unique id used for idempotent message sends. */
export function clientId(): string {
  counter = (counter + 1) % 1_000_000;
  return `${Date.now().toString(36)}-${counter.toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@<>()[\],;:"]+@[^\s@<>()[\],;:"]+\.[^\s@<>()[\],;:"]+$/.test(email.trim());
}

export function passwordIssue(password: string): string | null {
  if (password.length < 8) {
    return "8 caractères minimum.";
  }
  if (!/[A-Za-zÀ-ÿ]/.test(password) || !/[0-9]/.test(password)) {
    return "Au moins une lettre et un chiffre.";
  }
  if (password.length > 72) {
    return "72 caractères maximum.";
  }
  return null;
}
