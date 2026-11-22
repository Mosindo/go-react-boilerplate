import type { Gender } from "../api/types";

export const genderLabels: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const interestedInLabels: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export function formatDistance(km?: number): string | null {
  if (km === undefined || km === null) {
    return null;
  }
  return km <= 1 ? "À moins de 2 km" : `À ${km} km`;
}

const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });
const weekdayFormatter = new Intl.DateTimeFormat("fr-FR", { weekday: "long" });

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

/** "14:05", "Hier", "lundi" or "3 mars", relative to now. */
export function formatListTime(iso: string, now = new Date()): string {
  const d = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (days <= 0) {
    return timeFormatter.format(d);
  }
  if (days === 1) {
    return "Hier";
  }
  if (days < 7) {
    return weekdayFormatter.format(d);
  }
  return dayFormatter.format(d);
}

export function formatMessageTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

/** Day separator label for the chat thread. */
export function formatDayLabel(iso: string, now = new Date()): string {
  const d = new Date(iso);
  const days = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (days <= 0) {
    return "Aujourd'hui";
  }
  if (days === 1) {
    return "Hier";
  }
  return new Intl.DateTimeFormat("fr-FR", { weekday: "long", day: "numeric", month: "long" }).format(d);
}

export function sameDay(a: string, b: string): boolean {
  return startOfDay(new Date(a)) === startOfDay(new Date(b));
}
