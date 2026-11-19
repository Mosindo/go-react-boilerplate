import type { Gender, LookingFor } from "../api/types";

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

export const lookingForLabels: Record<LookingFor, string> = {
  relationship: "Une relation sérieuse",
  casual: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  unsure: "Je verrai bien"
};

export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  if (km <= 1) return "À moins de 2 km";
  return `À environ ${km} km`;
}

const pad = (n: number) => String(n).padStart(2, "0");

export function formatTime(iso: string): string {
  const d = new Date(iso);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** Short relative label used in lists: "14:05", "Hier", "lun.", "12/03". */
export function formatListDate(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const day = 24 * 60 * 60 * 1000;
  if (d.getTime() >= startOfToday) return formatTime(iso);
  if (d.getTime() >= startOfToday - day) return "Hier";
  if (d.getTime() >= startOfToday - 6 * day) {
    return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][d.getDay()];
  }
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}`;
}

/** Day separator label in a conversation. */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  const sameDay = (a: Date, b: Date) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  if (sameDay(d, now)) return "Aujourd'hui";
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(d, yesterday)) return "Hier";
  const months = ["janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."];
  const base = `${d.getDate()} ${months[d.getMonth()]}`;
  return d.getFullYear() === now.getFullYear() ? base : `${base} ${d.getFullYear()}`;
}
