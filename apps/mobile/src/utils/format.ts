import type { Gender, RelationshipGoal } from "../api/types";

export const genderLabels: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const seekingLabels: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export const goalLabels: Record<RelationshipGoal, string> = {
  long_term: "Une relation sérieuse",
  short_term: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  undecided: "Je verrai bien"
};

export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) {
    return null;
  }
  if (km <= 1) {
    return "à moins de 1 km";
  }
  return `à ${km} km`;
}

const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { weekday: "long", day: "numeric", month: "long" });
const shortDayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

export function formatTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

/** Compact relative time for lists: "14:02", "hier", "12 mars". */
export function formatListTime(iso: string, now = new Date()): string {
  const date = new Date(iso);
  if (isSameDay(date, now)) {
    return timeFormatter.format(date);
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (isSameDay(date, yesterday)) {
    return "hier";
  }
  return shortDayFormatter.format(date);
}

export function formatDayLabel(iso: string, now = new Date()): string {
  const date = new Date(iso);
  if (isSameDay(date, now)) {
    return "Aujourd'hui";
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (isSameDay(date, yesterday)) {
    return "Hier";
  }
  const label = dayFormatter.format(date);
  return label.charAt(0).toUpperCase() + label.slice(1);
}

/** Age in full years from a YYYY-MM-DD birthdate. */
export function ageFromBirthdate(birthdate: string, now = new Date()): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(birthdate);
  if (!match) {
    return null;
  }
  const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])];
  const date = new Date(year, month - 1, day);
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null;
  }
  let age = now.getFullYear() - year;
  if (now.getMonth() < month - 1 || (now.getMonth() === month - 1 && now.getDate() < day)) {
    age -= 1;
  }
  return age;
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
}

export function passwordProblem(password: string): string | null {
  if (password.length < 8) {
    return "8 caractères minimum.";
  }
  if (!/[A-Za-zÀ-ÿ]/.test(password) || !/\d/.test(password)) {
    return "Ajoutez au moins une lettre et un chiffre.";
  }
  return null;
}
